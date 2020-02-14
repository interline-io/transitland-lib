package dmfr

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/interline-io/gotransit/gtdb"
)

// Command is the main entry point to the DMFR command
type Command struct {
	test int
}

// Run the DMFR command.
func (cmd *Command) Run(args []string) error {
	fl := flag.NewFlagSet("dmfr", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: dmfr <command> [<args>]")
		fmt.Println("dmfr commands:")
		fmt.Println("  validate")
		fmt.Println("  merge")
		fmt.Println("  sync")
		fmt.Println("  import")
		fmt.Println("  fetch")
		fl.PrintDefaults()
	}
	fl.Parse(args)
	subc := fl.Arg(0)
	if subc == "" {
		fl.Usage()
		return nil
	}
	type runner interface {
		Run([]string) error
	}
	var r runner
	switch subc {
	case "validate":
		r = &ValidateCommand{}
	case "merge":
		r = &MergeCommand{}
	case "sync":
		r = &SyncCommand{}
	case "import":
		r = &ImportCommand{}
	case "fetch":
		r = &FetchCommand{}
	default:
		return fmt.Errorf("Invalid command: %q", subc)
	}
	return r.Run(fl.Args()[1:]) // consume first arg
}

/////

// MergeCommand merges together multiple DMFR files. Not implemented.
type MergeCommand struct{}

// Run executes this command.
func (MergeCommand) Run(args []string) error {
	return errors.New("not implemented")
}

//// Util

// https://stackoverflow.com/questions/28322997/how-to-get-a-list-of-values-into-a-flag-in-golang/28323276#28323276
type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, ",")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// mustGetWriter opens & creates a db writer, panic on failure
func mustGetWriter(dburl string, create bool) *gtdb.Writer {
	// Writer
	writer, err := gtdb.NewWriter(dburl)
	if err != nil {
		panic(err)
	}
	if err := writer.Open(); err != nil {
		panic(err)
	}
	if create {
		if err := writer.Create(); err != nil {
			panic(err)
		}
	}
	return writer
}
