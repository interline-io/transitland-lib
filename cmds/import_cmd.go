package cmds

import (
	"context"
	"errors"
	"fmt"
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

type ImportCommandResult struct {
	Result     importer.Result
	FatalError error
}

// ImportCommand imports FeedVersions into a database.
type ImportCommand struct {
	Options      importer.Options
	Workers      int
	Limit        int
	Fail         bool
	DBURL        string
	CoverDate    string
	FetchedSince string
	Latest       bool
	DryRun       bool
	FeedIDs      []string
	FVIDs        []string
	FVSHA1       []string
	Results      []ImportCommandResult
	Adapter      tldb.Adapter // allow for mocks
	// internal
	fvidfile   string
	fvsha1file string
}

func (cmd *ImportCommand) HelpDesc() (string, string) {
	return "Import feed versions", "Use after the `fetch` command"
}

func (cmd *ImportCommand) HelpArgs() string {
	return "[flags] [feeds...]"
}

func (cmd *ImportCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringSliceVar(&cmd.Options.Extensions, "ext", nil, "Include GTFS Extension")
	fl.StringSliceVar(&cmd.FVIDs, "fvid", nil, "Import specific feed version ID")
	fl.StringVar(&cmd.fvidfile, "fvid-file", "", "Specify feed version IDs in file, one per line; equivalent to multiple --fvid")
	fl.StringVar(&cmd.fvsha1file, "fv-sha1-file", "", "Specify feed version IDs by SHA1 in file, one per line")
	fl.StringSliceVar(&cmd.FVSHA1, "fv-sha1", nil, "Feed version SHA1")
	fl.IntVar(&cmd.Workers, "workers", 1, "Worker threads")
	fl.IntVar(&cmd.Limit, "limit", 0, "Import at most n feeds")
	fl.BoolVar(&cmd.Fail, "fail", false, "Exit with error code if any fetch is not successful")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.StringVar(&cmd.Options.Storage, "storage", ".", "Storage location; can be s3://... az://... or path to a directory")
	fl.StringVar(&cmd.CoverDate, "date", "", "Service on date")
	fl.StringVar(&cmd.FetchedSince, "fetched-since", "", "Fetched since")
	fl.BoolVar(&cmd.Latest, "latest", false, "Only import latest feed version available for each feed")
	fl.BoolVar(&cmd.DryRun, "dryrun", false, "Dry run; print feeds that would be imported and exit")
	fl.BoolVar(&cmd.Options.Activate, "activate", false, "Set as active feed version after import")
	// Copy options
	fl.Float64Var(&cmd.Options.SimplifyShapes, "simplify-shapes", 0.0, "Simplify shapes with this tolerance (ex. 0.000005)")
	fl.BoolVar(&cmd.Options.InterpolateStopTimes, "interpolate-stop-times", false, "Interpolate missing StopTime arrival/departure values")
	fl.BoolVar(&cmd.Options.DeduplicateJourneyPatterns, "deduplicate-stop-times", false, "Deduplicate StopTimes using Journey Patterns")
	fl.BoolVar(&cmd.Options.CreateMissingShapes, "create-missing-shapes", false, "Create missing Shapes from Trip stop-to-stop geometries")
	fl.BoolVar(&cmd.Options.SimplifyCalendars, "simplify-calendars", false, "Attempt to simplify CalendarDates into regular Calendars")
	fl.BoolVar(&cmd.Options.NormalizeTimezones, "normalize-timezones", false, "Normalize timezones and apply default stop timezones based on agency and parent stops")
}

// Parse command line flags
func (cmd *ImportCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	cmd.FeedIDs = fl.Args()
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
		if len(cmd.FVIDs) == 0 {
			return errors.New("--fvid-file specified but no lines were read")
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
		if len(cmd.FVSHA1) == 0 {
			return errors.New("--fv-sha1-file specified but no lines were read")
		}
	}
	return nil
}

// Run this command) Run(ctx context.Context) error
func (cmd *ImportCommand) Run(ctx context.Context) error {
	if cmd.Workers < 1 {
		cmd.Workers = 1
	}
	if cmd.Adapter == nil {
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return err
		}
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
		Where("feed_versions.sha1 <> ''").
		Where("feed_versions.file <> ''").
		OrderBy("feed_versions.id desc")
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
	log.Infof("Importing %d feed versions", len(qrs))
	jobs := make(chan importer.Options, len(qrs))
	results := make(chan ImportCommandResult, len(qrs))
	for _, fvid := range qrs {
		jobs <- importer.Options{
			FeedVersionID: fvid,
			Storage:       cmd.Options.Storage,
			Activate:      cmd.Options.Activate,
			Options:       cmd.Options.Options,
		}
	}
	close(jobs)

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < cmd.Workers; w++ {
		wg.Add(1)
		go dmfrImportWorker(ctx, cmd.Adapter, cmd.DryRun, jobs, results, &wg)
	}
	wg.Wait()
	close(results)

	// Check results
	var fatalError error
	for result := range results {
		cmd.Results = append(cmd.Results, result)
		if result.FatalError != nil {
			fatalError = result.FatalError
		} else if fvi := result.Result.FeedVersionImport; !fvi.Success {
			if cmd.Fail {
				fatalError = fmt.Errorf("import failed: %s", fvi.ExceptionLog)
			}
		}
	}
	if fatalError != nil {
		log.Infof("Exiting with error because at least one import had fatal error: %s", fatalError.Error())
		return fatalError
	}
	return nil
}

func dmfrImportWorker(ctx context.Context, adapter tldb.Adapter, dryrun bool, jobs <-chan importer.Options, results chan<- ImportCommandResult, wg *sync.WaitGroup) {
	type qr struct {
		FeedVersionID   int
		FeedID          int
		FeedOnestopID   string
		FeedVersionSHA1 string
	}
	for opts := range jobs {
		q := qr{}
		if err := adapter.Get(&q, "SELECT feed_versions.id as feed_version_id, feed_versions.feed_id as feed_id, feed_versions.sha1 as feed_version_sha1, current_feeds.onestop_id as feed_onestop_id FROM feed_versions INNER JOIN current_feeds ON current_feeds.id = feed_versions.feed_id WHERE feed_versions.id = ?", opts.FeedVersionID); err != nil {
			log.Errorf("Could not get details for FeedVersion %d", opts.FeedVersionID)
			continue
		}
		if dryrun {
			log.Infof("Feed %s (id:%d): FeedVersion %s (id:%d): dry-run", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID)
			continue
		}
		log.Infof("Feed %s (id:%d): FeedVersion %s (id:%d): begin", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID)
		t := time.Now()
		result, err := importer.ImportFeedVersion(ctx, adapter, opts)
		t2 := float64(time.Now().UnixNano()-t.UnixNano()) / 1e9 // 1000000000.0
		if err != nil {
			log.Errorf("Feed %s (id:%d): FeedVersion %s (id:%d): critical failure, rolled back: %s (t:%0.2fs)", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, result.FeedVersionImport.ExceptionLog, t2)
		} else if result.FeedVersionImport.Success {
			log.Infof("Feed %s (id:%d): FeedVersion %s (id:%d): success: count %v errors: %v referrors: %v (t:%0.2fs)", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, result.FeedVersionImport.EntityCount, result.FeedVersionImport.SkipEntityErrorCount, result.FeedVersionImport.SkipEntityReferenceCount, t2)
		} else {
			log.Infof("Feed %s (id:%d): FeedVersion %s (id:%d): error: %s, (t:%0.2fs)", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, result.FeedVersionImport.ExceptionLog, t2)
		}
		results <- ImportCommandResult{
			Result:     result,
			FatalError: err,
		}
	}
	wg.Done()
}
