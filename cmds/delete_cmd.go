package cmds

import (
	"context"
	"errors"
	"os"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/importer"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/spf13/pflag"
)

// DeleteCommand deletes feed versions.
type DeleteCommand struct {
	FVArgs      FeedVersionArgs
	ExtraTables []string
	DryRun      bool
	DBURL       string
	Adapter     tldb.Adapter // allow for mocks
}

func (cmd *DeleteCommand) HelpDesc() (string, string) {
	return "Delete feed versions", ""
}

func (cmd *DeleteCommand) HelpArgs() string {
	return "[flags] <fvid>..."
}

func (cmd *DeleteCommand) AddFlags(fl *pflag.FlagSet) {
	cmd.FVArgs.AddFlags(fl)
	fl.StringSliceVar(&cmd.ExtraTables, "extra-table", nil, "Extra tables to delete feed_version_id")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	addDryRunFlag(fl, &cmd.DryRun, "Dry run; log the feed versions that would be deleted and exit")
}

// Parse command line flags
func (cmd *DeleteCommand) Parse(args []string) error {
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	if err := cmd.FVArgs.Parse(args); err != nil {
		return err
	}
	if cmd.FVArgs.Empty() {
		return errors.New("must provide at least one feed version id as an argument or with --fvid-file")
	}
	return nil
}

// Run this command
func (cmd *DeleteCommand) Run(ctx context.Context) error {
	if cmd.Adapter == nil {
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return err
		}
		cmd.Adapter = writer.Adapter
		defer writer.Close()
	}
	fvids, err := cmd.FVArgs.SelectIDs(ctx, cmd.Adapter)
	if err != nil {
		return err
	}
	for _, fvid := range fvids {
		if cmd.DryRun {
			log.For(ctx).Info().Int("feed_version_id", fvid).Msg("dry-run")
			continue
		}
		log.For(ctx).Info().Int("feed_version_id", fvid).Msg("begin")
		if err := cmd.Adapter.Tx(func(atx tldb.Adapter) error {
			return importer.DeleteFeedVersion(ctx, atx, fvid, cmd.ExtraTables)
		}); err != nil {
			log.For(ctx).Error().Err(err).Int("feed_version_id", fvid).Msg("failure")
			return err
		}
		log.For(ctx).Info().Int("feed_version_id", fvid).Msg("success")
	}
	return nil
}
