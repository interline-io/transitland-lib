package main

import (
	"flag"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tldb"
)

// copyCommand
type copyCommand struct {
	// Default options
	copier.Options
	// Typical DMFR options
	fvid       int
	create     bool
	extensions arrayFlags
	filters    arrayFlags
}

func (cmd *copyCommand) Run(args []string) error {
	fl := flag.NewFlagSet("copy", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: copy <reader> <writer>")
		fl.PrintDefaults()
	}
	fl.Var(&cmd.extensions, "ext", "Include GTFS Extension")
	fl.IntVar(&cmd.fvid, "fvid", 0, "Specify FeedVersionID when writing to a database")
	fl.BoolVar(&cmd.create, "create", false, "Create a basic database schema if none exists")
	// Copy options
	fl.BoolVar(&cmd.AllowEntityErrors, "allow-entity-errors", false, "Allow entities with errors to be copied")
	fl.BoolVar(&cmd.AllowReferenceErrors, "allow-reference-errors", false, "Allow entities with reference errors to be copied")
	//
	fl.Parse(args)
	if fl.NArg() < 2 {
		fl.Usage()
		log.Exit("Requires input reader and output writer")
	}
	// Reader / Writer
	reader := ext.MustGetReader(fl.Arg(0))
	defer reader.Close()
	writer := ext.MustGetWriter(fl.Arg(1), cmd.create)
	defer writer.Close()
	// Setup copier
	cp := copier.NewCopier(reader, writer)
	cp.AllowEntityErrors = cmd.AllowEntityErrors
	cp.AllowReferenceErrors = cmd.AllowReferenceErrors
	if dbw, ok := writer.(*tldb.Writer); ok {
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
	for _, extName := range cmd.extensions {
		e, err := ext.GetExtension(extName)
		if err != nil {
			log.Exit("No extension for: %s", extName)
		}
		cp.AddExtension(e)
		if cmd.create {
			if err := e.Create(writer); err != nil {
				log.Exit("Could not load extension: %s", err)
			}
		}
	}
	// Add filters
	for _, extName := range cmd.filters {
		ef, err := ext.GetEntityFilter(extName)
		if err != nil {
			log.Exit("No filter for '%s': %s", extName, err)
		}
		cp.AddEntityFilter(ef)
	}
	result := cp.Copy()
	result.DisplaySummary()
	return nil
}
