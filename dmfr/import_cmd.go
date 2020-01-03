package dmfr

import (
	"flag"
	"fmt"
	"os"
	"sync"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/log"
)

type dmfrImportCommand struct {
	workers    int
	limit      uint64
	dburl      string
	gtfsdir    string
	s3         string
	coverdate  string
	latest     bool
	dryrun     bool
	activate   bool
	feedids    []string
	extensions arrayFlags
	adapter    gtdb.Adapter // allow for mocks
}

func (cmd *dmfrImportCommand) Run(args []string) error {
	fl := flag.NewFlagSet("import", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: import [feedids...]")
		fl.PrintDefaults()
	}
	fl.Var(&cmd.extensions, "ext", "Include GTFS Extension")
	fl.IntVar(&cmd.workers, "workers", 1, "Worker threads")
	fl.StringVar(&cmd.dburl, "dburl", "", "Database URL (default: $DMFR_DATABASE_URL)")
	fl.StringVar(&cmd.gtfsdir, "gtfsdir", ".", "GTFS Directory")
	fl.StringVar(&cmd.s3, "s3", "", "Get GTFS files from S3 bucket/prefix")
	fl.StringVar(&cmd.coverdate, "date", "", "Service on date")
	fl.Uint64Var(&cmd.limit, "limit", 0, "Import at most n feeds")
	fl.BoolVar(&cmd.latest, "latest", false, "Only import latest feed version available for each feed")
	fl.BoolVar(&cmd.dryrun, "dryrun", false, "Dry run; print feeds that would be imported and exit")
	fl.BoolVar(&cmd.activate, "activate", false, "Set as active feed version after import")
	fl.Parse(args)
	cmd.feedids = fl.Args()
	if cmd.dburl == "" {
		cmd.dburl = os.Getenv("DMFR_DATABASE_URL")
	}
	if cmd.adapter == nil {
		writer := mustGetWriter(cmd.dburl, true)
		cmd.adapter = writer.Adapter
		defer writer.Close()
	}
	// Query to get FVs to import
	q := cmd.adapter.Sqrl().
		Select("feed_versions.id").
		From("feed_versions").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id").
		LeftJoin("feed_version_gtfs_imports ON feed_versions.id = feed_version_gtfs_imports.feed_version_id").
		Where("feed_version_gtfs_imports.id IS NULL").
		OrderBy("feed_versions.id")
	if cmd.latest {
		// Only fetch latest feed version for each feed
		q = q.
			Join("(SELECT id, created_at, ROW_NUMBER() OVER (PARTITION BY feed_id ORDER BY created_at DESC) AS rank FROM feed_versions) latest ON latest.id = feed_versions.id").
			Where("latest.rank = 1")
	}
	if cmd.limit > 0 {
		// Max feeds
		q = q.Limit(cmd.limit)
	}
	if len(cmd.feedids) > 0 {
		// Limit to specified feeds
		q = q.Where(sq.Eq{"onestop_id": cmd.feedids})
	}
	// if cmd.coverdate == "" {
	// 	// Set default date
	// 	cmd.coverdate = time.Now().Format("2006-01-02")
	// }
	if cmd.coverdate != "" {
		// Limit to service date
		q = q.
			Where(sq.LtOrEq{"feed_versions.earliest_calendar_date": cmd.coverdate}).
			Where(sq.GtOrEq{"feed_versions.latest_calendar_date": cmd.coverdate})
	}
	qstr, qargs, err := q.ToSql()
	if err != nil {
		return err
	}
	qrs := []int{}
	err = cmd.adapter.Select(&qrs, qstr, qargs...)
	if err != nil {
		return err
	}
	///////////////
	// Here we go
	log.Info("Importing %d feed versions", len(qrs))
	jobs := make(chan ImportOptions, len(qrs))
	results := make(chan ImportResult, len(qrs))
	for _, fvid := range qrs {
		jobs <- ImportOptions{
			FeedVersionID: fvid,
			Directory:     cmd.gtfsdir,
			S3:            cmd.s3,
			Extensions:    cmd.extensions,
			Activate:      cmd.activate,
		}
	}
	close(jobs)
	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < cmd.workers; w++ {
		wg.Add(1)
		go dmfrImportWorker(w, cmd.adapter, cmd.dryrun, jobs, results, &wg)
	}
	wg.Wait()
	return nil
}

func dmfrImportWorker(id int, adapter gtdb.Adapter, dryrun bool, jobs <-chan ImportOptions, results chan<- ImportResult, wg *sync.WaitGroup) {
	type qr struct {
		FeedVersionID   int
		FeedID          int
		FeedOnestopID   string
		FeedVersionSHA1 string
	}
	for opts := range jobs {
		q := qr{}
		if err := adapter.Get(&q, "SELECT feed_versions.id as feed_version_id, feed_Versions.feed_id as feed_id, feed_versions.sha1 as sha1, current_feeds.onestop_id as feed_onestop_id FROM feed_versions INNER JOIN current_feeds ON current_feeds.id = feed_versions.feed_id WHERE feed_versions.id = ?", opts.FeedVersionID); err != nil {
			log.Info("Serious error: could not get details for FeedVersion %d", opts.FeedVersionID)
			continue
		}
		log.Debug("Feed %s (id:%d): FeedVersion %s (id: %d): begin", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID)
		if dryrun {
			log.Info("Feed %s (id:%d): FeedVersion %s (id: %d): dry-run", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID)
			continue
		}
		result, err := MainImportFeedVersion(adapter, opts)
		if err != nil {
			log.Info("Feed %s (id:%d): FeedVersion %s (id: %d): critical failure, rolled back: %s", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, result.FeedVersionImport.ExceptionLog)
		} else if result.FeedVersionImport.Success {
			log.Info("Feed %s (id:%d): FeedVersion %s (id: %d): success: count %v errors: %v referrors: %v", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, result.FeedVersionImport.EntityCount, result.FeedVersionImport.SkipEntityErrorCount, result.FeedVersionImport.SkipEntityReferenceCount)
		} else {
			log.Info("Feed %s (id:%d): FeedVersion %s (id: %d): error: %s", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, result.FeedVersionImport.ExceptionLog)
		}
		results <- result
	}
	wg.Done()
}
