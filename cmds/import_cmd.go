package cmds

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/importer"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/tldb"
	sq "github.com/irees/squirrel"
	"github.com/spf13/pflag"
)

type ImportCommandResult struct {
	Result     importer.Result
	FatalError error
}

// ImportJob specifies a single feed version import operation.
type ImportJob struct {
	FeedVersionID int
}

// ImportCommand imports FeedVersions into a database.
type ImportCommand struct {
	FeedIDs    []string    // Filter by feed onestop_id
	FVSHA1     []string    // Filter by feed version SHA1
	ImportJobs []ImportJob // Programmatic list of import operations
	Options    importer.Options
	Workers    int
	Limit      int
	Fail       bool
	DBURL      string
	Latest     bool
	DryRun     bool
	Results    []ImportCommandResult
	Adapter    tldb.Adapter // allow for mocks
	// internal
	fvids           []string
	fvidfile        string
	fvsha1file      string
	dmfrFile        string
	errorThresholds []string
}

func (cmd *ImportCommand) HelpDesc() (string, string) {
	return "Import feed versions", "Use after the `fetch` command"
}

func (cmd *ImportCommand) HelpArgs() string {
	return "[flags] [feeds...]"
}

func (cmd *ImportCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringSliceVar(&cmd.Options.ExtensionDefs, "ext", nil, "Include GTFS Extension")
	fl.StringSliceVar(&cmd.fvids, "fvid", nil, "Import specific feed version ID")
	fl.StringVar(&cmd.fvidfile, "fvid-file", "", "Specify feed version IDs in file, one per line; equivalent to multiple --fvid")
	fl.StringVar(&cmd.fvsha1file, "fv-sha1-file", "", "Specify feed version IDs by SHA1 in file, one per line")
	fl.StringSliceVar(&cmd.FVSHA1, "fv-sha1", nil, "Feed version SHA1")
	fl.IntVar(&cmd.Workers, "workers", 1, "Worker threads")
	fl.IntVar(&cmd.Limit, "limit", 0, "Import at most n feeds")
	fl.BoolVar(&cmd.Fail, "fail", false, "Exit with error code if any fetch is not successful")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.StringVar(&cmd.Options.Storage, "storage", ".", "Storage location; can be s3://... az://... or path to a directory")
	fl.BoolVar(&cmd.Latest, "latest", false, "Only import latest feed version available for each feed")
	fl.BoolVar(&cmd.DryRun, "dryrun", false, "Dry run; print feeds that would be imported and exit")
	fl.BoolVar(&cmd.Options.Activate, "activate", false, "Set as active feed version after import")
	fl.StringVar(&cmd.dmfrFile, "dmfr", "", "Filter by feed IDs in DMFR file; equivalent to specifying feed IDs as arguments")
	// Copy options
	fl.Float64Var(&cmd.Options.SimplifyShapes, "simplify-shapes", 0.0, "Simplify shapes with this tolerance (ex. 0.000005)")
	fl.BoolVar(&cmd.Options.InterpolateStopTimes, "interpolate-stop-times", false, "Interpolate missing StopTime arrival/departure values")
	fl.BoolVar(&cmd.Options.DeduplicateJourneyPatterns, "deduplicate-stop-times", false, "Deduplicate StopTimes using Journey Patterns")
	fl.BoolVar(&cmd.Options.CreateMissingShapes, "create-missing-shapes", false, "Create missing Shapes from Trip stop-to-stop geometries")
	fl.BoolVar(&cmd.Options.SimplifyCalendars, "simplify-calendars", false, "Attempt to simplify CalendarDates into regular Calendars")
	fl.BoolVar(&cmd.Options.NormalizeTimezones, "normalize-timezones", false, "Normalize timezones and apply default stop timezones based on agency and parent stops")
	fl.StringSliceVar(&cmd.errorThresholds, "error-threshold", nil, "Fail import if file exceeds error percentage; format: 'filename:percent' or '*:percent' for default (e.g., 'stops.txt:5' or '*:10')")
}

