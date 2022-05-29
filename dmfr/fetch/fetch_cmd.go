package fetch

import (
	"errors"
	"flag"
	"os"
	"sync"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
)

// Command fetches feeds defined a DMFR database.
type Command struct {
	Options Options
	Workers int
	Limit   int
	DBURL   string
	DryRun  bool
	FeedIDs []string
	adapter tldb.Adapter
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
	fl.StringVar(&cmd.Options.FeedURL, "feed-url", "", "Manually fetch a single URL; you must specify exactly one feed_id")
	fl.BoolVar(&cmd.Options.FeedCreate, "create-feed", false, "Create feed records if not found")
	fl.StringVar(&fetchedAt, "fetched-at", "", "Manually specify fetched_at value, e.g. 2020-02-06T12:34:56Z")
	fl.StringVar(&secretsFile, "secrets", "", "Path to DMFR Secrets file")
	fl.IntVar(&cmd.Workers, "workers", 1, "Worker threads")
	fl.IntVar(&cmd.Limit, "limit", 0, "Maximum number of feeds to fetch")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.StringVar(&cmd.Options.Directory, "gtfsdir", ".", "GTFS Directory")
	fl.BoolVar(&cmd.DryRun, "dry-run", false, "Dry run; print feeds that would be imported and exit")
	fl.BoolVar(&cmd.Options.IgnoreDuplicateContents, "ignore-duplicate-contents", false, "Allow duplicate internal SHA1 contents")
	fl.BoolVar(&cmd.Options.AllowS3Fetch, "allow-s3-fetch", false, "Allow fetching from S3 urls")
	fl.BoolVar(&cmd.Options.AllowFTPFetch, "allow-ftp-fetch", false, "Allow fetching from FTP urls")
	fl.BoolVar(&cmd.Options.AllowLocalFetch, "allow-local-fetch", false, "Allow fetching from filesystem directories/zip files")
	fl.StringVar(&cmd.Options.S3, "s3", "", "Upload GTFS files to S3 bucket/prefix")
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
	if cmd.Options.FeedURL != "" && len(cmd.FeedIDs) != 1 {
		return errors.New("you must specify exactly one feed_id when using -fetch-url")
	}
	return nil
}

// Run executes this command.
func (cmd *Command) Run() error {
	// Get feeds
	if cmd.adapter == nil {
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return err
		}
		cmd.adapter = writer.Adapter
		defer writer.Close()
	}
	if len(cmd.FeedIDs) == 0 {
		q := cmd.adapter.Sqrl().
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
		err = cmd.adapter.Select(&feeds, qstr, qargs...)
		if err != nil {
			return err
		}
		for _, feed := range feeds {
			if feed.URLs.StaticCurrent != "" {
				cmd.FeedIDs = append(cmd.FeedIDs, feed.FeedID)
			}
		}
	}
	///////////////
	// Here we go
	log.Infof("Fetching %d feeds", len(cmd.FeedIDs))
	var wg sync.WaitGroup
	jobs := make(chan Options, len(cmd.FeedIDs))
	results := make(chan Result, len(cmd.FeedIDs))
	for w := 0; w < cmd.Workers; w++ {
		wg.Add(1)
		go fetchWorker(w, cmd.adapter, cmd.DryRun, jobs, results, &wg)
	}
	for _, feedid := range cmd.FeedIDs {
		opts := Options{
			FeedID:                  feedid,
			FeedCreate:              cmd.Options.FeedCreate,
			FeedURL:                 cmd.Options.FeedURL,
			URLType:                 cmd.Options.URLType,
			Directory:               cmd.Options.Directory,
			S3:                      cmd.Options.S3,
			IgnoreDuplicateContents: cmd.Options.IgnoreDuplicateContents,
			AllowS3Fetch:            cmd.Options.AllowS3Fetch,
			AllowFTPFetch:           cmd.Options.AllowFTPFetch,
			AllowLocalFetch:         cmd.Options.AllowLocalFetch,
			FetchedAt:               cmd.Options.FetchedAt,
			Secrets:                 cmd.Options.Secrets,
		}
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

func fetchWorker(id int, adapter tldb.Adapter, DryRun bool, jobs <-chan Options, results chan<- Result, wg *sync.WaitGroup) {
	for opts := range jobs {
		// Get FeedID for pretty printing.
		osid := opts.FeedID
		log.Infof("Feed %s: start", osid)
		if DryRun {
			log.Infof("Feed %s: dry-run", osid)
			continue
		}
		var fr Result
		var fv tl.FeedVersion
		t := time.Now()
		err := adapter.Tx(func(atx tldb.Adapter) error {
			var fe error
			fv, fr, fe = FeedStateFetch(atx, opts)
			return fe
		})
		t2 := float64(time.Now().UnixNano()-t.UnixNano()) / 1e9 // 1000000000.0
		fid := fv.FeedID
		furl := fv.URL
		if err != nil {
			fr.Error = err
			log.Error().Err(err).Msgf("Feed %s (id:%d): url: %s critical error: %s (t:%0.2fs)", osid, fid, furl, err.Error(), t2)
		} else if fr.FetchError != nil {
			log.Error().Err(fr.FetchError).Msgf("Feed %s (id:%d): url: %s fetch error: %s (t:%0.2fs)", osid, fid, furl, fr.FetchError.Error(), t2)
		} else if fr.Found {
			log.Infof("Feed %s (id:%d): url: %s found sha1: %s (id:%d) (t:%0.2fs)", osid, fid, furl, fv.SHA1, fv.ID, t2)
		} else {
			log.Infof("Feed %s (id:%d): url: %s new: %s (id:%d) (t:%0.2fs)", osid, fid, furl, fv.SHA1, fv.ID, t2)
		}
		results <- fr
	}
	wg.Done()
}
