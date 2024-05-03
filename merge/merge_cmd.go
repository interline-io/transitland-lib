package merge

import (
	"errors"
	"flag"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters/multireader"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/tl"
)

// Command
type Command struct {
	Options           copier.Options
	readerPaths       []string
	writerPath        string
	writeExtraColumns bool
}

func (cmd *Command) Parse(args []string) error {
	fl := flag.NewFlagSet("merge", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: merge <writer> [readers...]")
		fl.PrintDefaults()
	}
	fl.Parse(args)
	if fl.NArg() < 2 {
		fl.Usage()
		return errors.New("requires output writer and at least one reader")
	}
	cmd.writerPath = fl.Arg(0)
	cmd.readerPaths = fl.Args()[1:]
	return nil
}

func (cmd *Command) Run() error {
	var readers []tl.Reader
	for _, p := range cmd.readerPaths {
		// Open reader
		reader, err := ext.OpenReader(p)
		if err != nil {
			return err
		}
		readers = append(readers, reader)
	}

	reader := multireader.NewReader(readers...)
	if err := reader.Open(); err != nil {
		return err
	}

	defer reader.Close()
	writer, err := ext.OpenWriter(cmd.writerPath, true)
	if err != nil {
		return err
	}
	defer writer.Close()

	// Setup copier
	cp, err := copier.NewCopier(reader, writer, cmd.Options)
	if err != nil {
		return err
	}
	result := cp.Copy()
	result.DisplaySummary()
	return nil
}
