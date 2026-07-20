package cmds

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	sq "github.com/irees/squirrel"
	"github.com/spf13/pflag"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/validator"
)

type RebuildStatsOptions struct {
	FeedVersionID           int
	Storage                 string
	ValidationReportStorage string
	SaveValidationReport    bool
	Stats                   []string
}

type RebuildStatsResult struct {
	Error error
}

// RebuildStatsCommand rebuilds feed version statistics
type RebuildStatsCommand struct {
	FVArgs  FeedVersionArgs
	Options RebuildStatsOptions
	Workers int
	DryRun  bool
	DBURL   string
	Adapter tldb.Adapter // allow for mocks
}

func (cmd *RebuildStatsCommand) HelpDesc() (string, string) {
	return "Rebuild statistics for feed versions", "With no feed version ids given, rebuilds stats for all feed versions."
}

func (cmd *RebuildStatsCommand) HelpArgs() string {
	return "[flags] [fvid...]"
}

func (cmd *RebuildStatsCommand) AddFlags(fl *pflag.FlagSet) {
	cmd.FVArgs.AddFlags(fl)
	fl.IntVar(&cmd.Workers, "workers", 1, "Worker threads")
	addDryRunFlag(fl, &cmd.DryRun, "Dry run; log the feed versions that would be rebuilt and exit")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.StringVar(&cmd.Options.Storage, "storage", "", "Storage destination; can be s3://... az://... or path to a directory")
	fl.BoolVar(&cmd.Options.SaveValidationReport, "validation-report", false, "Save validation report")
	fl.StringVar(&cmd.Options.ValidationReportStorage, "validation-report-storage", "", "Storage path for saving validation report JSON")
	fl.StringSliceVar(&cmd.Options.Stats, "stats", nil, "Subset of stats to rebuild (default all); valid: "+strings.Join(stats.AllStats, ","))
}

// Parse command line flags
func (cmd *RebuildStatsCommand) Parse(args []string) error {
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	if err := cmd.FVArgs.Parse(args); err != nil {
		return err
	}
	if err := stats.ValidateStatNames(cmd.Options.Stats); err != nil {
		return err
	}
	return nil
}

// Run this command
func (cmd *RebuildStatsCommand) Run(ctx context.Context) error {
	if cmd.Workers < 1 {
		cmd.Workers = 1
	}
	if cmd.Adapter == nil {
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return err
		}
		cmd.Adapter = writer.Adapter
		defer writer.Close()
	}
	// Match import_cmd's default-query filter: skip soft-deleted and storage-less FVs.
	q := cmd.Adapter.Sqrl().
		Select("feed_versions.id").
		From("feed_versions").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id").
		Where("current_feeds.deleted_at IS NULL").
		Where("feed_versions.deleted_at IS NULL").
		Where("feed_versions.sha1 <> ''").
		Where("feed_versions.file <> ''").
		OrderBy("feed_versions.id desc")
	sel := sq.Or{}
	if len(cmd.FVArgs.FVIDs) > 0 {
		sel = append(sel, sq.Eq{"feed_versions.id": cmd.FVArgs.FVIDs})
	}
	if len(cmd.FVArgs.FVSHA1) > 0 {
		sel = append(sel, sq.Eq{"feed_versions.sha1": cmd.FVArgs.FVSHA1})
	}
	if len(sel) > 0 {
		q = q.Where(sel)
	}
	qstr, qargs, err := q.ToSql()
	if err != nil {
		return err
	}
	qrs := []int{}
	err = cmd.Adapter.Select(ctx, &qrs, qstr, qargs...)
	if err != nil {
		return err
	}
	if expected := explicitSelectorCount(cmd.FVArgs.FVIDs, cmd.FVArgs.FVSHA1); expected > len(qrs) {
		log.For(ctx).Warn().
			Int("requested", expected).
			Int("processed", len(qrs)).
			Int("skipped", expected-len(qrs)).
			Msg("some explicitly requested feed versions were skipped (soft-deleted, missing sha1/file, or not found)")
	}
	///////////////
	// Here we go
	log.For(ctx).Info().Msgf("rebuilding stats for %d feed versions", len(qrs))
	jobs := make(chan RebuildStatsOptions, len(qrs))
	results := make(chan RebuildStatsResult, len(qrs))
	for _, fvid := range qrs {
		jobs <- RebuildStatsOptions{
			FeedVersionID:           fvid,
			Storage:                 cmd.Options.Storage,
			ValidationReportStorage: cmd.Options.ValidationReportStorage,
			SaveValidationReport:    cmd.Options.SaveValidationReport,
			Stats:                   cmd.Options.Stats,
		}
	}
	close(jobs)
	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < cmd.Workers; w++ {
		wg.Add(1)
		go rebuildStatsWorker(w, ctx, cmd.Adapter, cmd.DryRun, jobs, results, &wg)
	}
	wg.Wait()
	return nil
}

