package copier

import (
	"errors"
	"fmt"

	"github.com/interline-io/transitland-lib/cmd/tlcli"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/spf13/pflag"
)

// Command
type Command struct {
	Options
	fvid              int
	create            bool
	extensions        []string
	readerPath        string
	writerPath        string
	writeExtraColumns bool
}

func (cmd *Command) HelpDesc() (string, string) {
	return "Copy feed", ""
}

func (cmd *Command) HelpArgs() string {
	return "[flags] <reader> <writer>"
}

func (cmd *Command) AddFlags(fl *pflag.FlagSet) {
	fl.StringSliceVar(&cmd.extensions, "ext", nil, "Include GTFS Extension")
	fl.IntVar(&cmd.fvid, "fvid", 0, "Specify FeedVersionID when writing to a database")
	fl.BoolVar(&cmd.create, "create", false, "Create a basic database schema if none exists")
	fl.BoolVar(&cmd.CopyExtraFiles, "write-extra-files", false, "Copy additional files found in source to destination")
	fl.BoolVar(&cmd.writeExtraColumns, "write-extra-columns", false, "Include extra columns in output")
	fl.BoolVar(&cmd.AllowEntityErrors, "allow-entity-errors", false, "Allow entities with errors to be copied")
	fl.BoolVar(&cmd.AllowReferenceErrors, "allow-reference-errors", false, "Allow entities with reference errors to be copied")
	fl.IntVar(&cmd.Options.ErrorLimit, "error-limit", 1000, "Max number of detailed errors per error group")
}

func (cmd *Command) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	if fl.NArg() < 2 {
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
	if cmd.writeExtraColumns {
		if v, ok := writer.(tl.WriterWithExtraColumns); ok {
			v.WriteExtraColumns(true)
		} else {
			return errors.New("writer does not support extra output columns")
		}
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
	result.DisplayErrors()
	result.DisplayWarnings()
	return nil
}
