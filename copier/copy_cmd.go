package copier

import (
	"errors"
	"flag"
	"fmt"

	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/internal/cli"
	"github.com/interline-io/transitland-lib/log"
	"github.com/interline-io/transitland-lib/tldb"
)

// Command
type Command struct {
	Options
	fvid       int
	create     bool
	extensions cli.ArrayFlags
	readerPath string
	writerPath string
}

func (cmd *Command) Parse(args []string) error {
	fl := flag.NewFlagSet("copy", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: copy <reader> <writer>")
		fl.PrintDefaults()
	}
	fl.Var(&cmd.extensions, "ext", "Include GTFS Extension")
	fl.IntVar(&cmd.fvid, "fvid", 0, "Specify FeedVersionID when writing to a database")
	fl.BoolVar(&cmd.create, "create", false, "Create a basic database schema if none exists")
	fl.BoolVar(&cmd.AllowEntityErrors, "allow-entity-errors", false, "Allow entities with errors to be copied")
	fl.BoolVar(&cmd.AllowReferenceErrors, "allow-reference-errors", false, "Allow entities with reference errors to be copied")
	fl.Parse(args)
	if fl.NArg() < 2 {
		fl.Usage()
		return errors.New("requires input reader and output writer")
	}
	cmd.readerPath = fl.Arg(0)
	cmd.writerPath = fl.Arg(1)
	return nil
}

func (cmd *Command) Run() error {
	// Reader / Writer
	reader, err := ext.OpenReader(cmd.readerPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	writer, err := ext.OpenWriter(cmd.writerPath, cmd.create)
	if err != nil {
		return err
	}
	defer writer.Close()
	// Create feed version
	if dbw, ok := writer.(*tldb.Writer); ok {
		if cmd.fvid != 0 {
			dbw.FeedVersionID = cmd.fvid
		}
		if dbw.FeedVersionID == 0 {
			fvid, err := dbw.CreateFeedVersion(reader)
			if err != nil {
				return fmt.Errorf("error creating feed version: %s", err.Error())
			}
			dbw.FeedVersionID = fvid
		}
		cmd.Options.NormalizeServiceIDs = true
	}
	// Setup copier
	cmd.Options.Extensions = cmd.extensions
	cp, err := NewCopier(reader, writer, cmd.Options)
	if err != nil {
		return err
	}
	result := cp.Copy()
	result.DisplaySummary()
	return nil
}
