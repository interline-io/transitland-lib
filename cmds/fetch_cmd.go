package cmds

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/fetch"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tldbutil"
	"github.com/interline-io/transitland-lib/validator"
	"github.com/spf13/pflag"
)

type FetchCommandResult struct {
	Result           fetch.Result
	FeedVersion      *tl.FeedVersion
	ValidationResult *validator.Result
	FatalError       error
}

// FetchCommand fetches feeds defined a DMFR database.
type FetchCommand struct {
	Options     fetch.Options
	CreateFeed  bool
	Workers     int
	Fail        bool
	Limit       int
	DBURL       string
	DryRun      bool
	FeedIDs     []string
	Results     []FetchCommandResult
	Adapter     tldb.Adapter // allow for mocks
	fetchedAt   string
	secretsFile string
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
	fl.StringVar(&cmd.secretsFile, "secrets", "", "Path to DMFR Secrets file")
	fl.IntVar(&cmd.Workers, "workers", 1, "Worker threads")
	fl.IntVar(&cmd.Limit, "limit", 0, "Maximum number of feeds to fetch")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.BoolVar(&cmd.Fail, "fail", false, "Exit with error code if any fetch is not successful")
	fl.BoolVar(&cmd.DryRun, "dry-run", false, "Dry run; print feeds that would be imported and exit")
	fl.BoolVar(&cmd.Options.IgnoreDuplicateContents, "ignore-duplicate-contents", false, "Allow duplicate internal SHA1 contents")
	fl.BoolVar(&cmd.Options.AllowFTPFetch, "allow-ftp-fetch", false, "Allow fetching from FTP urls")
	fl.BoolVar(&cmd.Options.AllowS3Fetch, "allow-s3-fetch", false, "Allow fetching from S3 urls")
	fl.BoolVar(&cmd.Options.AllowLocalFetch, "allow-local-fetch", false, "Allow fetching from filesystem directories/zip files")
	fl.BoolVar(&cmd.Options.SaveValidationReport, "validation-report", false, "Save validation report")
	fl.StringVar(&cmd.Options.ValidationReportStorage, "validation-report-storage", "", "Storage path for saving validation report JSON")
	fl.StringVar(&cmd.Options.Storage, "storage", ".", "Storage destination; can be s3://... az://... or path to a directory")
}

func (cmd *FetchCommand) Parse(args []string) error {
	if cmd.Workers < 1 {
		cmd.Workers = 1
	}
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	cmd.FeedIDs = args
	if cmd.fetchedAt != "" {
		t, err := time.Parse(time.RFC3339Nano, cmd.fetchedAt)
		if err != nil {
			return err
		}
		cmd.Options.FetchedAt = t
	}
	if cmd.secretsFile != "" {
		r, err := dmfr.LoadAndParseRegistry(cmd.secretsFile)
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
	// Get feeds
	if cmd.Adapter == nil {
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return err
		}
		cmd.Adapter = writer.Adapter
		defer writer.Close()
	}
	if len(cmd.FeedIDs) == 0 {
		q := cmd.Adapter.Sqrl().
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
		feeds := []tl.Feed{}
		err = cmd.Adapter.Select(&feeds, qstr, qargs...)
		if err != nil {
			return err
		}
		for _, feed := range feeds {
			if feed.URLs.StaticCurrent != "" {
				cmd.FeedIDs = append(cmd.FeedIDs, feed.FeedID)
			}
		}
	}
	return nil
}

