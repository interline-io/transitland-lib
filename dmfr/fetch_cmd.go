package dmfr

import (
	"errors"
	"flag"
	"os"
	"sync"
	"time"

	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tldb"
)

// FetchCommand fetches feeds defined a DMFR database.
type FetchCommand struct {
	FetchOptions FetchOptions
	Workers      int
	Limit        int
	DBURL        string
	DryRun       bool
	FeedIDs      []string
	adapter      tldb.Adapter
}

// Parse sets options from command line flags.
func (cmd *FetchCommand) Parse(args []string) error {
	secretsFile := ""
	fetchedAt := ""
	fl := flag.NewFlagSet("fetch", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: fetch [feed_id...]")
		fl.PrintDefaults()
	}
	fl.StringVar(&cmd.FetchOptions.FeedURL, "feed-url", "", "Manually fetch a single URL; you must specify exactly one feed_id")
	fl.BoolVar(&cmd.FetchOptions.FeedCreate, "create-feed", false, "Create feed records if not found")
	fl.StringVar(&fetchedAt, "fetched-at", "", "Manually specify fetched_at value, e.g. 2020-02-06T12:34:56Z")
	fl.StringVar(&secretsFile, "secrets", "", "Path to DMFR Secrets file")
	fl.IntVar(&cmd.Workers, "workers", 1, "Worker threads")
	fl.IntVar(&cmd.Limit, "limit", 0, "Maximum number of feeds to fetch")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $DMFR_DATABASE_URL)")
	fl.StringVar(&cmd.FetchOptions.Directory, "gtfsdir", ".", "GTFS Directory")
	fl.BoolVar(&cmd.DryRun, "dry-run", false, "Dry run; print feeds that would be imported and exit")
	fl.BoolVar(&cmd.FetchOptions.IgnoreDuplicateContents, "ignore-duplicate-contents", false, "Allow duplicate internal SHA1 contents")
	fl.StringVar(&cmd.FetchOptions.S3, "s3", "", "Upload GTFS files to S3 bucket/prefix")
	fl.Parse(args)
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("DMFR_DATABASE_URL")
	}
	cmd.FeedIDs = fl.Args()
	if fetchedAt != "" {
		t, err := time.Parse(time.RFC3339Nano, fetchedAt)
		if err != nil {
			return err
		}
		cmd.FetchOptions.FetchedAt = t
	}
	if secretsFile != "" {
		if err := cmd.FetchOptions.Secrets.Load(secretsFile); err != nil {
			return err
		}
	}
	if cmd.FetchOptions.FeedURL != "" && len(cmd.FeedIDs) != 1 {
		return errors.New("you must specify exactly one feed_id when using -fetch-url")
	}
	return nil
}

// Run executes this command.
func (cmd *FetchCommand) Run() error {
	// Get feeds
	if cmd.adapter == nil {
		writer := mustGetWriter(cmd.DBURL, true)
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
		feeds := []Feed{}
		err = cmd.adapter.Select(&feeds, qstr, qargs...)
		if err != nil {
			return err
		}
		for _, feed := range feeds {
			cmd.FeedIDs = append(cmd.FeedIDs, feed.FeedID)
		}
	}
	///////////////
	// Here we go
	log.Info("Fetching %d feeds", len(cmd.FeedIDs))
	fetchNew := 0
	fetchFound := 0
	fetchErrs := 0
	var wg sync.WaitGroup
	jobs := make(chan FetchOptions, len(cmd.FeedIDs))
	results := make(chan FetchResult, len(cmd.FeedIDs))
	for w := 0; w < cmd.Workers; w++ {
		wg.Add(1)
		go fetchWorker(w, cmd.adapter, cmd.DryRun, jobs, results, &wg)
	}
	for _, feedid := range cmd.FeedIDs {
		opts := FetchOptions{
			FeedID:                  feedid,
			FeedCreate:              cmd.FetchOptions.FeedCreate,
			FeedURL:                 cmd.FetchOptions.FeedURL,
			Directory:               cmd.FetchOptions.Directory,
			S3:                      cmd.FetchOptions.S3,
			IgnoreDuplicateContents: cmd.FetchOptions.IgnoreDuplicateContents,
			FetchedAt:               cmd.FetchOptions.FetchedAt,
			Secrets:                 cmd.FetchOptions.Secrets,
		}
		jobs <- opts
	}
	close(jobs)
	wg.Wait()
	close(results)
	for fr := range results {
		if fr.FetchError != nil {
			fetchErrs++
		} else if fr.FoundSHA1 {
			fetchFound++
		} else if fr.FoundDirSHA1 {
			fetchFound++
		} else {
			fetchNew++
		}
	}
	log.Info("Existing: %d New: %d Errors: %d", fetchFound, fetchNew, fetchErrs)
	return nil
}

func fetchWorker(id int, adapter tldb.Adapter, DryRun bool, jobs <-chan FetchOptions, results chan<- FetchResult, wg *sync.WaitGroup) {
	for opts := range jobs {
		// Get FeedID for pretty printing.
		osid := opts.FeedID
		log.Info("Feed %s: start", osid)
		if DryRun {
			log.Info("Feed %s: dry-run", osid)
			continue
		}
		var fr FetchResult
		err := adapter.Tx(func(atx tldb.Adapter) error {
			var fe error
			fr, fe = DatabaseFetch(atx, opts)
			return fe
		})
		fid := fr.FeedVersion.FeedID
		furl := fr.FeedVersion.URL
		if err != nil {
			log.Error("Feed %s (id:%d): url: %s critical error: %s", osid, fid, furl, err.Error())
		} else if fr.FetchError != nil {
			log.Error("Feed %s (id:%d): url: %s fetch error: %s", osid, fid, furl, fr.FetchError.Error())
		} else if fr.FoundSHA1 {
			log.Info("Feed %s (id:%d): url: %s found zip sha1: %s (id:%d)", osid, fid, furl, fr.FeedVersion.SHA1, fr.FeedVersion.ID)
		} else if fr.FoundDirSHA1 {
			log.Info("Feed %s (id:%d): url: %s found contents sha1: %s (id:%d)", osid, fid, furl, fr.FeedVersion.SHA1Dir, fr.FeedVersion.ID)
		} else {
			log.Info("Feed %s (id:%d): url: %s new: %s (id:%d)", osid, fid, furl, fr.FeedVersion.SHA1, fr.FeedVersion.ID)
		}
		results <- fr
	}
	wg.Done()
}
