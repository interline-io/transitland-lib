package cmds

import (
	"context"
	"errors"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/adapters/multireader"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/spf13/pflag"
)

// MergeCommand
type MergeCommand struct {
	Options     copier.Options
	readerPaths []string
	writerPath  string
}

func (cmd *MergeCommand) HelpDesc() (string, string) {
	return "Merge multiple GTFS feeds", ""
}

func (cmd *MergeCommand) HelpArgs() string {
	return "[flags] <writer> <readers...>"
}

func (cmd *MergeCommand) AddFlags(fl *pflag.FlagSet) {
}

func (cmd *MergeCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	if fl.NArg() < 2 {
		return errors.New("requires output writer and at least one reader")
	}
	cmd.writerPath = fl.Arg(0)
	cmd.readerPaths = fl.Args()[1:]
	return nil
}

func (cmd *MergeCommand) Run(ctx context.Context) error {
	var readers []adapters.Reader
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
