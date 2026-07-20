package cmds

import (
	"context"
	"errors"
	"os"
	"sync"

	sq "github.com/irees/squirrel"
	"github.com/spf13/pflag"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/importer"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/tldb"
)

// StatsRemoveOnestopIDsCommand removes onestop_id stats (agency/route/stop) for
// feed versions, e.g. to enforce a feed's onestop_id retention period. The feed
// versions themselves are unaffected; the removed rows only power AllowPrevious
// lookups. The active and materialized feed versions are always skipped.
type StatsRemoveOnestopIDsCommand struct {
	Workers  int
	DBURL    string
	DryRun   bool
	FVIDs    []string
	Adapter  tldb.Adapter // allow for mocks
	fvidfile string
}

func (cmd *StatsRemoveOnestopIDsCommand) HelpDesc() (string, string) {
	return "Remove onestop_id stats for feed versions",
		"Deletes agency/route/stop onestop_id rows for the given feed versions; the feed versions are otherwise unaffected. The active and materialized feed versions are always skipped."
}

func (cmd *StatsRemoveOnestopIDsCommand) HelpArgs() string {
	return "[flags]"
}

func (cmd *StatsRemoveOnestopIDsCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringSliceVar(&cmd.FVIDs, "fvid", nil, "Remove onestop_id stats for specific feed version ID")
	fl.StringVar(&cmd.fvidfile, "fvid-file", "", "Specify feed version IDs in file, one per line; equivalent to multiple --fvid")
	fl.IntVar(&cmd.Workers, "workers", 1, "Worker threads")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.BoolVar(&cmd.DryRun, "dryrun", false, "Dry run; log the feed versions that would be affected and exit")
}

func (cmd *StatsRemoveOnestopIDsCommand) Parse(args []string) error {
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	if cmd.fvidfile != "" {
		lines, err := tlcli.ReadFileLines(cmd.fvidfile)
		if err != nil {
			return err
		}
		for _, line := range lines {
			if line != "" {
				cmd.FVIDs = append(cmd.FVIDs, line)
			}
		}
	}
	if len(cmd.FVIDs) == 0 {
		return errors.New("specify at least one feed version with --fvid or --fvid-file")
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
	// Resolve to existing feed versions, always skipping the active and materialized
	// versions so a stale fvid list can never strip a live version.
	q := cmd.Adapter.Sqrl().
		Select("feed_versions.id").
		From("feed_versions").
		Where(sq.Eq{"feed_versions.id": cmd.FVIDs}).
		Where(`feed_versions.id NOT IN (
			SELECT active_feed_version_id FROM feed_states WHERE active_feed_version_id IS NOT NULL
			UNION
			SELECT materialized_feed_version_id FROM feed_states WHERE materialized_feed_version_id IS NOT NULL
		)`).
		OrderBy("feed_versions.id")
	qstr, qargs, err := q.ToSql()
	if err != nil {
		return err
	}
	fvids := []int{}
	if err := cmd.Adapter.Select(ctx, &fvids, qstr, qargs...); err != nil {
		return err
	}
	log.For(ctx).Info().
		Int("requested", len(cmd.FVIDs)).
		Int("selected", len(fvids)).
		Msg("stats-remove-onestop-ids: resolved feed versions (active/materialized and unknown ids skipped)")
	if cmd.DryRun {
		for _, fvid := range fvids {
			log.For(ctx).Info().Int("feed_version_id", fvid).Msg("stats-remove-onestop-ids: would remove (dry run)")
		}
		return nil
	}

	jobs := make(chan int, len(fvids))
	for _, fvid := range fvids {
		jobs <- fvid
	}
	close(jobs)
	var wg sync.WaitGroup
	for w := 0; w < cmd.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fvid := range jobs {
				err := cmd.Adapter.Tx(func(atx tldb.Adapter) error {
					return importer.RemoveOnestopIds(ctx, atx, fvid)
				})
				if err != nil {
					log.For(ctx).Error().Err(err).Int("feed_version_id", fvid).Msg("stats-remove-onestop-ids: failed")
				} else {
					log.For(ctx).Info().Int("feed_version_id", fvid).Msg("stats-remove-onestop-ids: removed")
				}
			}
		}()
	}
	wg.Wait()
	return nil
}
