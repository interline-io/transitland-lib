package cmds

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/fetch"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/validator"
	"github.com/spf13/pflag"
)

type FetchCommandResult struct {
	Result                     fetch.Result
	FeedVersion                *dmfr.FeedVersion
	FeedVersionValidatorResult *validator.Result
	FatalError                 error
}

// FetchJob specifies a single feed fetch operation.
// This allows programmatic specification of multiple fetches with different URLs.
type FetchJob struct {
	FeedID  string // Feed identifier (onestop_id)
	FeedURL string // URL to fetch from (optional; if empty, looks up from database)
}

// FetchCommand fetches feeds defined a DMFR database.
type FetchCommand struct {
	Options     fetch.StaticFetchOptions
	SecretsFile string
	CreateFeed  bool
	Workers     int
	Fail        bool
	Limit       int
	DBURL       string
	DryRun      bool
	FeedIDs     []string
	FetchJobs   []FetchJob // Programmatic list of fetch operations; takes precedence over FeedIDs
	Results     []FetchCommandResult
	Adapter     tldb.Adapter // allow for mocks
	fetchedAt   string
}

func (cmd *FetchCommand) HelpDesc() (string, string) {
	return "Fetch GTFS data and create feed versions", "Use after the `sync` command."
}

func (cmd *FetchCommand) HelpArgs() string {
	return "[flags] [feeds...]"
}

func (cmd *FetchCommand) AddFlags(fl *pflag.FlagSet) {
	fl.BoolVar(&cmd.CreateFeed, "create-feed", false, "Create feed record if not found")
	fl.StringVar(&cmd.Options.FeedURL, "feed-url", "", "Manually fetch a single URL; you must specify exactly one feed_id")
	fl.StringVar(&cmd.fetchedAt, "fetched-at", "", "Manually specify fetched_at value, e.g. 2020-02-06T12:34:56Z")
	fl.StringVar(&cmd.SecretsFile, "secrets", "", "Path to DMFR Secrets file")
	fl.IntVar(&cmd.Workers, "workers", 1, "Worker threads")
	fl.IntVar(&cmd.Limit, "limit", 0, "Maximum number of feeds to fetch")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.BoolVar(&cmd.Fail, "fail", false, "Exit with error code if any fetch is not successful")
	fl.BoolVar(&cmd.DryRun, "dry-run", false, "Dry run; print feeds that would be imported and exit")
	fl.BoolVar(&cmd.Options.StrictValidation, "strict", false, "Reject feeds with validation errors")
	fl.BoolVar(&cmd.Options.AllowFTPFetch, "allow-ftp-fetch", false, "Allow fetching from FTP urls")
	fl.BoolVar(&cmd.Options.AllowS3Fetch, "allow-s3-fetch", false, "Allow fetching from S3 urls")
	fl.BoolVar(&cmd.Options.AllowLocalFetch, "allow-local-fetch", false, "Allow fetching from filesystem directories/zip files")
	fl.BoolVar(&cmd.Options.SaveValidationReport, "validation-report", false, "Save validation report")
	fl.StringVar(&cmd.Options.ValidationReportStorage, "validation-report-storage", "", "Storage path for saving validation report JSON")
	fl.StringVar(&cmd.Options.Storage, "storage", ".", "Storage destination; can be s3://... az://... or path to a directory")
}

func (cmd *FetchCommand) Parse(args []string) error {
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	cmd.FeedIDs = args
	return nil
}

