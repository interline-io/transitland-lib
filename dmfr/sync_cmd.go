package dmfr

import (
	"flag"
	"fmt"
	"os"

	"github.com/interline-io/gotransit/gtdb"
)

/////

type SyncCommand struct {
	dburl      string
	filenames  []string
	hideunseen bool
	adapter    gtdb.Adapter // allow for mocks
}

func (cmd *SyncCommand) Run(args []string) error {
	fl := flag.NewFlagSet("sync", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: sync <filenames...>")
		fl.PrintDefaults()
	}
	fl.StringVar(&cmd.dburl, "dburl", "", "Database URL (default: $DMFR_DATABASE_URL)")
	fl.BoolVar(&cmd.hideunseen, "hideunseen", false, "Hide unseen feeds")
	fl.Parse(args)
	cmd.filenames = fl.Args()
	if cmd.dburl == "" {
		cmd.dburl = os.Getenv("DMFR_DATABASE_URL")
	}
	if cmd.adapter == nil {
		writer := mustGetWriter(cmd.dburl, true)
		cmd.adapter = writer.Adapter
		defer writer.Close()
	}
	opts := SyncOptions{
		Filenames:  cmd.filenames,
		HideUnseen: cmd.hideunseen,
	}
	return cmd.adapter.Tx(func(atx gtdb.Adapter) error {
		_, err := MainSync(atx, opts)
		return err
	})
}
