package dmfr

import (
	"flag"
	"os"

	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tldb"
)

// SyncCommand syncs a DMFR to a database.
type SyncCommand struct {
	DBURL      string
	Filenames  []string
	HideUnseen bool
	adapter    tldb.Adapter
}

// Parse command line options.
func (cmd *SyncCommand) Parse(args []string) error {
	fl := flag.NewFlagSet("sync", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: sync <Filenames...>")
		fl.PrintDefaults()
	}
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $DMFR_DATABASE_URL)")
	fl.BoolVar(&cmd.HideUnseen, "hide-unseen", false, "Hide unseen feeds")
	fl.Parse(args)
	cmd.Filenames = fl.Args()
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("DMFR_DATABASE_URL")
	}
	return nil
}

// Run this command.
func (cmd *SyncCommand) Run() error {
	if cmd.adapter == nil {
		writer := mustGetWriter(cmd.DBURL, true)
		cmd.adapter = writer.Adapter
		defer cmd.adapter.Close()
	}
	opts := SyncOptions{
		Filenames:  cmd.Filenames,
		HideUnseen: cmd.HideUnseen,
	}
	return cmd.adapter.Tx(func(atx tldb.Adapter) error {
		_, err := MainSync(atx, opts)
		return err
	})
}