// Run executes this command.
func (cmd *FetchCommand) Run(ctx context.Context) error {
	// Init
	if cmd.Workers < 1 {
		cmd.Workers = 1
	}
	if cmd.fetchedAt != "" {
		t, err := time.Parse(time.RFC3339Nano, cmd.fetchedAt)
		if err != nil {
			return err
		}
		cmd.Options.FetchedAt = t
	}
	if cmd.SecretsFile != "" {
		r, err := dmfr.LoadAndParseRegistry(cmd.SecretsFile)
		if err != nil {
			return err
		}
		cmd.Options.Secrets = r.Secrets
	}
	if cmd.Options.FetchedAt.IsZero() {
		cmd.Options.FetchedAt = time.Now()
	}
	cmd.Options.FetchedAt = cmd.Options.FetchedAt.UTC()
	if cmd.Options.FeedURL != "" && len(cmd.FeedIDs) != 1 {
		return errors.New("you must specify exactly one feed_id when using -fetch-url")
	}

	// Get adapter
	if cmd.Adapter == nil {
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return err
		}
		cmd.Adapter = writer.Adapter
		defer writer.Close()
	}
	adapter := cmd.Adapter

	// Populate FetchJobs from various input sources
	// FetchJobs and FeedIDs are additive; database query only when neither is specified
	if len(cmd.FeedIDs) > 0 {
		// FeedIDs provided, convert to FetchJobs
		for _, feedID := range cmd.FeedIDs {
			job := FetchJob{FeedID: feedID}
			// If single FeedURL specified via options, apply it to the single feed
			if cmd.Options.FeedURL != "" {
				job.FeedURL = cmd.Options.FeedURL
			}
			cmd.FetchJobs = append(cmd.FetchJobs, job)
		}
	}
	if len(cmd.FetchJobs) == 0 {
		// No FetchJobs or FeedIDs, query database for all feeds
		q := adapter.Sqrl().
			Select("*").
			From("current_feeds").
			Where("deleted_at IS NULL").
			Where("spec = ?", "gtfs")
		if cmd.Limit > 0 {
			q = q.Limit(uint64(cmd.Limit))
		}
		qstr, qargs, err := q.ToSql()
		if err != nil {
			return err
		}
		feeds := []dmfr.Feed{}
		err = adapter.Select(ctx, &feeds, qstr, qargs...)
		if err != nil {
			return err
		}
		for _, feed := range feeds {
			if feed.URLs.StaticCurrent != "" {
				cmd.FetchJobs = append(cmd.FetchJobs, FetchJob{
					FeedID:  feed.FeedID,
					FeedURL: feed.URLs.StaticCurrent,
				})
			}
		}
	}

	// Convert FetchJobs to internal fetchJob queue with full options
	var toFetch []fetchJob
	for _, job := range cmd.FetchJobs {
		// Get feed, create if not present and CreateFeed is specified
		feed := dmfr.Feed{}
		if err := adapter.Get(ctx, &feed, `SELECT * FROM current_feeds WHERE onestop_id = ?`, job.FeedID); err == sql.ErrNoRows && cmd.CreateFeed {
			feed.FeedID = job.FeedID
			feed.Spec = "gtfs"
			if feed.ID, err = adapter.Insert(ctx, &feed); err != nil {
				return err
			}
		} else if err != nil {
			return fmt.Errorf("problem with feed '%s': %s", job.FeedID, err.Error())
		}
		// Create feed state if not exists
		if _, err := stats.GetFeedState(ctx, adapter, feed.ID); err != nil {
			return err
		}
		// Prepare options for this fetch
		opts := cmd.Options // copy global options
		opts.FeedID = feed.ID
		if job.FeedURL != "" {
			opts.URLType = "manual"
			opts.FeedURL = job.FeedURL
		} else {
			opts.URLType = "static_current"
			opts.FeedURL = feed.URLs.StaticCurrent
		}
		toFetch = append(toFetch, fetchJob{OnestopID: feed.FeedID, StaticFetchOptions: opts})
	}

	///////////////
	// Here we go
	log.For(ctx).Info().Msgf("Fetching %d feeds", len(cmd.FetchJobs))
	jobs := make(chan fetchJob, len(cmd.FetchJobs))
	results := make(chan FetchCommandResult, len(cmd.FetchJobs))
	for _, opts := range toFetch {
		jobs <- opts
	}
	close(jobs)

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < cmd.Workers; w++ {
		wg.Add(1)
		go fetchWorker(ctx, cmd.Adapter, cmd.DryRun, jobs, results, &wg)
	}
	wg.Wait()
	close(results)

	// Check results
	var fatalError error
	fetchNew := 0
	fetchFound := 0
	fetchErrs := 0
	for result := range results {
		cmd.Results = append(cmd.Results, result)
		if result.FatalError != nil {
			// A fatal error occurred, always exit with error
			fatalError = result.FatalError
		} else if result.Result.FetchError != nil {
			fetchErrs++
			if cmd.Fail {
				// Exit with error if any fetch is not successful
				fatalError = result.Result.FetchError
			}
		} else if result.Result.Found {
			fetchFound++
		} else {
			fetchNew++
		}
	}
	log.For(ctx).Info().Msgf("Existing: %d New: %d Errors: %d", fetchFound, fetchNew, fetchErrs)
	if fatalError != nil {
		log.For(ctx).Info().Msgf("Exiting with error because at least one fetch had fatal error: %s", fatalError.Error())
		return fatalError
	}
	return nil
}

type fetchJob struct {
	OnestopID string
	fetch.StaticFetchOptions
}

func fetchWorker(ctx context.Context, adapter tldb.Adapter, DryRun bool, jobs <-chan fetchJob, results chan<- FetchCommandResult, wg *sync.WaitGroup) {
	for job := range jobs {
		// Start
		log.For(ctx).Info().Msgf("Feed %s: start", job.OnestopID)
		if DryRun {
			log.For(ctx).Info().Msgf("Feed %s: dry-run", job.OnestopID)
			continue
		}

		// Fetch
		var result fetch.StaticFetchResult
		t := time.Now()
		fatalError := adapter.Tx(func(atx tldb.Adapter) error {
			var fatalError error
			result, fatalError = fetch.StaticFetch(ctx, atx, job.StaticFetchOptions)
			return fatalError
		})
		t2 := float64(time.Now().UnixNano()-t.UnixNano()) / 1e9 // 1000000000.0

		// Log result
		fv := result.FeedVersion
		if fatalError != nil {
			log.For(ctx).Error().Err(fatalError).Msgf("Feed %s (id:%d): url: %s critical error: %s (t:%0.2fs)", job.OnestopID, job.Options.FeedID, result.URL, fatalError.Error(), t2)
		} else if result.FetchError != nil {
			log.For(ctx).Error().Err(result.FetchError).Msgf("Feed %s (id:%d): url: %s fetch error: %s (t:%0.2fs)", job.OnestopID, job.Options.FeedID, result.URL, result.FetchError.Error(), t2)
		} else if fv != nil && result.Found {
			log.For(ctx).Info().Msgf("Feed %s (id:%d): url: %s found sha1: %s (id:%d) (t:%0.2fs)", job.OnestopID, job.Options.FeedID, fv.URL, fv.SHA1, fv.ID, t2)
		} else if fv != nil {
			log.For(ctx).Info().Msgf("Feed %s (id:%d): url: %s new: %s (id:%d) (t:%0.2fs)", job.OnestopID, job.Options.FeedID, fv.URL, fv.SHA1, fv.ID, t2)
		} else {
			log.For(ctx).Info().Msgf("Feed %s (id:%d): url: %s invalid result (t:%0.2fs)", job.OnestopID, job.Options.FeedID, result.URL, t2)
		}
		results <- FetchCommandResult{
			Result:                     result.Result,
			FeedVersion:                result.FeedVersion,
			FeedVersionValidatorResult: result.FeedVersionValidatorResult,
			FatalError:                 fatalError,
		}
	}
	wg.Done()
}
