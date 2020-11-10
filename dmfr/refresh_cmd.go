package dmfr

import (
	"flag"
	"os"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
)

// RefreshOptions .
type RefreshOptions struct {
	FeedVersionID int
	Directory     string
	S3            string
}

// RefreshCommand updates statistics for a Feed Version
type RefreshCommand struct {
	Workers        int
	DBURL          string
	FVIDs          arrayFlags
	FVSHA1         arrayFlags
	Adapter        tldb.Adapter // allow for mocks
	Limit          int
	DryRun         bool
	RefreshOptions RefreshOptions
}

// Parse command line flags
func (cmd *RefreshCommand) Parse(args []string) error {
	fvidfile := ""
	fvsha1file := ""
	fl := flag.NewFlagSet("refresh", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: refresh [feedids...]")
		fl.PrintDefaults()
	}
	fl.Var(&cmd.FVIDs, "fvid", "Import specific feed version ID")
	fl.StringVar(&fvidfile, "fvid-file", "", "Specify feed version IDs in file, one per line; equivalent to multiple --fvid")
	fl.StringVar(&fvsha1file, "fv-sha1-file", "", "Specify feed version IDs by SHA1 in file, one per line")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $DMFR_DATABASE_URL)")
	fl.StringVar(&cmd.RefreshOptions.Directory, "gtfsdir", ".", "GTFS Directory")
	fl.StringVar(&cmd.RefreshOptions.S3, "s3", "", "Get GTFS files from S3 bucket/prefix")
	fl.IntVar(&cmd.Limit, "limit", 0, "Refresh at most n feed versions")
	fl.Parse(args)
	cmd.FVIDs = fl.Args()
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("DMFR_DATABASE_URL")
	}
	if fvidfile != "" {
		lines, err := getFileLines(fvidfile)
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
		lines, err := getFileLines(fvsha1file)
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
func (cmd *RefreshCommand) Run() error {
	if cmd.Adapter == nil {
		writer := mustGetWriter(cmd.DBURL, true)
		cmd.Adapter = writer.Adapter
		defer writer.Close()
	}
	// Query to get FVs to import
	q := cmd.Adapter.Sqrl().
		Select("feed_versions.id").
		From("feed_versions").
		OrderBy("feed_versions.id")
	if len(cmd.FVIDs) > 0 {
		// Explicitly specify fvids
		q = q.Where(sq.Eq{"feed_versions.id": cmd.FVIDs})
	}
	if len(cmd.FVSHA1) > 0 {
		// Explicitly specify fv sha1
		q = q.Where(sq.Eq{"feed_versions.sha1": cmd.FVSHA1})
	}
	if cmd.Limit > 0 {
		// Max feeds
		q = q.Limit(uint64(cmd.Limit))
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
	log.Info("Refreshing %d feed versions", len(qrs))
	for _, fvid := range qrs {
		log.Info("Feed Version: %d", fvid)
		err := cmd.Adapter.Tx(func(atx tldb.Adapter) error {
			return dmfrRefresh(atx, RefreshOptions{
				FeedVersionID: fvid,
				Directory:     cmd.RefreshOptions.Directory,
				S3:            cmd.RefreshOptions.S3,
			})
		})
		if err != nil {
			log.Error("Could not refresh, skipping: %s", err.Error())
		}
	}
	return nil
}

func dmfrRefresh(adapter tldb.Adapter, opts RefreshOptions) error {
	// Get FV
	fv := tl.FeedVersion{ID: opts.FeedVersionID}
	if err := adapter.Find(&fv); err != nil {
		return err
	}
	// Get reader
	reader, err := tlcsv.NewReader(dmfrGetReaderURL(opts.S3, opts.Directory, fv.File))
	if err != nil {
		return err
	}
	if err := reader.Open(); err != nil {
		return err
	}
	defer reader.Close()
	// Delete file infos
	if _, err := adapter.Sqrl().Delete(FeedVersionFileInfo{}.TableName()).Where(sq.Eq{"feed_version_id": fv.ID}).Exec(); err != nil {
		return err
	}
	// Delete service levels
	if _, err := adapter.Sqrl().Delete(FeedVersionServiceLevel{}.TableName()).Where(sq.Eq{"feed_version_id": fv.ID}).Exec(); err != nil {
		return err
	}
	// Update stats
	if err := createFeedStats(adapter, reader, fv.ID); err != nil {
		return err
	}
	return nil
}
