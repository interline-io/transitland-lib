package cmds

import (
	"context"
	"os"
	"sync"
	"time"

	sq "github.com/irees/squirrel"
	"github.com/spf13/pflag"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/ext/builders"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
)

// GeohashBackfillCommand computes per-FV stop geohash cells for feed versions
// that were fetched before the fetch-time hook existed, or for any FV missing
// rows in tl_feed_version_geohashes.
type GeohashBackfillCommand struct {
	Workers  int
	Storage  string
	DBURL    string
	FVIDs    []string
	Force    bool
	Adapter  tldb.Adapter
	fvidfile string
}

func (cmd *GeohashBackfillCommand) HelpDesc() (string, string) {
	return "Compute per-FV stop geohash cells for feed versions missing them",
		"Walks feed versions with no rows in tl_feed_version_geohashes, opens each cached zip, runs the FeedVersionGeohashBuilder, and inserts the resulting cells. Use --force to recompute and replace existing rows."
}

func (cmd *GeohashBackfillCommand) HelpArgs() string {
	return "[flags]"
}

func (cmd *GeohashBackfillCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringSliceVar(&cmd.FVIDs, "fvid", nil, "Backfill cells for a specific feed version ID")
	fl.StringVar(&cmd.fvidfile, "fvid-file", "", "Specify feed version IDs in file, one per line")
	fl.IntVar(&cmd.Workers, "workers", 1, "Worker threads")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.StringVar(&cmd.Storage, "storage", "", "Storage destination; can be s3://... az://... or path to a directory")
	fl.BoolVar(&cmd.Force, "force", false, "Recompute and replace cells even if they already exist")
}

func (cmd *GeohashBackfillCommand) Parse(args []string) error {
	_ = tlcli.NewNArgs(args)
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
	return nil
}

func (cmd *GeohashBackfillCommand) Run(ctx context.Context) error {
	if cmd.Adapter == nil {
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return err
		}
		cmd.Adapter = writer.Adapter
		defer writer.Close()
	}
	q := cmd.Adapter.Sqrl().
		Select("feed_versions.id").
		From("feed_versions").
		OrderBy("feed_versions.id desc")
	if len(cmd.FVIDs) > 0 {
		q = q.Where(sq.Eq{"feed_versions.id": cmd.FVIDs})
	}
	if !cmd.Force {
		q = q.Where("NOT EXISTS (SELECT 1 FROM tl_feed_version_geohashes WHERE feed_version_id = feed_versions.id)")
	}
	qstr, qargs, err := q.ToSql()
	if err != nil {
		return err
	}
	var fvids []int
	if err := cmd.Adapter.Select(ctx, &fvids, qstr, qargs...); err != nil {
		return err
	}
	log.For(ctx).Info().Msgf("Backfilling geohashes for %d feed versions", len(fvids))

	jobs := make(chan int, len(fvids))
	for _, id := range fvids {
		jobs <- id
	}
	close(jobs)

	var wg sync.WaitGroup
	for w := 0; w < cmd.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fvid := range jobs {
				t := time.Now()
				if err := geohashBackfillOne(ctx, cmd.Adapter, cmd.Storage, fvid); err != nil {
					log.For(ctx).Error().Err(err).Msgf("feed_version %d: failed (%0.2fs)", fvid, time.Since(t).Seconds())
					continue
				}
				log.For(ctx).Info().Msgf("feed_version %d: success (%0.2fs)", fvid, time.Since(t).Seconds())
			}
		}()
	}
	wg.Wait()
	return nil
}

func geohashBackfillOne(ctx context.Context, adapter tldb.Adapter, storage string, fvid int) error {
	fv := dmfr.FeedVersion{}
	fv.ID = fvid
	if err := adapter.Find(ctx, &fv); err != nil {
		return err
	}
	tladapter, err := tlcsv.NewStoreAdapter(ctx, storage, fv.File, fv.Fragment.Val)
	if err != nil {
		return err
	}
	reader, err := tlcsv.NewReaderFromAdapter(tladapter)
	if err != nil {
		return err
	}
	if err := reader.Open(); err != nil {
		return err
	}
	defer reader.Close()

	b := builders.NewFeedVersionGeohashBuilder()
	if _, err := copier.QuietCopy(ctx, reader, &empty.Writer{}, func(o *copier.Options) {
		o.AddExtension(b)
	}); err != nil {
		return err
	}
	return adapter.Tx(func(atx tldb.Adapter) error {
		return stats.WriteFeedVersionGeohashes(ctx, atx, fv.ID, b.Cells())
	})
}
