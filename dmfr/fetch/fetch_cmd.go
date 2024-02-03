package fetch

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
)

// Command fetches feeds defined a DMFR database.
type Command struct {
	Options    Options
	CreateFeed bool
	Workers    int
	Limit      int
	DBURL      string
	DryRun     bool
	FeedIDs    []string
	Adapter    tldb.Adapter
	Results    []StaticFetchResult
}

// Parse sets options from command line flags.
func (cmd *Command) Parse(args []string) error {
	secretsFile := ""
	fetchedAt := ""
	fl := flag.NewFlagSet("fetch", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: fetch [feed_id...]")
		fl.PrintDefaults()
	}
	fl.BoolVar(&cmd.CreateFeed, "create-feed", false, "Create feed record if not found")
	fl.StringVar(&cmd.Options.FeedURL, "feed-url", "", "Manually fetch a single URL; you must specify exactly one feed_id")
	fl.StringVar(&fetchedAt, "fetched-at", "", "Manually specify fetched_at value, e.g. 2020-02-06T12:34:56Z")
	fl.StringVar(&secretsFile, "secrets", "", "Path to DMFR Secrets file")
	fl.IntVar(&cmd.Workers, "workers", 1, "Worker threads")
	fl.IntVar(&cmd.Limit, "limit", 0, "Maximum number of feeds to fetch")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.BoolVar(&cmd.DryRun, "dry-run", false, "Dry run; print feeds that would be imported and exit")
	fl.BoolVar(&cmd.Options.IgnoreDuplicateContents, "ignore-duplicate-contents", false, "Allow duplicate internal SHA1 contents")
	fl.BoolVar(&cmd.Options.AllowFTPFetch, "allow-ftp-fetch", false, "Allow fetching from FTP urls")
	fl.BoolVar(&cmd.Options.AllowS3Fetch, "allow-s3-fetch", false, "Allow fetching from S3 urls")
	fl.BoolVar(&cmd.Options.AllowLocalFetch, "allow-local-fetch", false, "Allow fetching from filesystem directories/zip files")
	fl.BoolVar(&cmd.Options.SaveValidationReport, "validation-report", false, "Save validation report")
	fl.StringVar(&cmd.Options.ValidationReportStorage, "validation-report-storage", "", "Storage path for saving validation report JSON")
	fl.StringVar(&cmd.Options.Storage, "storage", ".", "Storage destination; can be s3://... az://... or path to a directory")

	fl.Parse(args)
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	cmd.FeedIDs = fl.Args()
	if fetchedAt != "" {
		t, err := time.Parse(time.RFC3339Nano, fetchedAt)
		if err != nil {
			return err
		}
		cmd.Options.FetchedAt = t
	}
	if secretsFile != "" {
		r, err := dmfr.LoadAndParseRegistry(secretsFile)
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
func (cmd *Command) Run() error {
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
		if _, err := dmfr.GetFeedState(adapter, feed.ID); err != nil {
			return err
		}
		// Prepare options for this fetch
		opts := Options{
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
	var wg sync.WaitGroup
	jobs := make(chan fetchJob, len(cmd.FeedIDs))
	results := make(chan StaticFetchResult, len(cmd.FeedIDs))
	for w := 0; w < cmd.Workers; w++ {
		wg.Add(1)
		go fetchWorker(w, cmd.Adapter, cmd.DryRun, jobs, results, &wg)
	}
	for _, opts := range toFetch {
		jobs <- opts
	}
	close(jobs)
	wg.Wait()
	close(results)
	var fatalError error
	fetchFatalErrors := 0
	fetchNew := 0
	fetchFound := 0
	fetchErrs := 0
	for fr := range results {
		cmd.Results = append(cmd.Results, fr)
		if fr.Result.Error != nil {
			fetchFatalErrors++
			fatalError = fr.Result.Error
		} else if fr.Result.FetchError != nil {
			fetchErrs++
		} else if fr.Result.Found {
			fetchFound++
		} else {
			fetchNew++
		}
	}
	log.Infof("Existing: %d New: %d Errors: %d", fetchFound, fetchNew, fetchErrs)
	if fatalError != nil {
		log.Infof("Exiting with error because at least one feed had fatal error: %s", fatalError.Error())
		return fatalError
	}
	return nil
}

type fetchJob struct {
	OnestopID string
	Options
}

func fetchWorker(id int, adapter tldb.Adapter, DryRun bool, jobs <-chan fetchJob, results chan<- StaticFetchResult, wg *sync.WaitGroup) {
	for job := range jobs {
		// Start
		log.Infof("Feed %s: start", job.OnestopID)
		if DryRun {
			log.Infof("Feed %s: dry-run", job.OnestopID)
			continue
		}

		// Fetch
		var result StaticFetchResult
		t := time.Now()
		fetchError := adapter.Tx(func(atx tldb.Adapter) error {
			var fetchError error
			result, fetchError = StaticFetch(atx, job.Options)
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
		results <- result
	}
	wg.Done()
}
