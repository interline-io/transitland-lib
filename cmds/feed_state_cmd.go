package cmds

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/feedstate"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/spf13/pflag"
)

// FeedStateManagerCommand manages feed state and materialized index tables
type FeedStateManagerCommand struct {
	DBURL             string
	ActivateFVIDs     []string
	DeactivateFVIDs   []string
	SetActiveFVIDs    []string
	SetActiveFVIDFile string
	DryRun            bool
	Adapter           tldb.Adapter // allow for mocks
}

func (cmd *FeedStateManagerCommand) HelpDesc() (string, string) {
	return "Manage feed state and materialized tables", `This command manages feed state including which feed versions are active, and maintains materialized tables that cache active route, stop, and agency data for improved query performance. It provides centralized control over feed version activation across the entire system.`
}

func (cmd *FeedStateManagerCommand) HelpArgs() string {
	return ""
}

func (cmd *FeedStateManagerCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL")
	fl.StringSliceVar(&cmd.ActivateFVIDs, "activate", nil, "Activate these feed version IDs (deactivates other versions in same feeds)")
	fl.StringSliceVar(&cmd.DeactivateFVIDs, "deactivate", nil, "Deactivate these feed version IDs")
	fl.StringSliceVar(&cmd.SetActiveFVIDs, "set-active", nil, "Set ONLY these feed version IDs as active (deactivates all others)")
	fl.StringVar(&cmd.SetActiveFVIDFile, "set-active-fvid-file", "", "Set ONLY these feed version IDs as active from file (one per line)")
	fl.BoolVar(&cmd.DryRun, "dry-run", false, "Show what would be done without making changes")
}

// Parse command line flags
func (cmd *FeedStateManagerCommand) Parse(args []string) error {
	fl := pflag.NewFlagSet("feed-state", pflag.ContinueOnError)
	fl.Usage = func() {
		fmt.Println("Usage: feed-state")
		fl.PrintDefaults()
	}
	cmd.AddFlags(fl)
	if err := fl.Parse(args); err != nil {
		return err
	}

	// Process set-active-fvid-file if specified
	if cmd.SetActiveFVIDFile != "" {
		lines, err := tlcli.ReadFileLines(cmd.SetActiveFVIDFile)
		if err != nil {
			return err
		}
		for _, line := range lines {
			if line != "" {
				cmd.SetActiveFVIDs = append(cmd.SetActiveFVIDs, line)
			}
		}
		if len(cmd.SetActiveFVIDs) == 0 {
			return fmt.Errorf("--set-active-fvid-file specified but no lines were read")
		}
	}

	return nil
}

// Run the feed state management command
func (cmd *FeedStateManagerCommand) Run(ctx context.Context) error {
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}

	// Open database connection
	if cmd.Adapter == nil {
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		cmd.Adapter = writer.Adapter
		defer cmd.Adapter.Close()
	}

	manager := feedstate.NewManager(cmd.Adapter)
	start := time.Now()

	// Check for operations that modify feed state
	hasOperations := len(cmd.SetActiveFVIDs) > 0 || len(cmd.ActivateFVIDs) > 0 || len(cmd.DeactivateFVIDs) > 0

	if hasOperations {
		// Handle feed state operations
		return cmd.handleFeedStateOperations(ctx, manager, start)
	}

	cmd.logResults(ctx, manager, time.Since(start), "Feed state operations completed")
	return nil
}

// handleFeedStateOperations processes all feed state operations in a unified way
func (cmd *FeedStateManagerCommand) handleFeedStateOperations(ctx context.Context, manager *feedstate.Manager, start time.Time) error {
	// Parse all feed version IDs upfront
	var err error
	activateIDs, err := cmd.parseFeedVersionIDs(cmd.ActivateFVIDs)
	if err != nil {
		return fmt.Errorf("invalid activate IDs: %w", err)
	}

	deactivateIDs, err := cmd.parseFeedVersionIDs(cmd.DeactivateFVIDs)
	if err != nil {
		return fmt.Errorf("invalid deactivate IDs: %w", err)
	}

	setActiveIDs, err := cmd.parseFeedVersionIDs(cmd.SetActiveFVIDs)
	if err != nil {
		return fmt.Errorf("invalid set-active IDs: %w", err)
	}

	// Validate that set-active is not mixed with other operations
	if len(setActiveIDs) > 0 && (len(activateIDs) > 0 || len(deactivateIDs) > 0) {
		return fmt.Errorf("--set-active cannot be used with --activate or --deactivate")
	}

	// Handle dry run
	if cmd.DryRun {
		return cmd.showDryRunOperations(ctx, manager, activateIDs, deactivateIDs, setActiveIDs)
	}

	// Execute operations in transaction
	return cmd.Adapter.Tx(func(atx tldb.Adapter) error {
		txManager := feedstate.NewManager(atx)

		if len(setActiveIDs) > 0 {
			// Set-active: replace entire active set
			if err := txManager.SetActiveFeedVersions(ctx, setActiveIDs); err != nil {
				return fmt.Errorf("failed to set active feed versions: %w", err)
			}

			cmd.logResults(ctx, manager, time.Since(start), "Feed versions set as active")
		} else {
			// Activate/Deactivate: individual operations
			if len(deactivateIDs) > 0 {
				log.For(ctx).Info().Ints("deactivate", deactivateIDs).Msg("Deactivating feed versions")
				for _, fvid := range deactivateIDs {
					if err := txManager.DeactivateFeedVersion(ctx, fvid); err != nil {
						return fmt.Errorf("failed to deactivate feed version %d: %w", fvid, err)
					}
				}
			}

			if len(activateIDs) > 0 {
				log.For(ctx).Info().Ints("activate", activateIDs).Msg("Activating feed versions")
				for _, fvid := range activateIDs {
					if err := txManager.ActivateFeedVersion(ctx, fvid); err != nil {
						return fmt.Errorf("failed to activate feed version %d: %w", fvid, err)
					}
				}
			}

			if len(activateIDs) > 0 || len(deactivateIDs) > 0 {
				cmd.logResults(ctx, manager, time.Since(start), "Feed versions updated")
			}
		}

		return nil
	})

	return nil
}

