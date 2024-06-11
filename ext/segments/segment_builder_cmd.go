package segments

import (
	"errors"
	"flag"
	"log"

	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/tlcsv"
)

type SegmentBuilderCommand struct {
	infile  string
	outfile string
}

func (cmd *SegmentBuilderCommand) Parse(args []string) error {
	fl := flag.NewFlagSet("segment_debug", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: segment_debug <reader> <out.geojson>")
		log.Print("This command is experimental; it may provide incorrect results or crash on large feeds.")
		fl.PrintDefaults()
	}
	fl.Parse(args)
	if fl.NArg() < 1 {
		fl.Usage()
		return errors.New("requires input readers")
	}
	if fl.NArg() < 2 {
		fl.Usage()
		return errors.New("requires output geojson")
	}
	cmd.infile = fl.Arg(0)
	cmd.outfile = fl.Arg(1)
	return nil
}

func (cmd *SegmentBuilderCommand) Run() error {
	reader, err := tlcsv.NewReader(cmd.infile)
	if err != nil {
		return err
	}
	writer := direct.NewWriter()
	cp, err := copier.NewCopier(reader, writer, copier.Options{
		InterpolateStopTimes: true,
	})
	if err != nil {
		return err
	}
	e := NewSegmentBuilder()
	e.outfile = cmd.outfile
	cp.AddExtension(e)
	cpr := cp.Copy()
	if cpr.WriteError != nil {
		return err
	}
	return nil
}
