package cmds

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/importer"
	"github.com/interline-io/transitland-lib/tldb"
	sq "github.com/irees/squirrel"
	"github.com/spf13/pflag"
)

// UnimportCommand imports FeedVersions into a database.
type UnimportCommand struct {
	FVArgs       FeedVersionArgs
	ScheduleOnly bool
	ExtraTables  []string
	DryRun       bool
	Extensions   []string
	DBURL        string
	Workers      int
	Adapter      tldb.Adapter // allow for mocks
}

func (cmd *UnimportCommand) HelpDesc() (string, string) {
	return "Unimport feed versions", "The `unimport` command deletes previously imported data from feed versions. The feed version record itself is not deleted. You may optionally specify removal of only schedule data, leaving routes, stops, etc. in place."
}

func (cmd *UnimportCommand) HelpArgs() string {
	return "[flags] <fvid>..."
}

func (cmd *UnimportCommand) AddFlags(fl *pflag.FlagSet) {
	cmd.FVArgs.AddFlags(fl)
	fl.StringSliceVar(&cmd.ExtraTables, "extra-table", nil, "Extra tables to delete feed_version_id")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.BoolVar(&cmd.DryRun, "dryrun", false, "Dry run; log the feed versions that would be unimported and exit")
	fl.BoolVar(&cmd.ScheduleOnly, "schedule-only", false, "Unimport stop times, trips, transfers, shapes, and frequencies")
}

// Parse command line flags
func (cmd *UnimportCommand) Parse(args []string) error {
	cmd.Workers = 1
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	if err := cmd.FVArgs.Parse(args); err != nil {
		return err
	}
	if cmd.FVArgs.Empty() {
		return errors.New("must provide at least one feed version id as an argument or with --fvid-file")
	}
	return nil
}

type jobOptions struct {
	FeedVersionID int
	ScheduleOnly  bool
	ExtraTables   []string
	DryRun        bool
}

// Run this command
func (cmd *UnimportCommand) Run(ctx context.Context) error {
	if cmd.Adapter == nil {
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return err
		}
		cmd.Adapter = writer.Adapter
		defer writer.Close()
	}
	// Resolve to imported feed versions; unimport only applies to versions that
	// have an import record.
	sel := sq.Or{}
	if len(cmd.FVArgs.FVIDs) > 0 {
		sel = append(sel, sq.Eq{"feed_versions.id": cmd.FVArgs.FVIDs})
	}
	if len(cmd.FVArgs.FVSHA1) > 0 {
		sel = append(sel, sq.Eq{"feed_versions.sha1": cmd.FVArgs.FVSHA1})
	}
	qrs := []int{}
	q := cmd.Adapter.Sqrl().
		Select("feed_versions.id").
		From("feed_versions").
		LeftJoin("feed_version_gtfs_imports ON feed_versions.id = feed_version_gtfs_imports.feed_version_id").
		Where("feed_version_gtfs_imports.id IS NOT NULL").
		Where(sel).
		OrderBy("feed_versions.id desc")
	qstr, qargs, err := q.ToSql()
	if err != nil {
		return err
	}
	if err := cmd.Adapter.Select(ctx, &qrs, qstr, qargs...); err != nil {
		return err
	}
	if cmd.ScheduleOnly {
		log.For(ctx).Info().Msgf("Unmporting schedule data from %d feed versions", len(qrs))
	} else {
		log.For(ctx).Info().Msgf("Unmporting %d feed versions", len(qrs))
	}

	jobs := make(chan jobOptions, len(qrs))
	for _, fvid := range qrs {
		jobs <- jobOptions{
			FeedVersionID: fvid,
			ScheduleOnly:  cmd.ScheduleOnly,
			ExtraTables:   cmd.ExtraTables,
			DryRun:        cmd.DryRun,
		}
	}
	close(jobs)
	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < cmd.Workers; w++ {
		wg.Add(1)
		go dmfrUnimportWorker(w, ctx, cmd.Adapter, jobs, &wg)
	}
	wg.Wait()
	return nil
}

func dmfrUnimportWorker(id int, ctx context.Context, adapter tldb.Adapter, jobs <-chan jobOptions, wg *sync.WaitGroup) {
	_ = id
	type qr struct {
		FeedVersionID   int
		FeedID          int
		FeedOnestopID   string
		FeedVersionSHA1 string
	}
	for opts := range jobs {
		q := qr{}
		query := `
		SELECT
			feed_versions.id as feed_version_id,
			feed_versions.feed_id as feed_id,
			feed_versions.sha1 as feed_version_sha1,
			current_feeds.onestop_id as feed_onestop_id
		FROM feed_versions
		INNER JOIN current_feeds ON current_feeds.id = feed_versions.feed_id
		WHERE feed_versions.id = ?
		`
		if err := adapter.Get(ctx, &q, query, opts.FeedVersionID); err != nil {
			log.For(ctx).Error().Msgf("Could not get details for FeedVersion %d", opts.FeedVersionID)
			continue
		}
		if opts.DryRun {
			log.For(ctx).Info().Msgf("Feed %s (id:%d): FeedVersion %s (id:%d): dry-run", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID)
			continue
		}
		log.For(ctx).Info().Msgf("Feed %s (id:%d): FeedVersion %s (id:%d): begin", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID)
		t := time.Now()
		err := adapter.Tx(func(atx tldb.Adapter) error {
			var err error
			if opts.ScheduleOnly {
				err = importer.UnimportSchedule(ctx, atx, opts.FeedVersionID)
			} else {
				err = importer.UnimportFeedVersion(ctx, atx, opts.FeedVersionID, opts.ExtraTables)
			}
			return err
		})
		t2 := float64(time.Now().UnixNano()-t.UnixNano()) / 1e9 // 1000000000.0
		if err != nil {
			log.For(ctx).Error().Msgf("Feed %s (id:%d): FeedVersion %s (id:%d): critical failure, rolled back: %s (t:%0.2fs)", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, err.Error(), t2)
		} else {
			log.For(ctx).Info().Msgf("Feed %s (id:%d): FeedVersion %s (id:%d): success (t:%0.2fs)", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, t2)
		}
	}
	wg.Done()
}
