package sync

import (
	"os"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/cli"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/spf13/cobra"
)

// Cobra setup

var pcmd = Command{}

var CobraCommand = &cobra.Command{
	Use:   "sync [flags] <filenames...>",
	Args:  cobra.MinimumNArgs(1),
	Short: "sync command",
	RunE:  cli.CobraHelper(&pcmd),
}

func init() {
	fl := CobraCommand.Flags()
	fl.Usage = func() {
		log.Print("Usage: sync <Filenames...>")
		fl.PrintDefaults()
	}
	fl.StringVar(&pcmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.BoolVar(&pcmd.HideUnseen, "hide-unseen", false, "Hide unseen feeds")
	fl.BoolVar(&pcmd.HideUnseenOperators, "hide-unseen-operators", false, "Hide unseen operators")
}

///////////////

// Command syncs a DMFR to a database.
type Command struct {
	DBURL   string
	Adapter tldb.Adapter
	Options
}

// Parse command line options.
func (cmd *Command) PreRunE(args []string) error {
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
