package main

import (
	"flag"

	"github.com/interline-io/gotransit/dmfr"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/log"
)

type dmfrCommand struct {
	args []string
}

func (cmd *dmfrCommand) run(args []string) {
	fl := flag.NewFlagSet("dmfr", flag.ExitOnError)
	fl.Parse(args)
	cmd.args = fl.Args()
	subc := args[0]
	switch subc {
	case "validate":
		cmd.cmdValidate(args[1:])
	case "merge":
		cmd.cmdMerge(args[1:])
	case "sync":
		cmd.cmdImport(args[1], args[2:])
	default:
		exit("Invalid subcommand: %q", subc)
	}
}

func (cmd *dmfrCommand) cmdImport(dburl string, filenames []string) {
	writer := MustGetDBWriter(dburl, true)
	log.Info("Syncing %d DMFRs to %s", len(filenames), dburl)
	writer.Adapter.Tx(func(atx gtdb.Adapter) error {
		dmfr.MainSync(atx, filenames)
		return nil
	})
}

func (cmd *dmfrCommand) cmdValidate(filenames []string) {
	for _, arg := range filenames {
		log.Info("Loading DMFR: %s", arg)
		registry, err := dmfr.LoadAndParseRegistry(arg)
		if err != nil {
			exit("Error when loading DMFR: %s", err)
		}
		log.Info("Success loading DMFR with %d feeds", len(registry.Feeds))
	}
}

func (cmd *dmfrCommand) cmdMerge(filenames []string) {
	exit("not implemented")
}