// Parse command line flags
func (cmd *ImportCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	cmd.FeedIDs = fl.Args()
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	// Load feed IDs from DMFR file
	if cmd.dmfrFile != "" {
		reg, err := dmfr.LoadAndParseRegistry(cmd.dmfrFile)
		if err != nil {
			return err
		}
		for _, feed := range reg.Feeds {
			cmd.FeedIDs = append(cmd.FeedIDs, feed.FeedID)
		}
	}
	// Read fvid file and add to fvids
	if cmd.fvidfile != "" {
		lines, err := tlcli.ReadFileLines(cmd.fvidfile)
		if err != nil {
			return err
		}
		for _, line := range lines {
			if line != "" {
				cmd.fvids = append(cmd.fvids, line)
			}
		}
	}
	// Convert fvids to ImportJobs
	for _, fvidStr := range cmd.fvids {
		fvid, err := strconv.Atoi(fvidStr)
		if err != nil {
			return fmt.Errorf("invalid feed version ID '%s': %w", fvidStr, err)
		}
		cmd.ImportJobs = append(cmd.ImportJobs, ImportJob{FeedVersionID: fvid})
	}
	// Read fvsha1 file
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
	if len(cmd.errorThresholds) > 0 {
		thresholds, err := parseErrorThresholds(cmd.errorThresholds)
		if err != nil {
			return err
		}
		cmd.Options.ErrorThreshold = thresholds
	}
	return nil
}

