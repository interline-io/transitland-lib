package dmfr

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/log"
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
		r = &dmfrValidateCommand{}
	case "merge":
		r = &dmfrMergeCommand{}
	case "sync":
		r = &dmfrSyncCommand{}
	case "import":
		r = &dmfrImportCommand{}
	case "fetch":
		r = &dmfrFetchCommand{}
	default:
		return fmt.Errorf("Invalid command: %q", subc)
	}
	return r.Run(fl.Args()[1:]) // consume first arg
}

/////

type dmfrSyncCommand struct {
	dburl      string
	filenames  []string
	hideunseen bool
	adapter    gtdb.Adapter // allow for mocks
}

func (cmd *dmfrSyncCommand) Run(args []string) error {
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

/////

type dmfrValidateCommand struct{}

func (dmfrValidateCommand) Run(args []string) error {
	fl := flag.NewFlagSet("validate", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: validate <filenames...>")
		fl.PrintDefaults()
	}
	fl.Parse(args)
	if fl.NArg() == 0 {
		fl.Usage()
		return nil
	}
	filenames := fl.Args()
	errs := []error{}
	for _, filename := range filenames {
		log.Info("Loading DMFR: %s", filename)
		registry, err := LoadAndParseRegistry(filename)
		if err != nil {
			errs = append(errs, err)
			log.Info("%s: Error when loading DMFR: %s", filename, err.Error())
		} else {
			log.Info("%s: Success loading DMFR with %d feeds", filename, len(registry.Feeds))
		}
	}
	if len(errs) > 0 {
		return errors.New("")
	}
	return nil
}

/////

type dmfrMergeCommand struct{}

func (dmfrMergeCommand) Run(args []string) error {
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
