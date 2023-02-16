package merge

import (
	"errors"
	"flag"
	"strings"

	"github.com/interline-io/transitland-lib/adapters/multireader"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/filters"
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

type splitPath struct {
	prefix string
	path   string
}

func (cmd *Command) Run() error {
	var splitPaths []splitPath
	pfx, _ := filters.NewPrefixFilter()
	for _, p := range cmd.readerPaths {
		a := strings.Split(p, ":")
		if len(a) >= 2 {
			splitPaths = append(splitPaths, splitPath{prefix: a[0], path: a[1]})
		} else {
			splitPaths = append(splitPaths, splitPath{prefix: p, path: p})
		}
	}

	var readers []tl.Reader
	for _, p := range splitPaths {
		// Reader / Writer
		reader, err := ext.NewReader(p.path)
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
	cp.AddExtension(pfx)

	result := cp.Copy()
	result.DisplaySummary()
	return nil
}
