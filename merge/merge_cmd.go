package merge

import (
	"errors"
	"flag"

	"github.com/interline-io/transitland-lib/adapters/multi"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/log"
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
	fl := flag.NewFlagSet("copy", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: copy <reader> <writer>")
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
		// Reader / Writer
		reader, err := ext.NewReader(p)
		if err != nil {
			return err
		}
		readers = append(readers, reader)
	}

	reader := multi.NewReader(readers)
	if err := reader.Open(); err != nil {
		return err
	}

	defer reader.Close()
	writer, err := ext.OpenWriter(cmd.writerPath, true)
	if err != nil {
		return err
	}
	// if cmd.writeExtraColumns {
	// 	if v, ok := writer.(tl.WriterWithExtraColumns); ok {
	// 		v.WriteExtraColumns(true)
	// 	} else {
	// 		return errors.New("writer does not support extra output columns")
	// 	}
	// }
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
