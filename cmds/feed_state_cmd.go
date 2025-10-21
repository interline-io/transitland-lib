package cmds

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/interline-io/transitland-lib/internal/feedstate"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/spf13/pflag"
)

// FeedStateManagerCommand manages feed state and materialized index tables
type FeedStateManagerCommand struct {
	DBURL              string
	ActivateFVIDs      []string
	DeactivateFVIDs    []string
	SetActiveFVIDs     []string
	SetActiveFVIDFile  string
	ForceMaterialize   []string
	ForceDematerialize []string
	ForceRematerialize []string
	DryRun             bool
	Adapter            tldb.Adapter // allow for mocks
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
	fl.StringSliceVar(&cmd.ForceMaterialize, "force-materialize", nil, "Force materialize these feed version IDs (manual intervention)")
	fl.StringSliceVar(&cmd.ForceDematerialize, "force-dematerialize", nil, "Force dematerialize these feed version IDs (manual intervention)")
	fl.StringSliceVar(&cmd.ForceRematerialize, "force-rematerialize", nil, "Force rematerialize these feed version IDs (dematerialize + materialize)")
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

	// Check for operations that modify feed state
	hasOperations := len(cmd.SetActiveFVIDs) > 0 || len(cmd.ActivateFVIDs) > 0 || len(cmd.DeactivateFVIDs) > 0 ||
		len(cmd.ForceMaterialize) > 0 || len(cmd.ForceDematerialize) > 0 || len(cmd.ForceRematerialize) > 0

	if hasOperations {
		// Handle feed state operations
		return cmd.handleFeedStateOperations(ctx, manager)
	}

	return nil
}

// handleFeedStateOperations processes all feed state operations
func (cmd *FeedStateManagerCommand) handleFeedStateOperations(ctx context.Context, manager *feedstate.Manager) error {
	// Parse feed version IDs
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

	forceMaterializeIDs, err := cmd.parseFeedVersionIDs(cmd.ForceMaterialize)
	if err != nil {
		return fmt.Errorf("invalid force-materialize IDs: %w", err)
	}

	forceDematerializeIDs, err := cmd.parseFeedVersionIDs(cmd.ForceDematerialize)
	if err != nil {
		return fmt.Errorf("invalid force-dematerialize IDs: %w", err)
	}

	forceRematerializeIDs, err := cmd.parseFeedVersionIDs(cmd.ForceRematerialize)
	if err != nil {
		return fmt.Errorf("invalid force-rematerialize IDs: %w", err)
	}

	// Basic validation
	if len(setActiveIDs) > 0 && (len(activateIDs) > 0 || len(deactivateIDs) > 0) {
		return fmt.Errorf("--set-active cannot be used with --activate or --deactivate")
	}

	// Dry run - show what would be done
	if cmd.DryRun {
		fmt.Printf("DRY RUN: Would execute the following operations:\n")
		if len(setActiveIDs) > 0 {
			fmt.Printf("  SetActiveFeedVersions(%v)\n", setActiveIDs)
		}
		if len(activateIDs) > 0 {
			fmt.Printf("  ActivateFeedVersion for each: %v\n", activateIDs)
		}
		if len(deactivateIDs) > 0 {
			fmt.Printf("  DeactivateFeedVersion for each: %v\n", deactivateIDs)
		}
		if len(forceMaterializeIDs) > 0 {
			fmt.Printf("  MaterializeFeedVersion for each: %v\n", forceMaterializeIDs)
		}
		if len(forceDematerializeIDs) > 0 {
			fmt.Printf("  DematerializeFeedVersion for each: %v\n", forceDematerializeIDs)
		}
		if len(forceRematerializeIDs) > 0 {
			fmt.Printf("  DematerializeFeedVersion + MaterializeFeedVersion for each: %v\n", forceRematerializeIDs)
		}
		return nil
	}

	// Execute in transaction - thin wrapper around manager methods
	return cmd.Adapter.Tx(func(atx tldb.Adapter) error {
		txManager := feedstate.NewManager(atx)

		if len(setActiveIDs) > 0 {
			return txManager.SetActiveFeedVersions(ctx, setActiveIDs)
		}

		for _, fvid := range deactivateIDs {
			if err := txManager.DeactivateFeedVersion(ctx, fvid); err != nil {
				return err
			}
		}

		for _, fvid := range activateIDs {
			if err := txManager.ActivateFeedVersion(ctx, fvid); err != nil {
				return err
			}
		}

		// Force dematerialize operations (need to get feed_id first)
		for _, fvid := range forceDematerializeIDs {
			if err := txManager.DematerializeFeedVersion(ctx, fvid); err != nil {
				return err
			}
		}

		// Force materialize operations
		for _, fvid := range forceMaterializeIDs {
			if err := txManager.MaterializeFeedVersion(ctx, fvid); err != nil {
				return err
			}
		}

		// Force rematerialize operations (dematerialize + materialize)
		for _, fvid := range forceRematerializeIDs {
			if err := txManager.DematerializeFeedVersion(ctx, fvid); err != nil {
				return err
			}
			if err := txManager.MaterializeFeedVersion(ctx, fvid); err != nil {
				return err
			}
		}

		return nil
	})
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

// Help returns help text for the command
func (cmd *FeedStateManagerCommand) Help() string {
	return `Feed state management

Manage which feed versions are active in the system.

BASIC OPERATIONS:
  --activate <ids>         Activate feed versions
  --deactivate <ids>       Deactivate feed versions  
  --set-active <ids>       Set complete active set (replaces all)

MANUAL INTERVENTION:
  --force-materialize <ids>    Force materialize feed versions
  --force-dematerialize <ids>  Force dematerialize feed versions
  --force-rematerialize <ids>  Force rematerialize (dematerialize + materialize)

OPTIONS:
  --dry-run               Show what would be done

EXAMPLES:
  feed-state --activate 123,456
  feed-state --set-active 123,456,789
  feed-state --force-rematerialize 789
`
}
