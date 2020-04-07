package dmfr

import (
	"flag"
	"fmt"
	"os"

	"github.com/interline-io/gotransit/gtdb"
)

// SyncCommand syncs a DMFR to a database.
type SyncCommand struct {
	DBURL      string
	Filenames  []string
	HideUnseen bool
	Adapter    gtdb.Adapter
}

// Parse command line options.
func (cmd *SyncCommand) Parse(args []string) error {
	fl := flag.NewFlagSet("sync", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: sync <Filenames...>")
		fl.PrintDefaults()
	}
	fl.StringVar(&cmd.DBURL, "DBURL", "", "Database URL (default: $DMFR_DATABASE_URL)")
	fl.BoolVar(&cmd.HideUnseen, "HideUnseen", false, "Hide unseen feeds")
	fl.Parse(args)
	cmd.Filenames = fl.Args()
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("DMFR_DATABASE_URL")
	}
	return nil
}

// Run this command.
func (cmd *SyncCommand) Run() error {
	if cmd.Adapter == nil {
		writer := mustGetWriter(cmd.DBURL, true)
		cmd.Adapter = writer.Adapter
		defer cmd.Adapter.Close()
	}
	opts := SyncOptions{
		Filenames:  cmd.Filenames,
		HideUnseen: cmd.HideUnseen,
	}
	return cmd.Adapter.Tx(func(atx gtdb.Adapter) error {
		_, err := MainSync(atx, opts)
		return err
	})
}