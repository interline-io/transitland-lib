package cmds

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters/tldb"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/unimporter"
	"github.com/spf13/pflag"
)

type DeleteCommand struct {
	ExtraTables []string
	DryRun      bool
	FVID        int
	DBURL       string
	Adapter     tldb.Adapter // allow for mocks
}

func (cmd *DeleteCommand) HelpDesc() (string, string) {
	return "Delete feed versions", ""
}

func (cmd *DeleteCommand) HelpArgs() string {
	return "[flags] <fvid>"
}

func (cmd *DeleteCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringSliceVar(&cmd.ExtraTables, "extra-table", nil, "Extra tables to delete feed_version_id")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.BoolVar(&cmd.DryRun, "dryrun", false, "Dry run; print feeds that would be imported and exit")
}

// Parse command line flags
func (cmd *DeleteCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	if fl.NArg() != 1 {
		return errors.New("must provide exactly one feed version id")
	}
	sid := fl.Arg(0)
	var err error
	cmd.FVID, err = strconv.Atoi(sid)
	if err != nil {
		return fmt.Errorf("could not parse '%s' as int", sid)
	}
	return nil
}

// Run this command
func (cmd *DeleteCommand) Run() error {
	if cmd.Adapter == nil {
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return err
		}
		cmd.Adapter = writer.Adapter
		defer writer.Close()
	}
	qrs := 0
	q := cmd.Adapter.Sqrl().
		Select("feed_versions.id").
		From("feed_versions").
		Where(sq.Eq{"feed_versions.id": cmd.FVID})
	qstr, qargs, err := q.ToSql()
	if err != nil {
		return err
	}
	err = cmd.Adapter.Get(&qrs, qstr, qargs...)
	if err == sql.ErrNoRows {
		return fmt.Errorf("feed version %d does not exist", cmd.FVID)
	} else if err != nil {
		return err
	}
	if cmd.DryRun {
		log.Info().Msgf("Deleting feed version: %d (dry run)", cmd.FVID)
	} else {
		log.Info().Msgf("Deleting feed version: %d", cmd.FVID)
		err := cmd.Adapter.Tx(func(atx tldb.Adapter) error {
			return unimporter.DeleteFeedVersion(cmd.Adapter, cmd.FVID, cmd.ExtraTables)
		})
		if err != nil {
			return err
		}
	}
	return nil
}
