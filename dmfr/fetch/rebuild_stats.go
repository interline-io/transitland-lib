package fetch

import (
	"flag"
	"os"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/cli"
	"github.com/interline-io/transitland-lib/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/request"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
)

type RebuildStatsOptions struct {
	FeedVersionID int
	Directory     string
	S3            string
}

type RebuildStatsResult struct {
	Error error
}

// RebuildStatsCommand rebuilds feed version statistics
type RebuildStatsCommand struct {
	Options RebuildStatsOptions
	Workers int
	DBURL   string
	FeedIDs []string
	FVIDs   cli.ArrayFlags
	FVSHA1  cli.ArrayFlags
	Adapter tldb.Adapter // allow for mocks
}

// Parse command line flags
func (cmd *RebuildStatsCommand) Parse(args []string) error {
	fvidfile := ""
	fvsha1file := ""
	fl := flag.NewFlagSet("rebuild-stats", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: rebuild-stats [feedids...]")
		fl.PrintDefaults()
	}
	fl.Var(&cmd.FVIDs, "fvid", "Rebuild stats for specific feed version ID")
	fl.StringVar(&fvidfile, "fvid-file", "", "Specify feed version IDs in file, one per line; equivalent to multiple --fvid")
	fl.StringVar(&fvsha1file, "fv-sha1-file", "", "Specify feed version IDs by SHA1 in file, one per line")
	fl.Var(&cmd.FVSHA1, "fv-sha1", "Feed version SHA1")
	fl.IntVar(&cmd.Workers, "workers", 1, "Worker threads")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.StringVar(&cmd.Options.Directory, "gtfsdir", ".", "GTFS Directory")
	fl.StringVar(&cmd.Options.S3, "s3", "", "Get GTFS files from S3 bucket/prefix")
	fl.Parse(args)
	cmd.FeedIDs = fl.Args()
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	if fvidfile != "" {
		lines, err := cli.ReadFileLines(fvidfile)
		if err != nil {
			return err
		}
		for _, line := range lines {
			if line != "" {
				cmd.FVIDs = append(cmd.FVIDs, line)
			}
		}
	}
	if fvsha1file != "" {
		lines, err := cli.ReadFileLines(fvsha1file)
		if err != nil {
			return err
		}
		for _, line := range lines {
			if line != "" {
				cmd.FVSHA1 = append(cmd.FVSHA1, line)
			}
		}
	}
	return nil
}

// Run this command
func (cmd *RebuildStatsCommand) Run() error {
	if cmd.Adapter == nil {
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return err
		}
		cmd.Adapter = writer.Adapter
		defer writer.Close()
	}
	// Query to get FVs to import
	q := cmd.Adapter.Sqrl().
		Select("feed_versions.id").
		From("feed_versions").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id").
		OrderBy("feed_versions.id desc")
	if len(cmd.FeedIDs) > 0 {
		// Limit to specified feeds
		q = q.Where(sq.Eq{"onestop_id": cmd.FeedIDs})
	}
	if len(cmd.FVIDs) > 0 {
		// Explicitly specify fvids
		q = q.Where(sq.Eq{"feed_versions.id": cmd.FVIDs})
	}
	if len(cmd.FVSHA1) > 0 {
		// Explicitly specify fv sha1
		q = q.Where(sq.Eq{"feed_versions.sha1": cmd.FVSHA1})
	}
	qstr, qargs, err := q.ToSql()
	if err != nil {
		return err
	}
	qrs := []int{}
	err = cmd.Adapter.Select(&qrs, qstr, qargs...)
	if err != nil {
		return err
	}
	///////////////
	// Here we go
	log.Infof("Rebuilding stats for %d feed versions", len(qrs))
	jobs := make(chan RebuildStatsOptions, len(qrs))
	results := make(chan RebuildStatsResult, len(qrs))
	for _, fvid := range qrs {
		jobs <- RebuildStatsOptions{
			FeedVersionID: fvid,
			Directory:     cmd.Options.Directory,
			S3:            cmd.Options.S3,
		}
	}
	close(jobs)
	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < cmd.Workers; w++ {
		wg.Add(1)
		go rebuildStatsWorker(w, cmd.Adapter, false, jobs, results, &wg)
	}
	wg.Wait()
	return nil
}

