package cmds

import (
	"context"
	"os"

	"github.com/interline-io/transitland-lib/sync"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/spf13/pflag"
)

// SyncCommand syncs a DMFR to a database.
type SyncCommand struct {
	DBURL   string
	Adapter tldb.Adapter
	sync.Options
}

func (cmd *SyncCommand) HelpDesc() (string, string) {
	return "Sync DMFR files to database", ""
}

func (cmd *SyncCommand) HelpArgs() string {
	return "[flags] <filenames...>"
}

func (cmd *SyncCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.BoolVar(&cmd.HideUnseen, "hide-unseen", false, "Hide unseen feeds")
	fl.BoolVar(&cmd.HideUnseenOperators, "hide-unseen-operators", false, "Hide unseen operators")
}

// Parse command line options.
func (cmd *SyncCommand) Parse(args []string) error {
	cmd.Filenames = args
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	return nil
}

// Run this command.
func (cmd *SyncCommand) Run(ctx context.Context) error {
	if cmd.Adapter == nil {
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return err
		}
		cmd.Adapter = writer.Adapter
		defer cmd.Adapter.Close()
	}
	return cmd.Adapter.Tx(func(atx tldb.Adapter) error {
		_, err := sync.Sync(atx, cmd.Options)
		return err
	})
}
