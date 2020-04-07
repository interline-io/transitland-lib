package dmfr

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/log"
)

// FetchCommand fetches feeds defined a DMFR database.
type FetchCommand struct {
	FetchOptions FetchOptions
	Workers      int
	Limit        int
	DBURL        string
	DryRun       bool
	FeedIDs      []string
	secretsFile  string
	fetchedAt    string
	adapter      gtdb.Adapter
}

// Parse sets options from command line flags.
func (cmd *FetchCommand) Parse(args []string) error {
	secretsFile := ""
	fetchedAt := ""
	fl := flag.NewFlagSet("fetch", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: fetch [feed_id...]")
		fl.PrintDefaults()
	}
	fl.StringVar(&cmd.FetchOptions.FeedURL, "feed-url", "", "Manually fetch a single URL; you must specify exactly one feed_id")
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
	q := cmd.adapter.Sqrl().
		Select("*").
		From("current_feeds").
		Where("deleted_at IS NULL").
		Where("spec = ?", "gtfs")
	if len(cmd.FeedIDs) > 0 {
		q = q.Where(sq.Eq{"onestop_id": cmd.FeedIDs})
	}
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
	///////////////
	// Here we go
	log.Info("Fetching %d feeds", len(feeds))
	fetchNew := 0
	fetchFound := 0
	fetchErrs := 0
	var wg sync.WaitGroup
	jobs := make(chan FetchOptions, len(feeds))
	results := make(chan FetchResult, len(feeds))
	for w := 0; w < cmd.Workers; w++ {
		wg.Add(1)
		go fetchWorker(w, cmd.adapter, cmd.DryRun, jobs, results, &wg)
	}
	for _, feed := range feeds {
		opts := FetchOptions{
			Feed:                    feed,
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

func fetchWorker(id int, adapter gtdb.Adapter, DryRun bool, jobs <-chan FetchOptions, results chan<- FetchResult, wg *sync.WaitGroup) {
	for opts := range jobs {
		var fr FetchResult
		osid := ""
		if err := adapter.Get(&osid, "SELECT current_feeds.onestop_id FROM current_feeds WHERE id = ?", opts.Feed.ID); err != nil {
			log.Info("Serious error: could not get details for Feed %d", opts.Feed.ID)
			continue
		}
		log.Debug("Feed %s (id:%d): url: %s begin", osid, fr.FeedVersion.FeedID, fr.FeedVersion.URL)
		if DryRun {
			log.Info("Feed %s (id:%d): dry-run", osid, opts.Feed.ID)
			continue
		}
		err := adapter.Tx(func(atx gtdb.Adapter) error {
			var fe error
			fr, fe = DatabaseFetch(atx, opts)
			return fe
		})
		if err != nil {
			log.Info("Feed %s (id:%d): url: %s critical error: %s", osid, fr.FeedVersion.FeedID, fr.FeedVersion.URL, err.Error())
		} else if fr.FetchError != nil {
			log.Info("Feed %s (id:%d): url: %s fetch error: %s", osid, fr.FeedVersion.FeedID, fr.FeedVersion.URL, fr.FetchError.Error())
		} else if fr.FoundSHA1 {
			log.Info("Feed %s (id:%d): url: %s found zip sha1: %s (id:%d)", osid, fr.FeedVersion.FeedID, fr.FeedVersion.URL, fr.FeedVersion.SHA1, fr.FeedVersion.ID)
		} else if fr.FoundDirSHA1 {
			log.Info("Feed %s (id:%d): url: %s found contents sha1: %s (id:%d)", osid, fr.FeedVersion.FeedID, fr.FeedVersion.URL, fr.FeedVersion.SHA1Dir, fr.FeedVersion.ID)
		} else {
			log.Info("Feed %s (id:%d): url: %s new: %s (id:%d)", osid, fr.FeedVersion.FeedID, fr.FeedVersion.URL, fr.FeedVersion.SHA1, fr.FeedVersion.ID)
		}
		results <- fr
	}
	wg.Done()
}
