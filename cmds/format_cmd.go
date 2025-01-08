package cmds

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/spf13/pflag"
)

// FormatCommand formats a DMFR file.
type FormatCommand struct {
	Filename string
	Save     bool
}

func (cmd *FormatCommand) HelpDesc() (string, string) {
	return "Format a DMFR file", ""
}

func (cmd *FormatCommand) HelpArgs() string {
	return "[flags] <filename>"
}

func (cmd *FormatCommand) AddFlags(fl *pflag.FlagSet) {
	fl.BoolVar(&cmd.Save, "save", false, "Save the formatted output back to the file")
}

// Parse command line options.
func (cmd *FormatCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	cmd.Filename = fl.Arg(0)
	return nil
}

// Run this command.
func (cmd *FormatCommand) Run(ctx context.Context) error {
	filename := cmd.Filename
	if filename == "" {
		return errors.New("must specify filename")
	}
	// First, validate DMFR
	_, err := dmfr.LoadAndParseRegistry(filename)
	if err != nil {
		log.For(ctx).Error().Msgf("%s: Error when loading DMFR: %s", filename, err.Error())
	}

	// Re-read as raw registry
	r, err := os.Open(filename)
	if err != nil {
		return err
	}
	rr, err := dmfr.ReadRawRegistry(r)
	if err != nil {
		log.For(ctx).Error().Msgf("%s: Error when loading DMFR: %s", filename, err.Error())
	}
	var buf bytes.Buffer
	if err := rr.Write(&buf); err != nil {
		return err
	}
	byteValue := buf.Bytes()
	if cmd.Save {
		// Write json
		f, err := os.Create(filename)
		if err != nil {
			return err
		}
		if _, err := f.Write(byteValue); err != nil {
			return err
		}
	} else {
		// Print
		fmt.Println(string(byteValue))
	}
	return nil
}
