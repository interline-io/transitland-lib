package sync

import (
	"flag"
	"os"

	"github.com/interline-io/transitland-lib/log"
	"github.com/interline-io/transitland-lib/tldb"
)

// Command syncs a DMFR to a database.
type Command struct {
	DBURL   string
	Adapter tldb.Adapter
	Options
}

// Parse command line options.
func (cmd *Command) Parse(args []string) error {
	fl := flag.NewFlagSet("sync", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: sync <Filenames...>")
		fl.PrintDefaults()
	}
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.BoolVar(&cmd.HideUnseen, "hide-unseen", false, "Hide unseen feeds")
	fl.BoolVar(&cmd.HideUnseenOperators, "hide-unseen-operators", false, "Hide unseen operators")
	fl.Parse(args)
	cmd.Filenames = fl.Args()
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	return nil
}

// Run this command.
func (cmd *Command) Run() error {
	if cmd.Adapter == nil {
		writer := tldb.MustGetWriter(cmd.DBURL, true)
		cmd.Adapter = writer.Adapter
		defer cmd.Adapter.Close()
	}
	return cmd.Adapter.Tx(func(atx tldb.Adapter) error {
		_, err := MainSync(atx, cmd.Options)
		return err
	})
}
