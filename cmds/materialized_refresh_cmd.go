package cmds

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/materialized"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/spf13/pflag"
)

// MaterializedRefreshCommand refreshes materialized route and stop tables
type MaterializedRefreshCommand struct {
	DBURL       string
	FeedIDs     []string
	FullRefresh bool
	DryRun      bool
	Stats       bool
	Health      bool
	Adapter     tldb.Adapter // allow for mocks
}

func (cmd *MaterializedRefreshCommand) HelpDesc() (string, string) {
	return "Refresh materialized route and stop index tables", `This command manages materialized tables that cache active route and stop data for improved query performance. It can perform incremental updates or full rebuilds.`
}

func (cmd *MaterializedRefreshCommand) HelpArgs() string {
	return ""
}

func (cmd *MaterializedRefreshCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL")
	fl.StringSliceVar(&cmd.FeedIDs, "feed-ids", nil, "Specific feed IDs to refresh (comma-separated)")
	fl.BoolVar(&cmd.FullRefresh, "full-refresh", false, "Rebuild all materialized tables from scratch")
	fl.BoolVar(&cmd.DryRun, "dry-run", false, "Show what would be done without making changes")
	fl.BoolVar(&cmd.Stats, "stats", false, "Show materialized table statistics")
	fl.BoolVar(&cmd.Health, "health", false, "Show health check results")
}

// Parse command line flags
func (cmd *MaterializedRefreshCommand) Parse(args []string) error {
	fl := pflag.NewFlagSet("materialized-refresh", pflag.ContinueOnError)
	fl.Usage = func() {
		fmt.Println("Usage: materialized-refresh")
		fl.PrintDefaults()
	}
	cmd.AddFlags(fl)
	return fl.Parse(args)
}

// Run the materialized refresh command
func (cmd *MaterializedRefreshCommand) Run(ctx context.Context) error {
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

	refresher := materialized.NewRefresher(cmd.Adapter)

	// Handle stats request
	if cmd.Stats {
		return cmd.showStats(ctx, refresher)
	}

	// Parse feed IDs if provided
	var feedIDs []int
	if len(cmd.FeedIDs) > 0 {
		for _, feedIDStr := range cmd.FeedIDs {
			feedID, err := strconv.Atoi(feedIDStr)
			if err != nil {
				return fmt.Errorf("invalid feed ID '%s': %w", feedIDStr, err)
			}
			feedIDs = append(feedIDs, feedID)
		}
	}

	opts := materialized.RefreshOptions{
		FeedIDs: feedIDs,
		DryRun:  cmd.DryRun,
	}

	// Execute refresh
	start := time.Now()

	if cmd.FullRefresh {
		log.For(ctx).Info().Msg("Starting full refresh of materialized tables")
		if err := refresher.FullRefresh(ctx); err != nil {
			return fmt.Errorf("full refresh failed: %w", err)
		}
	} else {
		log.For(ctx).Info().Msg("Starting incremental refresh of materialized tables")
		if err := refresher.RefreshMaterializedTables(ctx, opts); err != nil {
			return fmt.Errorf("incremental refresh failed: %w", err)
		}
	}

	duration := time.Since(start)

	if !cmd.DryRun {
		// Show final stats
		stats, err := refresher.GetRefreshStats(ctx)
		if err != nil {
			log.For(ctx).Warn().Err(err).Msg("Failed to get refresh stats")
		} else {
			log.For(ctx).Info().
				Int("routes", stats.RouteCount).
				Int("stops", stats.StopCount).
				Int("feeds", stats.FeedCount).
				Int("total_feeds", stats.TotalFeeds).
				Int("out_of_sync", stats.OutOfSync).
				Dur("duration", duration).
				Msg("Materialized refresh completed")
		}
	} else {
		log.For(ctx).Info().
			Dur("duration", duration).
			Msg("Dry run completed")
	}

	return nil
}

// showStats displays materialized table statistics
func (cmd *MaterializedRefreshCommand) showStats(ctx context.Context, refresher *materialized.Refresher) error {
	stats, err := refresher.GetRefreshStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	fmt.Printf("Materialized Table Statistics:\n")
	fmt.Printf("  Routes: %d\n", stats.RouteCount)
	fmt.Printf("  Stops: %d\n", stats.StopCount)
	fmt.Printf("  Materialized Feeds: %d\n", stats.FeedCount)
	fmt.Printf("  Total Active Feeds: %d\n", stats.TotalFeeds)
	fmt.Printf("  Out of Sync: %d\n", stats.OutOfSync)
	if stats.LastRefresh != nil {
		fmt.Printf("  Last Refresh: %s\n", stats.LastRefresh.Format(time.RFC3339))
	} else {
		fmt.Printf("  Last Refresh: Never\n")
	}

	return nil
}

// Help returns help text for the command
func (cmd *MaterializedRefreshCommand) Help() string {
	return `Refresh materialized route and stop index tables

This command manages materialized tables that cache active route and stop data
for improved query performance. It can perform incremental updates or full rebuilds.

Examples:
  # Show current statistics
  materialized-refresh --dburl postgres://... --stats

  # Perform incremental refresh (recommended)
  materialized-refresh --dburl postgres://... 

  # Refresh specific feeds only
  materialized-refresh --dburl postgres://... --feed-ids 1,2,3

  # Full rebuild (slower, but guaranteed consistency)
  materialized-refresh --dburl postgres://... --full-refresh

  # See what changes would be made without applying them
  materialized-refresh --dburl postgres://... --dry-run
`
}