// showDryRunOperations displays what would happen in dry run mode
func (cmd *FeedStateManagerCommand) showDryRunOperations(ctx context.Context, manager *feedstate.Manager, activateIDs, deactivateIDs, setActiveIDs []int) error {
	currentActive, err := manager.GetActiveFeedVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current active feed versions: %w", err)
	}

	fmt.Printf("DRY RUN: Feed State Operations\n")
	fmt.Printf("Current active feed versions: %v\n", currentActive)

	if len(setActiveIDs) > 0 {
		fmt.Printf("Would set active feed versions to: %v\n", setActiveIDs)
		fmt.Printf("All other feed versions would be deactivated\n")
	} else {
		if len(activateIDs) > 0 {
			fmt.Printf("Would activate: %v\n", activateIDs)
		}
		if len(deactivateIDs) > 0 {
			fmt.Printf("Would deactivate: %v\n", deactivateIDs)
		}
		if len(activateIDs) > 0 || len(deactivateIDs) > 0 {
			newActive := cmd.calculateNewActiveSet(currentActive, activateIDs, deactivateIDs)
			fmt.Printf("Resulting active feed versions: %v\n", newActive)
		}
	}

	return nil
}

// calculateNewActiveSet computes the new active set given current state and operations
func (cmd *FeedStateManagerCommand) calculateNewActiveSet(currentActive, activateIDs, deactivateIDs []int) []int {
	// Start with current active set
	activeSet := make(map[int]bool)
	for _, id := range currentActive {
		activeSet[id] = true
	}

	// Apply deactivations first
	for _, id := range deactivateIDs {
		delete(activeSet, id)
	}

	// Apply activations
	for _, id := range activateIDs {
		activeSet[id] = true
	}

	// Convert back to slice
	var result []int
	for id := range activeSet {
		result = append(result, id)
	}

	return result
}

// parseFeedVersionIDs converts string slice to int slice with validation
func (cmd *FeedStateManagerCommand) parseFeedVersionIDs(fvidStrings []string) ([]int, error) {
	var feedVersionIDs []int
	for _, fvidStr := range fvidStrings {
		fvid, err := strconv.Atoi(fvidStr)
		if err != nil {
			return nil, fmt.Errorf("invalid feed version ID '%s': %w", fvidStr, err)
		}
		feedVersionIDs = append(feedVersionIDs, fvid)
	}
	return feedVersionIDs, nil
}

// logResults logs the completion of an operation
func (cmd *FeedStateManagerCommand) logResults(ctx context.Context, manager *feedstate.Manager, duration time.Duration, message string) {
	log.For(ctx).Info().
		Dur("duration", duration).
		Msg(message)
}

// Help returns help text for the command
func (cmd *FeedStateManagerCommand) Help() string {
	return `Feed state management

This command provides precise control over which feed versions are active
and maintains materialized tables for optimal query performance.

OPERATIONS:

1. ACTIVATE feed versions:
   feed-state --activate 123,456,789
   
2. DEACTIVATE specific feed versions:
   feed-state --deactivate 123,456
   
3. COMBINE activate and deactivate:
   feed-state --activate 789 --deactivate 456,123
   
4. SET ACTIVE (replace entire active set):
   feed-state --set-active 123,456,789
   
5. SET ACTIVE from file:
   feed-state --set-active-fvid-file /path/to/active_feeds.txt

MAINTENANCE:
  # Sync materialized tables with current feed_states
  feed-state
   
  # Preview changes without applying
  feed-state --activate 789 --deactivate 456 --dry-run

OPERATION DETAILS:

--activate <ids>:
  Adds specified feed versions to the active set. Can be combined with --deactivate.

--deactivate <ids>:
  Removes specified feed versions from the active set. Can be combined with --activate.

--set-active <ids>:
  Replaces the ENTIRE active set with only these feed versions.
  Cannot be combined with --activate or --deactivate.

--set-active-fvid-file <file>:
  Same as --set-active but reads feed version IDs from file (one per line).

EXAMPLES:

# Activate new feed versions
feed-state --activate 789,999

# Deactivate problematic feed versions  
feed-state --deactivate 456,123

# Activate and deactivate in one command
feed-state --activate 789 --deactivate 456

# Replace entire system state
feed-state --set-active 123,789,999

# Batch operation from file
echo -e "123\n789\n999" > active_feeds.txt
feed-state --set-active-fvid-file active_feeds.txt

# Preview before making changes
feed-state --activate 789 --deactivate 456 --dry-run

All operations automatically update materialized tables and maintain
consistency between feed_states and materialized indexes.
`
}