func rebuildStatsWorker(id int, adapter tldb.Adapter, dryrun bool, jobs <-chan RebuildStatsOptions, results chan<- RebuildStatsResult, wg *sync.WaitGroup) {
	type qr struct {
		FeedVersionID   int
		FeedID          int
		FeedOnestopID   string
		FeedVersionSHA1 string
	}
	for opts := range jobs {
		q := qr{}
		if err := adapter.Get(&q, "SELECT feed_versions.id as feed_version_id, feed_versions.feed_id as feed_id, feed_versions.sha1 as feed_version_sha1, current_feeds.onestop_id as feed_onestop_id FROM feed_versions INNER JOIN current_feeds ON current_feeds.id = feed_versions.feed_id WHERE feed_versions.id = ?", opts.FeedVersionID); err != nil {
			log.Errorf("Could not get details for FeedVersion %d", opts.FeedVersionID)
			continue
		}
		if dryrun {
			log.Infof("Feed %s (id:%d): FeedVersion %s (id:%d): dry-run", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID)
			continue
		}
		log.Infof("Feed %s (id:%d): FeedVersion %s (id:%d): begin", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID)
		t := time.Now()
		result, err := rebuildStatsMain(adapter, opts)
		t2 := float64(time.Now().UnixNano()-t.UnixNano()) / 1e9 // 1000000000.0
		if err != nil {
			log.Errorf("Feed %s (id:%d): FeedVersion %s (id:%d): critical failure, rolled back: %s (t:%0.2fs)", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, err.Error(), t2)
		} else {
			log.Infof("Feed %s (id:%d): FeedVersion %s (id:%d): success (t:%0.2fs)", q.FeedOnestopID, q.FeedID, q.FeedVersionSHA1, q.FeedVersionID, t2)
		}
		results <- result
	}
	wg.Done()
}

func rebuildStatsMain(adapter tldb.Adapter, opts RebuildStatsOptions) (RebuildStatsResult, error) {
	// Get FV
	fv := tl.FeedVersion{ID: opts.FeedVersionID}
	if err := adapter.Find(&fv); err != nil {
		return RebuildStatsResult{}, err
	}
	// Get Reader
	var reqOpts []request.RequestOption
	reqOpts = append(reqOpts, request.WithAllowLocal)
	if opts.S3 != "" {
		reqOpts = append(reqOpts, request.WithAllowS3)
	}
	adapterUrl := dmfr.GetReaderURL(opts.S3, opts.Directory, fv.File, fv.SHA1)
	reader, err := tlcsv.NewReaderFromAdapter(tlcsv.NewURLAdapter(adapterUrl, reqOpts...))
	if err != nil {
		return RebuildStatsResult{}, err
	}
	if err := reader.Open(); err != nil {
		return RebuildStatsResult{}, err
	}
	// Import
	errImport := adapter.Tx(func(atx tldb.Adapter) error {
		if err := createFeedStats(atx, reader, fv.ID); err != nil {
			return err
		}
		return nil
	})
	return RebuildStatsResult{}, errImport
}

func createFeedStats(atx tldb.Adapter, reader *tlcsv.Reader, fvid int) error {
	// Get FeedVersionFileInfos
	fvfis, err := dmfr.NewFeedVersionFileInfosFromReader(reader)
	if err != nil {
		return err
	}
	// Get service statistics
	fvsls, err := dmfr.NewFeedVersionServiceInfosFromReader(reader)
	if err != nil {
		return err
	}
	// Get fvsw
	fvsw, err := dmfr.NewFeedVersionServiceWindowFromReader(reader)
	if err != nil {
		return err
	}
	fvsw.FeedVersionID = fvid
	// Delete any existing records
	tables := []string{"feed_version_file_infos", "feed_version_service_levels", "feed_version_service_windows"}
	for _, table := range tables {
		q, args, err := atx.Sqrl().Delete(table).Where(sq.Eq{"feed_version_id": fvid}).ToSql()
		if err != nil {
			return err
		}
		if _, err := atx.DBX().Exec(q, args...); err != nil {
			return err
		}
	}
	// Insert FVFIs
	for _, fvfi := range fvfis {
		fvfi.UpdateTimestamps()
		fvfi.FeedVersionID = fvid
		if _, err := atx.Insert(&fvfi); err != nil {
			return err
		}
	}
	// Insert FVSW
	if _, err := atx.Insert(&fvsw); err != nil {
		return err
	}
	// Batch insert FVSLs
	bt := make([]any, len(fvsls))
	for i := range fvsls {
		fvsls[i].FeedVersionID = fvid
		bt[i] = &fvsls[i]
	}
	if err := atx.CopyInsert(bt); err != nil {
		return err
	}
	return nil
}
