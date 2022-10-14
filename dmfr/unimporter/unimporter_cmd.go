package unimporter

import (
	"errors"
	"flag"
	"os"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/internal/cli"
	"github.com/interline-io/transitland-lib/log"
	"github.com/interline-io/transitland-lib/tldb"
)

// Command imports FeedVersions into a database.
type Command struct {
	ScheduleOnly bool
	ExtraTables  []string
	DryRun       bool
	FVIDs        []string
	FVSHA1       cli.ArrayFlags
	Extensions   cli.ArrayFlags
	FeedIDs      cli.ArrayFlags
	DBURL        string
	Workers      int
	Adapter      tldb.Adapter // allow for mocks
}

// Parse command line flags
func (cmd *Command) Parse(args []string) error {
	fvidfile := ""
	fvsha1file := ""
	fl := flag.NewFlagSet("import", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: unimport [fvids]")
		fl.PrintDefaults()
	}
	// fl.Var(&cmd.Extensions, "ext", "Include GTFS Extension") // TODO
	fl.Var(&cmd.FeedIDs, "feed", "Feed ID")
	fl.Var(&cmd.FVSHA1, "fv-sha1", "Feed version SHA1")
	fl.StringVar(&fvidfile, "fvid-file", "", "Specify feed version IDs in file, one per line; equivalent to multiple --fvid")
	fl.StringVar(&fvsha1file, "fv-sha1-file", "", "Specify feed version IDs by SHA1 in file, one per line")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.BoolVar(&cmd.DryRun, "dryrun", false, "Dry run; print feeds that would be imported and exit")
	fl.BoolVar(&cmd.ScheduleOnly, "schedule-only", false, "Unimport stop times, trips, transfers, shapes, and frequencies")
	fl.Parse(args)
	cmd.Workers = 1
	cmd.FVIDs = fl.Args()
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	if fvidfile != "" {
		lines, err := cli.ReadFileLines(fvidfile)
		if err != nil {
			return err
		}
		for _, line := range lines {
			if line != "" {
				cmd.FVIDs = append(cmd.FVIDs, line)
			}
		}
	}
	if fvsha1file != "" {
		lines, err := cli.ReadFileLines(fvsha1file)
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
func (cmd *Command) Run() error {
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
		log.Infof("Unmporting schedule data from %d feed versions", len(qrs))
	} else {
		log.Infof("Unmporting %d feed versions", len(qrs))
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
		go dmfrUnimportWorker(w, cmd.Adapter, jobs, &wg)
	}
	wg.Wait()
	return nil
}

func dmfrUnimportWorker(id int, adapter tldb.Adapter, jobs <-chan jobOptions, wg *sync.WaitGroup) {
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
			log.Infof("Feed %s (id:%d): FeedVersion %s (id:%d): dry-run", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID)
			continue
		}
		log.Infof("Feed %s (id:%d): FeedVersion %s (id:%d): begin", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID)
		t := time.Now()
		err := adapter.Tx(func(atx tldb.Adapter) error {
			var err error
			if opts.ScheduleOnly {
				err = UnimportSchedule(atx, opts.FeedVersionID)
			} else {
				err = UnimportFeedVersion(atx, opts.FeedVersionID, opts.ExtraTables)
			}
			return err
		})
		t2 := float64(time.Now().UnixNano()-t.UnixNano()) / 1e9 // 1000000000.0
		if err != nil {
			log.Errorf("Feed %s (id:%d): FeedVersion %s (id:%d): critical failure, rolled back: %s (t:%0.2fs)", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, err.Error(), t2)
		} else {
			log.Infof("Feed %s (id:%d): FeedVersion %s (id:%d): success (t:%0.2fs)", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, t2)
		}
	}
	wg.Done()
}
