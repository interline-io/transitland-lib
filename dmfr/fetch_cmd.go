package dmfr

import (
	"flag"
	"fmt"
	"os"
	"sync"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/log"
)

// FetchCommand fetches all URLs and creates new FeedVersions.
type FetchCommand struct {
	FetchOptions
	SecretsFile string
	DmfrFile    string
	Workers     int
	Limit       int
	DryRun      bool
	DBURL       string
	feedids     []string
	adapter     gtdb.Adapter
}

// Run .
func (cmd *FetchCommand) Run(args []string) error {
	if err := cmd.Parse(args); err != nil {
		return err
	}
	return cmd.Fetch()
}

// Parse .
func (cmd *FetchCommand) Parse(args []string) error {
	fl := flag.NewFlagSet("fetch", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: fetch [feedids...]")
		fl.PrintDefaults()
	}
	fl.IntVar(&cmd.Workers, "workers", 1, "Worker threads")
	fl.IntVar(&cmd.Limit, "limit", 0, "Maximum number of feeds to fetch")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $DMFR_DATABASE_URL)")
	fl.StringVar(&cmd.DmfrFile, "dmfr", "", "DMFR File")
	fl.StringVar(&cmd.Directory, "gtfsdir", "", "GTFS Directory")
	fl.BoolVar(&cmd.DryRun, "dryrun", false, "Dry run; print feeds that would be imported and exit")
	fl.BoolVar(&cmd.IgnoreDuplicateContents, "allow-duplicate-contents", false, "Allow duplicate internal SHA1 contents")
	fl.StringVar(&cmd.SecretsFile, "secrets", "", "Authorizaton secrets file")
	fl.StringVar(&cmd.S3, "s3", "", "Upload GTFS files to S3 bucket/prefix")
	fl.Parse(args)
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("DMFR_DATABASE_URL")
	}
	cmd.feedids = fl.Args()
	return nil
}

// Fetch .
func (cmd *FetchCommand) Fetch() error {
	// Load secrets
	if cmd.SecretsFile != "" {
		cmd.secrets.Load(cmd.SecretsFile)
	}
	// Get feeds
	feeds := []Feed{}
	if cmd.DmfrFile != "" {
		reg, err := LoadAndParseRegistry(cmd.DmfrFile)
		if err != nil {
			panic(err)
		}
		feeds = reg.Feeds
	} else if cmd.DBURL != "" {
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
		if len(cmd.feedids) > 0 {
			q = q.Where(sq.Eq{"onestop_id": cmd.feedids})
		}
		if cmd.Limit > 0 {
			q = q.Limit(uint64(cmd.Limit))
		}
		qstr, qargs, err := q.ToSql()
		if err != nil {
			return err
		}
		err = cmd.adapter.Select(&feeds, qstr, qargs...)
		if err != nil {
			return err
		}
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
			FeedID:                  feed.ID,
			Directory:               cmd.Directory,
			S3:                      cmd.S3,
			IgnoreDuplicateContents: cmd.IgnoreDuplicateContents,
			secrets:                 cmd.secrets,
			feed:                    feed,
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

func fetchWorker(id int, adapter gtdb.Adapter, dryrun bool, jobs <-chan FetchOptions, results chan<- FetchResult, wg *sync.WaitGroup) {
	for opts := range jobs {
		var fr FetchResult
		osid := ""
		if adapter == nil {
			// pass
		} else if err := adapter.Get(&osid, "SELECT current_feeds.onestop_id FROM current_feeds WHERE id = ?", opts.FeedID); err != nil {
			log.Info("Serious error: could not get details for Feed %d", opts.FeedID)
			continue
		}
		log.Debug("Feed %s (id:%d): url: %s begin", osid, fr.FeedVersion.FeedID, fr.FeedVersion.URL)
		if dryrun {
			log.Info("Feed %s (id:%d): dry-run", osid, opts.FeedID)
			continue
		}
		var err error
		if adapter == nil {
			fr, err = Fetch(opts)
		} else {
			err = adapter.Tx(func(atx gtdb.Adapter) error {
				var fe error
				fr, fe = DatabaseFetch(atx, opts)
				return fe
			})
		}
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
