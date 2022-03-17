package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/interline-io/transitland-lib/dmfr/fetch"
	"github.com/interline-io/transitland-lib/dmfr/format"
	"github.com/interline-io/transitland-lib/dmfr/importer"
	"github.com/interline-io/transitland-lib/dmfr/lint"
	"github.com/interline-io/transitland-lib/dmfr/sync"
	"github.com/interline-io/transitland-lib/dmfr/unimporter"
	"github.com/interline-io/transitland-lib/dmfr/validate"
)

// dmfrCommand is the main entry point to the DMFR command
type dmfrCommand struct {
	subcommand runner
}

func (cmd *dmfrCommand) Parse(args []string) error {
	fl := flag.NewFlagSet("dmfr", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: dmfr <command> [<args>]")
		log.Print("dmfr commands:")
		log.Print("  validate")
		log.Print("  format")
		log.Print("  sync")
		log.Print("  import")
		log.Print("  unimport")
		log.Print("  fetch")
		fl.PrintDefaults()
	}
	fl.Parse(args)
	subc := fl.Arg(0)
	if subc == "" {
		fl.Usage()
		return nil
	}
	subargs := fl.Args()[1:]
	switch subc {
	case "validate":
		cmd.subcommand = &validate.Command{}
	case "format":
		cmd.subcommand = &format.Command{}
	case "lint":
		cmd.subcommand = &lint.Command{}
	case "sync":
		cmd.subcommand = &sync.Command{}
	case "import":
		cmd.subcommand = &importer.Command{}
	case "unimport":
		cmd.subcommand = &unimporter.Command{}
	case "fetch":
		cmd.subcommand = &fetch.Command{}
	default:
		return fmt.Errorf("invalid command: %q", subc)
	}
	return cmd.subcommand.Parse(subargs)
}

// Run the DMFR command.
func (cmd *dmfrCommand) Run() error {
	return cmd.subcommand.Run()
}
