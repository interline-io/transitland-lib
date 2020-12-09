package dmfr

import (
	"bufio"
	"flag"
	"os"
	"strings"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tldb"
)

// ImportCommand imports FeedVersions into a database.
type ImportCommand struct {
	Workers       int
	Limit         int
	DBURL         string
	CoverDate     string
	FetchedSince  string
	Latest        bool
	DryRun        bool
	FeedIDs       []string
	FVIDs         arrayFlags
	FVSHA1        arrayFlags
	Adapter       tldb.Adapter // allow for mocks
	ImportOptions ImportOptions
}

// Parse command line flags
func (cmd *ImportCommand) Parse(args []string) error {
	extflags := arrayFlags{}
	fvidfile := ""
	fvsha1file := ""
	fl := flag.NewFlagSet("import", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: import [feedids...]")
		fl.PrintDefaults()
	}
	fl.Var(&extflags, "ext", "Include GTFS Extension")
	fl.Var(&cmd.FVIDs, "fvid", "Import specific feed version ID")
	fl.StringVar(&fvidfile, "fvid-file", "", "Specify feed version IDs in file, one per line; equivalent to multiple --fvid")
	fl.StringVar(&fvsha1file, "fv-sha1-file", "", "Specify feed version IDs by SHA1 in file, one per line")
	fl.IntVar(&cmd.Workers, "workers", 1, "Worker threads")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $DMFR_DATABASE_URL)")
	fl.StringVar(&cmd.ImportOptions.Directory, "gtfsdir", ".", "GTFS Directory")
	fl.StringVar(&cmd.ImportOptions.S3, "s3", "", "Get GTFS files from S3 bucket/prefix")
	fl.StringVar(&cmd.CoverDate, "date", "", "Service on date")
	fl.StringVar(&cmd.FetchedSince, "fetched-since", "", "Fetched since")
	fl.IntVar(&cmd.Limit, "limit", 0, "Import at most n feeds")
	fl.BoolVar(&cmd.Latest, "latest", false, "Only import latest feed version available for each feed")
	fl.BoolVar(&cmd.DryRun, "dryrun", false, "Dry run; print feeds that would be imported and exit")
	fl.BoolVar(&cmd.ImportOptions.Activate, "activate", false, "Set as active feed version after import")
	fl.BoolVar(&cmd.ImportOptions.InterpolateStopTimes, "interpolate-stop-times", false, "Interpolate missing StopTime arrival/departure values")
	fl.BoolVar(&cmd.ImportOptions.DeduplicateJourneyPatterns, "deduplicate-stop-times", false, "Deduplicate StopTimes using Journey Patterns")
	fl.BoolVar(&cmd.ImportOptions.CreateMissingShapes, "create-missing-shapes", false, "Create missing Shapes from Trip stop-to-stop geometries")
	fl.Parse(args)
	cmd.FeedIDs = fl.Args()
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("DMFR_DATABASE_URL")
	}
	cmd.ImportOptions.Extensions = extflags
	if fvidfile != "" {
		lines, err := getFileLines(fvidfile)
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
		lines, err := getFileLines(fvsha1file)
		if err != nil {
			return err
		}
		for _, line := range lines {
			if line != "" {
				cmd.FVSHA1 = append(cmd.FVSHA1, line)
			}
		}
	}
	return nil
}

