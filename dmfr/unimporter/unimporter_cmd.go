package unimporter

import (
	"flag"
	"sync"
	"time"

	"github.com/interline-io/transitland-lib/internal/cli"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tldb"
)

// Command imports FeedVersions into a database.
type Command struct {
	Options Options
	DryRun  bool
	FVIDs   cli.ArrayFlags
	FVSHA1  cli.ArrayFlags
	DBURL   string
	Workers int
	Adapter tldb.Adapter // allow for mocks
}

// Parse command line flags
func (cmd *Command) Parse(args []string) error {
	extflags := cli.ArrayFlags{}
	fl := flag.NewFlagSet("import", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: import [feedids...]")
		fl.PrintDefaults()
	}
	fl.Var(&extflags, "ext", "Include GTFS Extension")
	fl.Var(&cmd.FVIDs, "fvid", "Import specific feed version ID")
	fl.Var(&cmd.FVSHA1, "fv-sha1", "Feed version SHA1")
	fl.BoolVar(&cmd.DryRun, "dryrun", false, "Dry run; print feeds that would be imported and exit")
	fl.Parse(args)
	return nil
}

// Run this command
func (cmd *Command) Run() error {
	if cmd.Adapter == nil {
		writer := tldb.MustGetWriter(cmd.DBURL, true)
		cmd.Adapter = writer.Adapter
		defer writer.Close()
	}
	qrs := []int{}
	log.Info("Importing %d feed versions", len(cmd.FVIDs))
	jobs := make(chan Options, len(qrs))
	results := make(chan Result, len(qrs))
	for _, fvid := range qrs {
		jobs <- Options{
			FeedVersionID: fvid,
		}
	}
	close(jobs)
	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < cmd.Workers; w++ {
		wg.Add(1)
		go dmfrUnimportWorker(w, cmd.Adapter, cmd.DryRun, jobs, results, &wg)
	}
	wg.Wait()
	return nil
}

func dmfrUnimportWorker(id int, adapter tldb.Adapter, dryrun bool, jobs <-chan Options, results chan<- Result, wg *sync.WaitGroup) {
	type qr struct {
		FeedVersionID   int
		FeedID          int
		FeedOnestopID   string
		FeedVersionSHA1 string
	}
	for opts := range jobs {
		q := qr{}
		query := `
		SELECT 
			feed_versions.id as feed_version_id, 
			feed_versions.feed_id as feed_id, 
			feed_versions.sha1 as feed_version_sha1, 
			current_feeds.onestop_id as feed_onestop_id 
		FROM feed_versions 
		INNER JOIN current_feeds ON current_feeds.id = feed_versions.feed_id 
		WHERE feed_versions.id = ?
		`
		if err := adapter.Get(&q, query, opts.FeedVersionID); err != nil {
			log.Error("Could not get details for FeedVersion %d", opts.FeedVersionID)
			continue
		}
		if dryrun {
			log.Info("Feed %s (id:%d): FeedVersion %s (id:%d): dry-run", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID)
			continue
		}
		log.Info("Feed %s (id:%d): FeedVersion %s (id:%d): begin", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID)
		t := time.Now()
		result, err := UnimportFeedVersion(adapter, opts)
		t2 := float64(time.Now().UnixNano()-t.UnixNano()) / 1e9 // 1000000000.0
		if err != nil {
			log.Error("Feed %s (id:%d): FeedVersion %s (id:%d): critical failure, rolled back: %s (t:%0.2fs)", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, result.ExceptionLog, t2)
		} else if result.Success {
			log.Info("Feed %s (id:%d): FeedVersion %s (id:%d): success (t:%0.2fs)", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, t2)
		} else {
			log.Info("Feed %s (id:%d): FeedVersion %s (id:%d): error: %s, (t:%0.2fs)", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, result.ExceptionLog, t2)
		}
		results <- result
	}
	wg.Done()
}
