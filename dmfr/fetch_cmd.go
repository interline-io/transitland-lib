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
	fetchURL  string
	fetchedAt string
	workers   int
	limit     int
	dburl     string
	gtfsdir   string
	allowdups bool
	s3        string
	dryrun    bool
	feedids   []string
	adapter   gtdb.Adapter
}

// Run executes this command.
func (cmd *FetchCommand) Run(args []string) error {
	fl := flag.NewFlagSet("fetch", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: fetch [feed_id...]")
		fl.PrintDefaults()
	}
	fl.StringVar(&cmd.fetchURL, "fetch-url", "", "Manually fetch a single URL; you must specify exactly one feed_id")
	fl.StringVar(&cmd.fetchedAt, "fetched-at", "", "Manually specify fetched_at value, e.g. 2020-02-06T12:34:56Z")
	fl.IntVar(&cmd.workers, "workers", 1, "Worker threads")
	fl.IntVar(&cmd.limit, "limit", 0, "Maximum number of feeds to fetch")
	fl.StringVar(&cmd.dburl, "dburl", "", "Database URL (default: $DMFR_DATABASE_URL)")
	fl.StringVar(&cmd.gtfsdir, "gtfsdir", ".", "GTFS Directory")
	fl.BoolVar(&cmd.dryrun, "dryrun", false, "Dry run; print feeds that would be imported and exit")
	fl.BoolVar(&cmd.allowdups, "allow-duplicate-contents", false, "Allow duplicate internal SHA1 contents")
	fl.StringVar(&cmd.s3, "s3", "", "Upload GTFS files to S3 bucket/prefix")
	fl.Parse(args)
	if cmd.dburl == "" {
		cmd.dburl = os.Getenv("DMFR_DATABASE_URL")
	}
	feedids := fl.Args()
	if cmd.adapter == nil {
		writer := mustGetWriter(cmd.dburl, true)
		cmd.adapter = writer.Adapter
		defer writer.Close()
	}
	FetchedAt := time.Time{}
	if cmd.fetchedAt != "" {
		t, err := time.Parse(time.RFC3339Nano, cmd.fetchedAt)
		if err != nil {
			return err
		}
		FetchedAt = t
	}
	if cmd.fetchURL != "" && len(feedids) != 1 {
		return errors.New("you must specify exactly one feed_id when using -fetch-url")
	}

	// Get feeds
	q := cmd.adapter.Sqrl().
		Select("*").
		From("current_feeds").
		Where("deleted_at IS NULL").
		Where("spec = ?", "gtfs")
	if len(feedids) > 0 {
		q = q.Where(sq.Eq{"onestop_id": feedids})
	}
	if cmd.limit > 0 {
		q = q.Limit(uint64(cmd.limit))
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
	for w := 0; w < cmd.workers; w++ {
		wg.Add(1)
		go dmfrFetchWorker(w, cmd.adapter, cmd.dryrun, jobs, results, &wg)
	}
	for _, feed := range feeds {
		opts := FetchOptions{
			FeedID:                  feed.ID,
			FeedURL:                 cmd.fetchURL,
			Directory:               cmd.gtfsdir,
			S3:                      cmd.s3,
			IgnoreDuplicateContents: cmd.allowdups,
			FetchedAt:               FetchedAt,
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

func dmfrFetchWorker(id int, adapter gtdb.Adapter, dryrun bool, jobs <-chan FetchOptions, results chan<- FetchResult, wg *sync.WaitGroup) {
	for opts := range jobs {
		var fr FetchResult
		osid := ""
		if err := adapter.Get(&osid, "SELECT current_feeds.onestop_id FROM current_feeds WHERE id = ?", opts.FeedID); err != nil {
			log.Info("Serious error: could not get details for Feed %d", opts.FeedID)
			continue
		}
		log.Debug("Feed %s (id:%d): url: %s begin", osid, fr.FeedVersion.FeedID, fr.FeedVersion.URL)
		if dryrun {
			log.Info("Feed %s (id:%d): dry-run", osid, opts.FeedID)
			continue
		}
		err := adapter.Tx(func(atx gtdb.Adapter) error {
			var fe error
			fr, fe = MainFetchFeed(atx, opts)
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
