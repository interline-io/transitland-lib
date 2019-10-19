package main

import (
	"flag"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/copier"
	"github.com/interline-io/gotransit/gtdb"
)

// basicCopyOptions
type basicCopyOptions struct {
	fvid                 int
	newfv                bool
	create               bool
	allowEntityErrors    bool
	allowReferenceErrors bool
	extensions           arrayFlags
	filters              arrayFlags
	args                 []string
}

// copyCommand
type copyCommand struct {
	basicCopyOptions
}

func (cmd *copyCommand) run(args []string) {
	fl := flag.NewFlagSet("copy", flag.ExitOnError)
	fl.Var(&cmd.extensions, "ext", "Include GTFS Extension")
	fl.IntVar(&cmd.fvid, "fvid", 0, "Specify FeedVersionID")
	fl.BoolVar(&cmd.newfv, "newfv", false, "Create a new FeedVersion from Reader")
	fl.BoolVar(&cmd.create, "create", false, "Create")
	fl.BoolVar(&cmd.allowEntityErrors, "allow-entity-errors", false, "Allow entity-level errors")
	fl.BoolVar(&cmd.allowReferenceErrors, "allow-reference-errors", false, "Allow reference errors")
	fl.Parse(args)
	cmd.args = fl.Args()
	if len(cmd.args) < 2 {
		exit("Requires input and output")
	}
	// Reader / Writer
	reader := MustGetReader(cmd.args[0])
	defer reader.Close()
	writer := MustGetWriter(cmd.args[1], cmd.create)
	defer writer.Close()
	// Setup copier
	cp := copier.NewCopier(reader, writer)
	cp.AllowEntityErrors = cmd.allowEntityErrors
	cp.AllowReferenceErrors = cmd.allowReferenceErrors
	if dbw, ok := writer.(*gtdb.Writer); ok {
		if cmd.fvid != 0 {
			dbw.FeedVersionID = cmd.fvid
		} else if cmd.newfv {
			if _, err := dbw.CreateFeedVersion(reader); err != nil {
				exit("Error creating FeedVersion: %s", err)
			}
		}
		cp.NormalizeServiceIDs = true
	}
	for _, ext := range cmd.extensions {
		e, err := gotransit.GetExtension(ext)
		if err != nil {
			exit("No extension for: %s", ext)
		}
		cp.AddExtension(e)
		if cmd.create {
			if err := e.Create(writer); err != nil {
				exit("%s", err)
			}
		}
	}
	// Add filters
	for _, ext := range cmd.filters {
		ef, err := gotransit.GetEntityFilter(ext)
		if err != nil {
			exit("No filter for '%s': %s", ext, err)
		}
		cp.AddEntityFilter(ef)
	}
	cp.Copy()
}