// Run this command
func (cmd *ImportCommand) Run() error {
	if cmd.Adapter == nil {
		writer := mustGetWriter(cmd.DBURL, true)
		cmd.Adapter = writer.Adapter
		defer writer.Close()
	}
	// Query to get FVs to import
	q := cmd.Adapter.Sqrl().
		Select("feed_versions.id").
		From("feed_versions").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id").
		LeftJoin("feed_version_gtfs_imports ON feed_versions.id = feed_version_gtfs_imports.feed_version_id").
		Where("feed_version_gtfs_imports.id IS NULL").
		OrderBy("feed_versions.id")
	if cmd.Latest {
		// Only fetch latest feed version for each feed
		q = q.
			Join("(SELECT id, created_at, ROW_NUMBER() OVER (PARTITION BY feed_id ORDER BY created_at DESC) AS rank FROM feed_versions) latest ON latest.id = feed_versions.id").
			Where("latest.rank = 1")
	}
	if len(cmd.FeedIDs) > 0 {
		// Limit to specified feeds
		q = q.Where(sq.Eq{"onestop_id": cmd.FeedIDs})
	}
	if cmd.FetchedSince != "" {
		// Limit to feeds fetched since a given date
		q = q.Where(sq.GtOrEq{"feed_versions.fetched_at": cmd.FetchedSince})
	}
	if cmd.CoverDate != "" {
		// Limit to service date
		q = q.
			Where(sq.LtOrEq{"feed_versions.earliest_calendar_date": cmd.CoverDate}).
			Where(sq.GtOrEq{"feed_versions.latest_calendar_date": cmd.CoverDate})
	}
	if len(cmd.FVIDs) > 0 {
		// Explicitly specify fvids
		q = q.Where(sq.Eq{"feed_versions.id": cmd.FVIDs})
	}
	if len(cmd.FVSHA1) > 0 {
		// Explicitly specify fv sha1
		q = q.Where(sq.Eq{"feed_versions.sha1": cmd.FVSHA1})
	}
	if cmd.Limit > 0 {
		// Max feeds
		q = q.Limit(uint64(cmd.Limit))
	}
	qstr, qargs, err := q.ToSql()
	if err != nil {
		return err
	}
	qrs := []int{}
	err = cmd.Adapter.Select(&qrs, qstr, qargs...)
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
			FeedVersionID:        fvid,
			Directory:            cmd.ImportOptions.Directory,
			S3:                   cmd.ImportOptions.S3,
			Extensions:           cmd.ImportOptions.Extensions,
			Activate:             cmd.ImportOptions.Activate,
			InterpolateStopTimes: cmd.ImportOptions.InterpolateStopTimes,
			CreateMissingShapes:  cmd.ImportOptions.CreateMissingShapes,
		}
	}
	close(jobs)
	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < cmd.Workers; w++ {
		wg.Add(1)
		go dmfrImportWorker(w, cmd.Adapter, cmd.DryRun, jobs, results, &wg)
	}
	wg.Wait()
	return nil
}

func dmfrImportWorker(id int, adapter tldb.Adapter, dryrun bool, jobs <-chan ImportOptions, results chan<- ImportResult, wg *sync.WaitGroup) {
	type qr struct {
		FeedVersionID   int
		FeedID          int
		FeedOnestopID   string
		FeedVersionSHA1 string
	}
	for opts := range jobs {
		q := qr{}
		if err := adapter.Get(&q, "SELECT feed_versions.id as feed_version_id, feed_versions.feed_id as feed_id, feed_versions.sha1 as feed_version_sha1, current_feeds.onestop_id as feed_onestop_id FROM feed_versions INNER JOIN current_feeds ON current_feeds.id = feed_versions.feed_id WHERE feed_versions.id = ?", opts.FeedVersionID); err != nil {
			log.Error("Could not get details for FeedVersion %d", opts.FeedVersionID)
			continue
		}
		if dryrun {
			log.Info("Feed %s (id:%d): FeedVersion %s (id:%d): dry-run", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID)
			continue
		}
		log.Info("Feed %s (id:%d): FeedVersion %s (id:%d): begin", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID)
		t := time.Now()
		result, err := MainImportFeedVersion(adapter, opts)
		t2 := float64(time.Now().UnixNano()-t.UnixNano()) / 1e9 // 1000000000.0
		if err != nil {
			log.Error("Feed %s (id:%d): FeedVersion %s (id:%d): critical failure, rolled back: %s (t:%0.2fs)", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, result.FeedVersionImport.ExceptionLog, t2)
		} else if result.FeedVersionImport.Success {
			log.Info("Feed %s (id:%d): FeedVersion %s (id:%d): success: count %v errors: %v referrors: %v (t:%0.2fs)", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, result.FeedVersionImport.EntityCount, result.FeedVersionImport.SkipEntityErrorCount, result.FeedVersionImport.SkipEntityReferenceCount, t2)
		} else {
			log.Info("Feed %s (id:%d): FeedVersion %s (id:%d): error: %s, (t:%0.2fs)", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, result.FeedVersionImport.ExceptionLog, t2)
		}
		results <- result
	}
	wg.Done()
}

func getFileLines(fn string) ([]string, error) {
	ret := []string{}
	file, err := os.Open(fn)
	if err != nil {
		return ret, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if t := scanner.Text(); t != "" {
			ret = append(ret, strings.TrimSpace(t))
		}
	}
	if err := scanner.Err(); err != nil {
		return ret, err
	}
	return ret, nil
}
