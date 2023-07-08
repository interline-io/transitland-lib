package fetch

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
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
	return nil
}

type fetchJob struct {
	OnestopID string
	Options
}

// Run executes this command.
func (cmd *Command) Run() error {
	if cmd.Adapter == nil {
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return err
		}
		cmd.Adapter = writer.Adapter
		defer writer.Close()
	}

	// Get default feeds
	if len(cmd.FeedIDs) == 0 {
		q := cmd.Adapter.Sqrl().
			Select("onestop_id").
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
		if err = cmd.Adapter.Select(&cmd.FeedIDs, qstr, qargs...); err != nil {
			return err
		}
	}

	// Check feeds
	// TODO: consider getting feed_fetches at the same time, but this would require a lateral join,
	// 		 wwhich sqlite does not support
	type feedCheck struct {
		tl.Feed
		FetchWait tt.Int
	}
	q := cmd.Adapter.Sqrl().
		Select("current_feeds.*", "feed_states.fetch_wait").
		From("current_feeds").
		LeftJoin("feed_states on feed_states.feed_id = current_feeds.id").
		Where(sq.Eq{"current_feeds.onestop_id": cmd.FeedIDs})
	var feedCheckResult []feedCheck
	if qstr, qargs, err := q.ToSql(); err != nil {
		return err
	} else if err := cmd.Adapter.Select(&feedCheckResult, qstr, qargs...); err != nil {
		return err
	}
	feedChecks := map[string]feedCheck{}
	for _, v := range feedCheckResult {
		feedChecks[v.FeedID] = v
	}

	var toFetch []fetchJob
	adapter := cmd.Adapter
	for _, osid := range cmd.FeedIDs {
		// Get feed, create if not present and CreateFeed is specified
		fmt.Println("Checking:", osid)
		feedCheck, ok := feedChecks[osid]
		if !ok && cmd.CreateFeed {
			feedCheck.FeedID = osid
			feedCheck.Spec = "gtfs"
			var err error
			if feedCheck.ID, err = adapter.Insert(&feedCheck.Feed); err != nil {
				return err
			}
		} else if !ok {
			return fmt.Errorf("feed not found: %s", osid)
		}

		// Prepare options for this fetch
		opts := Options{
			FeedID:                  feedCheck.ID,
			FeedURL:                 cmd.Options.FeedURL,
			FetchedAt:               cmd.Options.FetchedAt,
			URLType:                 cmd.Options.URLType,
			Storage:                 cmd.Options.Storage,
			IgnoreDuplicateContents: cmd.Options.IgnoreDuplicateContents,
			AllowFTPFetch:           cmd.Options.AllowFTPFetch,
			AllowS3Fetch:            cmd.Options.AllowS3Fetch,
			AllowLocalFetch:         cmd.Options.AllowLocalFetch,
			Secrets:                 cmd.Options.Secrets,
		}

		// Check if enough time has passed
		fmt.Println("\tchecking time for feed:", osid)
		if ok, err := CheckFetchWait(adapter, opts.FeedID, float64(feedCheck.FetchWait.Val)); err != nil {
			return err
		} else if !ok {
			fmt.Println("\tskipping feed, failed time check:", osid)
			continue
		}

		// Set defaults
		opts.URLType = "manual"
		if opts.FeedURL == "" {
			opts.URLType = "static_current"
			opts.FeedURL = feedCheck.URLs.StaticCurrent
		}
		if opts.FeedURL == "" {
			fmt.Println("\tskipping feed, no url:", osid)
			continue
		}

		toFetch = append(toFetch, fetchJob{OnestopID: osid, Options: opts})
	}

	///////////////
	// Here we go
	log.Infof("Fetching %d feeds", len(cmd.FeedIDs))
	var wg sync.WaitGroup
	jobs := make(chan fetchJob, len(cmd.FeedIDs))
	results := make(chan Result, len(cmd.FeedIDs))
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
		if fr.Error != nil {
			fetchFatalErrors++
			fatalError = fr.Error
		} else if fr.FetchError != nil {
			fetchErrs++
		} else if fr.Found {
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

func fetchWorker(id int, adapter tldb.Adapter, DryRun bool, jobs <-chan fetchJob, results chan<- Result, wg *sync.WaitGroup) {
	for job := range jobs {
		// Start
		osid := job.OnestopID
		log.Infof("Feed %s: start", osid)
		if DryRun {
			log.Infof("Feed %s: dry-run", osid)
			continue
		}

		// Fetch
		var fr Result
		var fv tl.FeedVersion
		t := time.Now()
		err := adapter.Tx(func(atx tldb.Adapter) error {
			var fe error
			fv, fr, fe = StaticFetch(atx, job.Options)
			return fe
		})
		t2 := float64(time.Now().UnixNano()-t.UnixNano()) / 1e9 // 1000000000.0

		// Check result
		if err != nil {
			fr.Error = err
			log.Error().Err(err).Msgf("Feed %s (id:%d): url: %s critical error: %s (t:%0.2fs)", osid, job.Options.FeedID, fv.URL, err.Error(), t2)
		} else if fr.FetchError != nil {
			log.Error().Err(fr.FetchError).Msgf("Feed %s (id:%d): url: %s fetch error: %s (t:%0.2fs)", osid, job.Options.FeedID, fv.URL, fr.FetchError.Error(), t2)
		} else if fr.Found {
			log.Infof("Feed %s (id:%d): url: %s found sha1: %s (id:%d) (t:%0.2fs)", osid, job.Options.FeedID, fv.URL, fv.SHA1, fv.ID, t2)
		} else {
			log.Infof("Feed %s (id:%d): url: %s new: %s (id:%d) (t:%0.2fs)", osid, job.Options.FeedID, fv.URL, fv.SHA1, fv.ID, t2)
		}
		results <- fr
	}
	wg.Done()
}
