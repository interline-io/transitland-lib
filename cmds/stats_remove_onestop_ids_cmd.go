package cmds

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/importer"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/spf13/pflag"
)

// StatsRemoveOnestopIDsCommand removes onestop_id stats (agency/route/stop) for
// feed versions, e.g. to enforce a feed's onestop_id retention period. The feed
// versions themselves are unaffected; the removed rows only power AllowPrevious
// lookups. The active and materialized feed versions are always skipped.
type StatsRemoveOnestopIDsCommand struct {
	FVArgs  FeedVersionArgs
	Workers int
	DBURL   string
	DryRun  bool
	Adapter tldb.Adapter // allow for mocks
}

func (cmd *StatsRemoveOnestopIDsCommand) HelpDesc() (string, string) {
	return "Remove onestop_id stats for feed versions",
		"Deletes agency/route/stop onestop_id rows for the given feed versions; the feed versions are otherwise unaffected. The active and materialized feed versions are always skipped."
}

func (cmd *StatsRemoveOnestopIDsCommand) HelpArgs() string {
	return "[flags] <fvid>..."
}

func (cmd *StatsRemoveOnestopIDsCommand) AddFlags(fl *pflag.FlagSet) {
	cmd.FVArgs.AddFlags(fl)
	fl.IntVar(&cmd.Workers, "workers", 1, "Worker threads")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	addDryRunFlag(fl, &cmd.DryRun, "Dry run; log the feed versions that would be affected and exit")
}

func (cmd *StatsRemoveOnestopIDsCommand) Parse(args []string) error {
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	if err := cmd.FVArgs.Parse(args); err != nil {
		return err
	}
	if cmd.FVArgs.Empty() {
		return errors.New("must provide at least one feed version id as an argument or with --fvid-file")
	}
	if cmd.Workers < 1 {
		cmd.Workers = 1
	}
	return nil
}

func (cmd *StatsRemoveOnestopIDsCommand) Run(ctx context.Context) error {
	if cmd.Adapter == nil {
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return err
		}
		cmd.Adapter = writer.Adapter
		defer writer.Close()
	}
	fvids, err := cmd.FVArgs.SelectIDs(ctx, cmd.Adapter)
	if err != nil {
		return err
	}
	// Never strip the active or materialized version of any feed.
	fvids, err = excludeLiveVersions(ctx, cmd.Adapter, fvids)
	if err != nil {
		return err
	}
	log.For(ctx).Info().
		Int("selected", len(fvids)).
		Msg("resolved feed versions; active/materialized skipped")
	if cmd.DryRun {
		for _, fvid := range fvids {
			log.For(ctx).Info().Int("feed_version_id", fvid).Msg("dry-run")
		}
		return nil
	}

	jobs := make(chan int, len(fvids))
	for _, fvid := range fvids {
		jobs <- fvid
	}
	close(jobs)
	var wg sync.WaitGroup
	var failed int64
	for w := 0; w < cmd.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fvid := range jobs {
				log.For(ctx).Info().Int("feed_version_id", fvid).Msg("begin")
				err := cmd.Adapter.Tx(func(atx tldb.Adapter) error {
					return importer.RemoveOnestopIds(ctx, atx, fvid)
				})
				if err != nil {
					atomic.AddInt64(&failed, 1)
					log.For(ctx).Error().Err(err).Int("feed_version_id", fvid).Msg("failure")
				} else {
					log.For(ctx).Info().Int("feed_version_id", fvid).Msg("success")
				}
			}
		}()
	}
	wg.Wait()
	if n := atomic.LoadInt64(&failed); n > 0 {
		return fmt.Errorf("stats-remove-onestop-ids: %d of %d feed versions failed", n, len(fvids))
	}
	return nil
}