// Run this command
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

	// Resolve SHA1s to ImportJobs
	if len(cmd.FVSHA1) > 0 {
		qstr, qargs, err := cmd.Adapter.Sqrl().
			Select("id").
			From("feed_versions").
			Where(sq.Eq{"sha1": cmd.FVSHA1}).
			ToSql()
		if err != nil {
			return err
		}
		var fvids []int
		if err := cmd.Adapter.Select(ctx, &fvids, qstr, qargs...); err != nil {
			return err
		}
		for _, fvid := range fvids {
			cmd.ImportJobs = append(cmd.ImportJobs, ImportJob{FeedVersionID: fvid})
		}
	}

	// If no ImportJobs specified, query database for feed versions to import
	// FeedIDs are additive with existing ImportJobs
	if len(cmd.FeedIDs) > 0 || len(cmd.ImportJobs) == 0 {
		q := cmd.Adapter.Sqrl().
			Select("feed_versions.id").
			From("feed_versions").
			Join("current_feeds ON current_feeds.id = feed_versions.feed_id").
			Where("current_feeds.deleted_at IS NULL").
			Where("feed_versions.deleted_at IS NULL")
		if len(cmd.FeedIDs) > 0 {
			q = q.Where(sq.Eq{"onestop_id": cmd.FeedIDs})
		}
		qstr, qargs, err := q.ToSql()
		if err != nil {
			return err
		}
		var qrs []int
		if err := cmd.Adapter.Select(ctx, &qrs, qstr, qargs...); err != nil {
			return err
		}
		for _, fvid := range qrs {
			cmd.ImportJobs = append(cmd.ImportJobs, ImportJob{FeedVersionID: fvid})
		}
	}

	// Final filtering: validate, dedupe, remove already imported, apply latest and limit
	if len(cmd.ImportJobs) > 0 {
		var fvids []int
		for _, job := range cmd.ImportJobs {
			fvids = append(fvids, job.FeedVersionID)
		}
		q := cmd.Adapter.Sqrl().
			Select("feed_versions.id").
			From("feed_versions").
			LeftJoin("feed_version_gtfs_imports ON feed_versions.id = feed_version_gtfs_imports.feed_version_id").
			Where(sq.Eq{"feed_versions.id": fvids}).
			Where("feed_version_gtfs_imports.id IS NULL").
			Where("feed_versions.sha1 <> ''").
			Where("feed_versions.file <> ''").
			OrderBy("feed_versions.id desc")
		if cmd.Latest {
			q = q.
				Join("(SELECT id, ROW_NUMBER() OVER (PARTITION BY feed_id ORDER BY created_at DESC) AS rank FROM feed_versions) latest ON latest.id = feed_versions.id").
				Where("latest.rank = 1")
		}
		if cmd.Limit > 0 {
			q = q.Limit(uint64(cmd.Limit))
		}
		qstr, qargs, err := q.ToSql()
		if err != nil {
			return err
		}
		var validFvids []int
		if err := cmd.Adapter.Select(ctx, &validFvids, qstr, qargs...); err != nil {
			return err
		}
		cmd.ImportJobs = nil
		for _, fvid := range validFvids {
			cmd.ImportJobs = append(cmd.ImportJobs, ImportJob{FeedVersionID: fvid})
		}
	}

	///////////////
	// Here we go
	log.For(ctx).Info().Msgf("Importing %d feed versions", len(cmd.ImportJobs))
	jobs := make(chan importer.Options, len(cmd.ImportJobs))
	results := make(chan ImportCommandResult, len(cmd.ImportJobs))
	for _, job := range cmd.ImportJobs {
		jobs <- importer.Options{
			FeedVersionID: job.FeedVersionID,
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
		log.For(ctx).Error().Err(fatalError).Msg("Exiting because at least one import had fatal error")
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
		qstr, qargs, err := adapter.Sqrl().
			Select(
				"feed_versions.id as feed_version_id",
				"feed_versions.feed_id as feed_id",
				"feed_versions.sha1 as feed_version_sha1",
				"current_feeds.onestop_id as feed_onestop_id",
			).
			From("feed_versions").
			Join("current_feeds ON current_feeds.id = feed_versions.feed_id").
			Where(sq.Eq{"feed_versions.id": opts.FeedVersionID}).
			ToSql()
		if err != nil {
			log.For(ctx).Error().Err(err).Int("feed_version_id", opts.FeedVersionID).Msg("could not build query")
			continue
		}
		q := qr{}
		if err := adapter.Get(ctx, &q, qstr, qargs...); err != nil {
			log.For(ctx).Error().Err(err).Int("feed_version_id", opts.FeedVersionID).Msg("could not get details")
			continue
		}
		jobLog := log.For(ctx).With().
			Str("feed_onestop_id", q.FeedOnestopID).
			Int("feed_id", q.FeedID).
			Str("feed_version_sha1", q.FeedVersionSHA1).
			Int("feed_version_id", q.FeedVersionID).
			Logger()
		if dryrun {
			jobLog.Info().Msg("dry-run")
			continue
		}
		jobLog.Info().Msg("begin")
		jobCtx := log.WithLogger(ctx, jobLog)
		t := time.Now()
		result, err := importer.ImportFeedVersion(jobCtx, adapter, opts)
		t2 := float64(time.Now().UnixNano()-t.UnixNano()) / 1e9 // 1000000000.0
		if err != nil {
			jobLog.Error().Err(err).Float64("duration", t2).Msg("critical failure, rolled back")
		} else if result.FeedVersionImport.Success {
			jobLog.Info().
				Float64("duration", t2).
				Interface("entity_count", result.FeedVersionImport.EntityCount).
				Interface("skip_entity_error_count", result.FeedVersionImport.SkipEntityErrorCount).
				Interface("skip_entity_reference_count", result.FeedVersionImport.SkipEntityReferenceCount).
				Msg("success")
		} else {
			jobLog.Error().Float64("duration", t2).Str("exception", result.FeedVersionImport.ExceptionLog).Msg("import failed")
		}
		results <- ImportCommandResult{
			Result:     result,
			FatalError: err,
		}
	}
	wg.Done()
}
