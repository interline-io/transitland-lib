package cmds

import (
	"os"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/spf13/pflag"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr/stats"
	"github.com/interline-io/transitland-lib/dmfr/store"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/validator"
)

type RebuildStatsOptions struct {
	FeedVersionID           int
	Storage                 string
	ValidationReportStorage string
	SaveValidationReport    bool
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
	FVIDs   []string
	FVSHA1  []string
	Adapter tldb.Adapter // allow for mocks
	// internal
	fvidfile   string
	fvsha1file string
}

func (cmd *RebuildStatsCommand) HelpDesc() (string, string) {
	return "Rebuild statistics for feeds or specific feed versions", ""
}

func (cmd *RebuildStatsCommand) HelpArgs() string {
	return "[flags] [feeds...]"
}

func (cmd *RebuildStatsCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringSliceVar(&cmd.FVIDs, "fvid", nil, "Rebuild stats for specific feed version ID")
	fl.StringVar(&cmd.fvidfile, "fvid-file", "", "Specify feed version IDs in file, one per line; equivalent to multiple --fvid")
	fl.StringVar(&cmd.fvsha1file, "fv-sha1-file", "", "Specify feed version IDs by SHA1 in file, one per line")
	fl.StringSliceVar(&cmd.FVSHA1, "fv-sha1", nil, "Feed version SHA1")
	fl.IntVar(&cmd.Workers, "workers", 1, "Worker threads")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.StringVar(&cmd.Options.Storage, "storage", "", "Storage destination; can be s3://... az://... or path to a directory")
	fl.BoolVar(&cmd.Options.SaveValidationReport, "validation-report", false, "Save validation report")
	fl.StringVar(&cmd.Options.ValidationReportStorage, "validation-report-storage", "", "Storage path for saving validation report JSON")
}

// Parse command line flags
func (cmd *RebuildStatsCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	cmd.FeedIDs = fl.Args()
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	if cmd.fvidfile != "" {
		lines, err := tlcli.ReadFileLines(cmd.fvidfile)
		if err != nil {
			return err
		}
		for _, line := range lines {
			if line != "" {
				cmd.FVIDs = append(cmd.FVIDs, line)
			}
		}
	}
	if cmd.fvsha1file != "" {
		lines, err := tlcli.ReadFileLines(cmd.fvsha1file)
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
			FeedVersionID:           fvid,
			Storage:                 cmd.Options.Storage,
			ValidationReportStorage: cmd.Options.ValidationReportStorage,
			SaveValidationReport:    cmd.Options.SaveValidationReport,
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
	_ = id
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
	fv := tl.FeedVersion{}
	fv.ID = opts.FeedVersionID
	if err := adapter.Find(&fv); err != nil {
		return RebuildStatsResult{}, err
	}
	// Get Reader
	tladapter, err := store.NewStoreAdapter(opts.Storage, fv.File, fv.Fragment.Val)
	if err != nil {
		return RebuildStatsResult{}, err
	}
	reader, err := tlcsv.NewReaderFromAdapter(tladapter)
	if err != nil {
		return RebuildStatsResult{}, err
	}
	if err := reader.Open(); err != nil {
		return RebuildStatsResult{}, err
	}
	// Save
	errImport := adapter.Tx(func(atx tldb.Adapter) error {
		if err := stats.CreateFeedStats(atx, reader, fv.ID); err != nil {
			return err
		}
		if opts.SaveValidationReport {
			if _, err := createFeedValidationReport(atx, reader, fv.ID, fv.FetchedAt, opts.ValidationReportStorage); err != nil {
				return err
			}
		}
		return nil
	})
	return RebuildStatsResult{}, errImport
}

func createFeedValidationReport(atx tldb.Adapter, reader *tlcsv.Reader, fvid int, fetchedAt time.Time, storage string) (*validator.Result, error) {
	// Create new report
	_ = fetchedAt
	opts := validator.Options{}
	opts.ErrorLimit = 10
	v, err := validator.NewValidator(reader, opts)
	if err != nil {
		return nil, err
	}
	validationResult, err := v.Validate()
	if err != nil {
		return nil, err
	}
	if err := validator.SaveValidationReport(atx, validationResult, fvid, storage); err != nil {
		return nil, err
	}
	return validationResult, nil
}
