package dmfr

import (
	"errors"
	"flag"
	"fmt"
	"log"

	fetch "github.com/interline-io/transitland-lib/dmfr/fetch"
	"github.com/interline-io/transitland-lib/dmfr/importer"
	"github.com/interline-io/transitland-lib/dmfr/sync"
	"github.com/interline-io/transitland-lib/dmfr/validate"
)

// Command is the main entry point to the DMFR command
type Command struct {
	test int
}

// Run the DMFR command.
func (cmd *Command) Run(args []string) error {
	fl := flag.NewFlagSet("dmfr", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: dmfr <command> [<args>]")
		log.Print("dmfr commands:")
		log.Print("  validate")
		log.Print("  merge")
		log.Print("  sync")
		log.Print("  import")
		log.Print("  fetch")
		log.Print("  recalculate")
		fl.PrintDefaults()
	}
	fl.Parse(args)
	subc := fl.Arg(0)
	if subc == "" {
		fl.Usage()
		return nil
	}
	type runner interface {
		Parse([]string) error
		Run() error
	}
	var r runner
	switch subc {
	case "validate":
		r = &validate.Command{}
	case "sync":
		r = &sync.Command{}
	case "import":
		r = &importer.Command{}
	case "fetch":
		r = &fetch.Command{}
	// case "recalculate":
	// 	r = &RecalculateCommand{}
	default:
		return fmt.Errorf("Invalid command: %q", subc)
	}
	// Parse; consume first arg
	if err := r.Parse(fl.Args()[1:]); err != nil {
		return err
	}
	return r.Run()
}

/////

// MergeCommand merges together multiple DMFR files. Not implemented.
type MergeCommand struct{}

// Parse command line options
func (MergeCommand) Parse(args []string) error {
	return errors.New("not implemented")
}

// Run executes this command.
func (MergeCommand) Run() error {
	return nil
}