// Run executes this command.
func (cmd *FetchCommand) Run() error {
	// Check feeds
	adapter := cmd.Adapter
	var toFetch []fetchJob
	for _, osid := range cmd.FeedIDs {
		// Get feed, create if not present and FeedCreate is specified
		feed := tl.Feed{}
		if err := adapter.Get(&feed, `SELECT * FROM current_feeds WHERE onestop_id = ?`, osid); err == sql.ErrNoRows && cmd.CreateFeed {
			feed.FeedID = osid
			feed.Spec = "gtfs"
			if feed.ID, err = adapter.Insert(&feed); err != nil {
				return err
			}
		} else if err != nil {
			return fmt.Errorf("problem with feed '%s': %s", osid, err.Error())
		}
		// Create feed state if not exists
		if _, err := tldbutil.GetFeedState(adapter, feed.ID); err != nil {
			return err
		}
		// Prepare options for this fetch
		opts := fetch.Options{
			FeedID:                  feed.ID,
			FeedURL:                 cmd.Options.FeedURL,
			FetchedAt:               cmd.Options.FetchedAt,
			URLType:                 cmd.Options.URLType,
			Storage:                 cmd.Options.Storage,
			IgnoreDuplicateContents: cmd.Options.IgnoreDuplicateContents,
			AllowFTPFetch:           cmd.Options.AllowFTPFetch,
			AllowS3Fetch:            cmd.Options.AllowS3Fetch,
			AllowLocalFetch:         cmd.Options.AllowLocalFetch,
			Secrets:                 cmd.Options.Secrets,
			SaveValidationReport:    cmd.Options.SaveValidationReport,
			ValidationReportStorage: cmd.Options.ValidationReportStorage,
		}
		opts.URLType = "manual"
		if opts.FeedURL == "" {
			opts.URLType = "static_current"
			opts.FeedURL = feed.URLs.StaticCurrent
		}
		toFetch = append(toFetch, fetchJob{OnestopID: feed.FeedID, Options: opts})
	}

	///////////////
	// Here we go
	log.Infof("Fetching %d feeds", len(cmd.FeedIDs))
	jobs := make(chan fetchJob, len(cmd.FeedIDs))
	results := make(chan FetchCommandResult, len(cmd.FeedIDs))
	for _, opts := range toFetch {
		jobs <- opts
	}
	close(jobs)

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < cmd.Workers; w++ {
		wg.Add(1)
		go fetchWorker(cmd.Adapter, cmd.DryRun, jobs, results, &wg)
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
			fatalError = result.FatalError
		} else if result.Result.FetchError != nil {
			fetchErrs++
			if cmd.Fail {
				fatalError = result.Result.FetchError
			}
		} else if result.Result.Found {
			fetchFound++
		} else {
			fetchNew++
		}
	}
	log.Infof("Existing: %d New: %d Errors: %d", fetchFound, fetchNew, fetchErrs)
	if fatalError != nil {
		log.Infof("Exiting with error because at least one fetch had fatal error: %s", fatalError.Error())
		return fatalError
	}
	return nil
}

type fetchJob struct {
	OnestopID string
	fetch.Options
}

func fetchWorker(adapter tldb.Adapter, DryRun bool, jobs <-chan fetchJob, results chan<- FetchCommandResult, wg *sync.WaitGroup) {
	for job := range jobs {
		// Start
		log.Infof("Feed %s: start", job.OnestopID)
		if DryRun {
			log.Infof("Feed %s: dry-run", job.OnestopID)
			continue
		}

		// Fetch
		var result fetch.StaticFetchResult
		t := time.Now()
		fetchError := adapter.Tx(func(atx tldb.Adapter) error {
			var fetchError error
			result, fetchError = fetch.StaticFetch(atx, job.Options)
			return fetchError
		})
		t2 := float64(time.Now().UnixNano()-t.UnixNano()) / 1e9 // 1000000000.0

		// Log result
		fv := result.FeedVersion
		if fetchError != nil {
			log.Error().Err(fetchError).Msgf("Feed %s (id:%d): url: %s critical error: %s (t:%0.2fs)", job.OnestopID, job.Options.FeedID, result.URL, fetchError.Error(), t2)
		} else if result.FetchError != nil {
			log.Error().Err(result.FetchError).Msgf("Feed %s (id:%d): url: %s fetch error: %s (t:%0.2fs)", job.OnestopID, job.Options.FeedID, result.URL, result.FetchError.Error(), t2)
		} else if fv != nil && result.Found {
			log.Infof("Feed %s (id:%d): url: %s found sha1: %s (id:%d) (t:%0.2fs)", job.OnestopID, job.Options.FeedID, fv.URL, fv.SHA1, fv.ID, t2)
		} else if fv != nil {
			log.Infof("Feed %s (id:%d): url: %s new: %s (id:%d) (t:%0.2fs)", job.OnestopID, job.Options.FeedID, fv.URL, fv.SHA1, fv.ID, t2)
		} else {
			log.Infof("Feed %s (id:%d): url: %s invalid result (t:%0.2fs)", job.OnestopID, job.Options.FeedID, result.URL, t2)
		}
		results <- FetchCommandResult{
			Result:           result.Result,
			FeedVersion:      result.FeedVersion,
			ValidationResult: result.ValidationResult,
			FatalError:       fetchError,
		}
	}
	wg.Done()
}
