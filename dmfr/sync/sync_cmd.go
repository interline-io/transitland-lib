package sync

import (
	"os"

	"github.com/interline-io/transitland-lib/tldb"
	"github.com/spf13/pflag"
)

// Command syncs a DMFR to a database.
type Command struct {
	DBURL   string
	Adapter tldb.Adapter
	Options
}

func (cmd *Command) AddFlags(fl *pflag.FlagSet) {
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.BoolVar(&cmd.HideUnseen, "hide-unseen", false, "Hide unseen feeds")
	fl.BoolVar(&cmd.HideUnseenOperators, "hide-unseen-operators", false, "Hide unseen operators")
}

// Parse command line options.
func (cmd *Command) Parse(args []string) error {
	cmd.Filenames = args
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	return nil
}

// Run this command.
func (cmd *Command) Run() error {
	if cmd.Adapter == nil {
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return err
		}
		cmd.Adapter = writer.Adapter
		defer cmd.Adapter.Close()
	}
	return cmd.Adapter.Tx(func(atx tldb.Adapter) error {
		_, err := MainSync(atx, cmd.Options)
		return err
	})
}
