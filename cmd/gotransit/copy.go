package main

import (
	"flag"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/copier"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/log"
)

// basicCopyOptions
type basicCopyOptions struct {
	fvid                 int
	create               bool
	allowEntityErrors    bool
	allowReferenceErrors bool
	extensions           arrayFlags
	filters              arrayFlags
}

// copyCommand
type copyCommand struct {
	basicCopyOptions
}

func (cmd *copyCommand) Run(args []string) error {
	fl := flag.NewFlagSet("copy", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: copy <reader> <writer>")
		fl.PrintDefaults()
	}
	fl.BoolVar(&cmd.allowEntityErrors, "allow-entity-errors", false, "Allow entities with errors to be copied")
	fl.BoolVar(&cmd.allowReferenceErrors, "allow-reference-errors", false, "Allow entities with reference errors to be copied")
	fl.Var(&cmd.extensions, "ext", "Include GTFS Extension")
	fl.IntVar(&cmd.fvid, "fvid", 0, "Specify FeedVersionID when writing to a database")
	fl.BoolVar(&cmd.create, "create", false, "Create a basic database schema if none exists")
	fl.Parse(args)
	if fl.NArg() < 2 {
		fl.Usage()
		log.Exit("Requires input reader and output writer")
	}
	// Reader / Writer
	reader := MustGetReader(fl.Arg(0))
	defer reader.Close()
	writer := MustGetWriter(fl.Arg(1), cmd.create)
	defer writer.Close()
	// Setup copier
	cp := copier.NewCopier(reader, writer)
	cp.AllowEntityErrors = cmd.allowEntityErrors
	cp.AllowReferenceErrors = cmd.allowReferenceErrors
	if dbw, ok := writer.(*gtdb.Writer); ok {
		if cmd.fvid != 0 {
			dbw.FeedVersionID = cmd.fvid
		} else {
			fvid, err := dbw.CreateFeedVersion(reader)
			if err != nil {
				log.Exit("Error creating FeedVersion: %s", err)
			}
			dbw.FeedVersionID = fvid
		}
		cp.NormalizeServiceIDs = true
	}
	for _, ext := range cmd.extensions {
		e, err := gotransit.GetExtension(ext)
		if err != nil {
			log.Exit("No extension for: %s", ext)
		}
		cp.AddExtension(e)
		if cmd.create {
			if err := e.Create(writer); err != nil {
				log.Exit("Could not load extension: %s", err)
			}
		}
	}
	// Add filters
	for _, ext := range cmd.filters {
		ef, err := gotransit.GetEntityFilter(ext)
		if err != nil {
			log.Exit("No filter for '%s': %s", ext, err)
		}
		cp.AddEntityFilter(ef)
	}
	result := cp.Copy()
	result.DisplaySummary()
	return nil
}
