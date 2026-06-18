package cmds

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/interline-io/log"
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
	SyncActive         bool
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
	fl.BoolVar(&cmd.SyncActive, "sync-active", false, "Make materialized tables match current active feed versions")
	fl.BoolVar(&cmd.DryRun, "dry-run", false, "Show what would be done without making changes")
}

// Parse command line flags
func (cmd *FeedStateManagerCommand) Parse(args []string) error {
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
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
	// Open database connection
	if cmd.Adapter == nil {
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		cmd.Adapter = writer.Adapter
		defer cmd.Adapter.Close()
	}

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

	// Execute in transaction
	return cmd.Adapter.Tx(func(atx tldb.Adapter) error {
		// Get current active feed versions to determine sync operations
		log.Info().Msg("Getting current active feed versions")
		txManager := feedstate.NewManager(atx)
		activeFeedVersions, err := txManager.GetActiveFeedVersions(ctx)
		if err != nil {
			return fmt.Errorf("failed to get active feed versions: %w", err)
		}
		activeSet := toSet(activeFeedVersions)

		// Sync active feed versions to materialized tables
		if cmd.SyncActive {
			log.For(ctx).Info().Msg("Getting materialized feed versions to sync active feed versions")
			var materializedFeedVersions []int
			materializedFeedVersions, err = txManager.GetMaterializedFeedVersions(ctx)
			if err != nil {
				return fmt.Errorf("failed to get materialized feed versions: %w", err)
			}
			// Determine which feed versions to materialize and dematerialize
			materializedSet := toSet(materializedFeedVersions)
			for fvid := range activeSet {
				if !materializedSet[fvid] {
					forceMaterializeIDs = append(forceMaterializeIDs, fvid)
				}
			}
			for fvid := range materializedSet {
				if !activeSet[fvid] {
					forceDematerializeIDs = append(forceDematerializeIDs, fvid)
				}
			}
		}

		if len(setActiveIDs) > 0 {
			log.For(ctx).Info().Msg("Calculating set-active changes")
			changes, err := txManager.CalculateSetActiveChanges(ctx, setActiveIDs)
			if err != nil {
				return fmt.Errorf("failed to calculate set-active changes: %w", err)
			}
			activateIDs = append(activateIDs, changes.ToActivate...)
			deactivateIDs = append(deactivateIDs, changes.ToDeactivate...)
		}

		log.For(ctx).Info().Msg("The following operations will be performed:")
		if len(forceDematerializeIDs) > 0 {
			log.For(ctx).Info().Ints("feed_version_ids", forceDematerializeIDs).Msgf("DematerializeFeedVersion for %d feed versions", len(forceDematerializeIDs))
		}
		if len(forceMaterializeIDs) > 0 {
			log.For(ctx).Info().Ints("feed_version_ids", forceMaterializeIDs).Msgf("MaterializeFeedVersion for %d feed versions", len(forceMaterializeIDs))
		}
		if len(forceRematerializeIDs) > 0 {
			log.For(ctx).Info().Ints("feed_version_ids", forceRematerializeIDs).Msgf("DematerializeFeedVersion + MaterializeFeedVersion for %d feed versions", len(forceRematerializeIDs))
		}
		if len(deactivateIDs) > 0 {
			log.For(ctx).Info().Ints("feed_version_ids", deactivateIDs).Msgf("DeactivateFeedVersion for %d feed versions", len(deactivateIDs))
		}
		if len(activateIDs) > 0 {
			log.For(ctx).Info().Ints("feed_version_ids", activateIDs).Msgf("ActivateFeedVersion for %d feed versions", len(activateIDs))
		}

		// Dry run - show what would be done
		if cmd.DryRun {
			log.For(ctx).Info().Msg("Dry run enabled - no changes will be made")
			return nil
		}

		// Force dematerialize operations (need to get feed_id first)
		for _, fvid := range forceDematerializeIDs {
			log.For(ctx).Info().Int("feed_version_id", fvid).Msg("Force dematerializing feed version")
			if err := txManager.DematerializeFeedVersion(ctx, fvid); err != nil {
				return err
			}
		}

		// Force materialize operations
		for _, fvid := range forceMaterializeIDs {
			log.For(ctx).Info().Int("feed_version_id", fvid).Msg("Force materializing feed version")
			if err := txManager.MaterializeFeedVersion(ctx, fvid); err != nil {
				return err
			}
		}

		// Force rematerialize operations (dematerialize + materialize)
		for _, fvid := range forceRematerializeIDs {
			log.For(ctx).Info().Int("feed_version_id", fvid).Msg("Force rematerializing feed version")
			if err := txManager.DematerializeFeedVersion(ctx, fvid); err != nil {
				return err
			}
			if err := txManager.MaterializeFeedVersion(ctx, fvid); err != nil {
				return err
			}
		}

		// Deactivate operations
		for _, fvid := range deactivateIDs {
			log.For(ctx).Info().Int("feed_version_id", fvid).Msg("Deactivating feed version")
			if err := txManager.DeactivateFeedVersion(ctx, fvid); err != nil {
				return err
			}
		}

		// Activate operations
		for _, fvid := range activateIDs {
			log.For(ctx).Info().Int("feed_version_id", fvid).Msg("Activating feed version")
			if err := txManager.ActivateFeedVersion(ctx, fvid); err != nil {
				return err
			}
		}

		log.For(ctx).Info().Msg("Feed state operations completed successfully")
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

Manage which feed versions are active in the system and maintain materialized tables
for improved query performance.

BASIC OPERATIONS:
  --activate <ids>         Activate feed versions (deactivates other versions of same feeds)
  --deactivate <ids>       Deactivate feed versions  
  --set-active <ids>       Set complete active set (replaces all active versions)
  --set-active-fvid-file   Read feed version IDs from file (one per line)

MANUAL INTERVENTION:
  --force-materialize <ids>    Force materialize feed versions (add to materialized tables)
  --force-dematerialize <ids>  Force dematerialize feed versions (remove from materialized tables)
  --force-rematerialize <ids>  Force rematerialize (dematerialize + materialize)

OPTIONS:
  --dry-run               Show what would be done without making changes
  --dburl                 Database connection URL (or use TL_DATABASE_URL env var)

EXAMPLES:
  # Show current state
  feed-state
  
  # Activate specific feed versions
  feed-state --activate 123,456
  
  # Set complete active set (deactivates all others)
  feed-state --set-active 123,456,789
  
  # Load active set from file
  feed-state --set-active-fvid-file active_feeds.txt
  
  # Force rematerialize problematic feed version
  feed-state --force-rematerialize 789
  
  # Preview operations without changes
  feed-state --dry-run --activate 123

NOTES:
  - All operations are executed in a single transaction
  - Activating a feed version automatically deactivates other versions of the same feed
  - Materialized tables contain denormalized data for faster queries
  - Use force operations only when normal activation/deactivation fails
`
}

func toSet(ints []int) map[int]bool {
	set := make(map[int]bool)
	for _, v := range ints {
		set[v] = true
	}
	return set
}
