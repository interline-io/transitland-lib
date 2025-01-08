package cmds

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/importer"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/spf13/pflag"
)

// UnimportCommand imports FeedVersions into a database.
type UnimportCommand struct {
	ScheduleOnly bool
	ExtraTables  []string
	DryRun       bool
	FVIDs        []string
	FVSHA1       []string
	Extensions   []string
	FeedIDs      []string
	DBURL        string
	Workers      int
	Adapter      tldb.Adapter // allow for mocks
	// internal
	fvidfile   string
	fvsha1file string
}

func (cmd *UnimportCommand) HelpDesc() (string, string) {
	return "Unimport feed versions", "The `unimport` command deletes previously imported data from feed versions. The feed version record itself is not deleted. You may optionally specify removal of only schedule data, leaving routes, stops, etc. in place."
}

func (cmd *UnimportCommand) HelpArgs() string {
	return "[flags] <fvids...>"
}

func (cmd *UnimportCommand) AddFlags(fl *pflag.FlagSet) {
	// fl.Var(&cmd.Extensions, "ext", "Include GTFS Extension") // TODO
	fl.StringSliceVar(&cmd.ExtraTables, "extra-table", nil, "Extra tables to delete feed_version_id")
	fl.StringSliceVar(&cmd.FeedIDs, "feed", nil, "Feed ID")
	fl.StringSliceVar(&cmd.FVSHA1, "fv-sha1", nil, "Feed version SHA1")
	fl.StringVar(&cmd.fvidfile, "fvid-file", "", "Specify feed version IDs in file, one per line; equivalent to multiple --fvid")
	fl.StringVar(&cmd.fvsha1file, "fv-sha1-file", "", "Specify feed version IDs by SHA1 in file, one per line")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.BoolVar(&cmd.DryRun, "dryrun", false, "Dry run; print feeds that would be imported and exit")
	fl.BoolVar(&cmd.ScheduleOnly, "schedule-only", false, "Unimport stop times, trips, transfers, shapes, and frequencies")

}

// Parse command line flags
func (cmd *UnimportCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	cmd.Workers = 1
	cmd.FVIDs = fl.Args()
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
	if cmd.fvsha1file != "" {
		lines, err := tlcli.ReadFileLines(cmd.fvsha1file)
		if err != nil {
			return err
		}
		for _, line := range lines {
			if line != "" {
				cmd.FVSHA1 = append(cmd.FVSHA1, line)
			}
		}
	}
	if len(cmd.FeedIDs)+len(cmd.FVIDs)+len(cmd.FVSHA1) == 0 {
		return errors.New("must provide feed ids, feed version ids, or feed version sha1s")
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
	qrs := []int{}
	q := cmd.Adapter.Sqrl().
		Select("feed_versions.id").
		From("feed_versions").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id").
		LeftJoin("feed_version_gtfs_imports ON feed_versions.id = feed_version_gtfs_imports.feed_version_id").
		Where("feed_version_gtfs_imports.id IS NOT NULL").
		OrderBy("feed_versions.id desc")
	if len(cmd.FeedIDs) > 0 {
		// Limit to specified feeds
		q = q.Where(sq.Eq{"onestop_id": cmd.FeedIDs})
	}
	if len(cmd.FVIDs) > 0 {
		// Explicitly specify fvids
		q = q.Where(sq.Eq{"feed_versions.id": cmd.FVIDs})
	}
	if len(cmd.FVSHA1) > 0 {
		// Explicitly specify fv sha1
		q = q.Where(sq.Eq{"feed_versions.sha1": cmd.FVSHA1})
	}
	qstr, qargs, err := q.ToSql()
	if err != nil {
		return err
	}
	err = cmd.Adapter.Select(&qrs, qstr, qargs...)
	if err != nil {
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
		if err := adapter.Get(&q, query, opts.FeedVersionID); err != nil {
			log.Errorf("Could not get details for FeedVersion %d", opts.FeedVersionID)
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
				err = importer.UnimportSchedule(atx, opts.FeedVersionID)
			} else {
				err = importer.UnimportFeedVersion(atx, opts.FeedVersionID, opts.ExtraTables)
			}
			return err
		})
		t2 := float64(time.Now().UnixNano()-t.UnixNano()) / 1e9 // 1000000000.0
		if err != nil {
			log.Errorf("Feed %s (id:%d): FeedVersion %s (id:%d): critical failure, rolled back: %s (t:%0.2fs)", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, err.Error(), t2)
		} else {
			log.For(ctx).Info().Msgf("Feed %s (id:%d): FeedVersion %s (id:%d): success (t:%0.2fs)", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, t2)
		}
	}
	wg.Done()
}