func rebuildStatsWorker(id int, ctx context.Context, adapter tldb.Adapter, dryrun bool, jobs <-chan RebuildStatsOptions, results chan<- RebuildStatsResult, wg *sync.WaitGroup) {
	_ = id
	type qr struct {
		FeedVersionID   int
		FeedID          int
		FeedOnestopID   string
		FeedVersionSHA1 string
	}
	for opts := range jobs {
		q := qr{}
		if err := adapter.Get(ctx, &q, "SELECT feed_versions.id as feed_version_id, feed_versions.feed_id as feed_id, feed_versions.sha1 as feed_version_sha1, current_feeds.onestop_id as feed_onestop_id FROM feed_versions INNER JOIN current_feeds ON current_feeds.id = feed_versions.feed_id WHERE feed_versions.id = ?", opts.FeedVersionID); err != nil {
			log.For(ctx).Error().Err(err).Int("feed_version_id", opts.FeedVersionID).Msg("could not get details")
			continue
		}
		jobLog := log.For(ctx).With().
			Str("feed_onestop_id", q.FeedOnestopID).
			Int("feed_id", q.FeedID).
			Str("feed_version_sha1", q.FeedVersionSHA1).
			Int("feed_version_id", q.FeedVersionID).
			Logger()
		if dryrun {
			jobLog.Info().Msg("dry-run")
			continue
		}
		jobLog.Info().Msg("begin")
		t := time.Now()
		result, err := rebuildStatsMain(ctx, adapter, opts)
		t2 := float64(time.Now().UnixNano()-t.UnixNano()) / 1e9 // 1000000000.0
		if err != nil {
			jobLog.Error().Err(err).Float64("duration", t2).Msg("critical failure, rolled back")
		} else {
			jobLog.Info().Float64("duration", t2).Msg("success")
		}
		results <- result
	}
	wg.Done()
}

func rebuildStatsMain(ctx context.Context, adapter tldb.Adapter, opts RebuildStatsOptions) (RebuildStatsResult, error) {
	// Get FV
	fv := dmfr.FeedVersion{}
	fv.ID = opts.FeedVersionID
	if err := adapter.Find(ctx, &fv); err != nil {
		return RebuildStatsResult{}, err
	}
	// Get Reader
	tladapter, err := tlcsv.NewStoreAdapter(ctx, opts.Storage, fv.File, fv.Fragment.Val)
	if err != nil {
		return RebuildStatsResult{}, err
	}
	reader, err := tlcsv.NewReaderFromAdapter(tladapter)
	if err != nil {
		return RebuildStatsResult{}, err
	}
	// Close removes the temp file downloaded by NewStoreAdapter; without this,
	// long-running batch rebuilds leak one zip per feed version and fill the disk
	defer reader.Close()
	if err := reader.Open(); err != nil {
		return RebuildStatsResult{}, err
	}
	// Save
	errImport := adapter.Tx(func(atx tldb.Adapter) error {
		if err := stats.CreateFeedStats(ctx, atx, reader, fv.ID, stats.WriteOptions{Stats: opts.Stats}); err != nil {
			return err
		}
		if opts.SaveValidationReport {
			if _, err := createFeedValidationReport(ctx, atx, reader, fv.ID, fv.FetchedAt, opts.ValidationReportStorage); err != nil {
				return err
			}
		}
		return nil
	})
	return RebuildStatsResult{}, errImport
}

func createFeedValidationReport(ctx context.Context, atx tldb.Adapter, reader *tlcsv.Reader, fvid int, fetchedAt time.Time, storage string) (*validator.Result, error) {
	// Create new report
	_ = fetchedAt
	opts := validator.Options{}
	opts.ErrorLimit = 10
	v, err := validator.NewValidator(reader, opts)
	if err != nil {
		return nil, err
	}
	validationResult, err := v.Validate(ctx)
	if err != nil {
		return nil, err
	}
	if err := validator.SaveValidationReport(ctx, atx, validationResult, fvid, storage); err != nil {
		return nil, err
	}
	return validationResult, nil
}
