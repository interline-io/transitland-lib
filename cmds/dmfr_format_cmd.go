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

// DmfrFormatCommand formats a DMFR file.
type DmfrFormatCommand struct {
	Filename string
	Save     bool
}

func (cmd *DmfrFormatCommand) HelpDesc() (string, string) {
	return "Format a DMFR file", ""
}

func (cmd *DmfrFormatCommand) HelpArgs() string {
	return "[flags] <filename>"
}

func (cmd *DmfrFormatCommand) AddFlags(fl *pflag.FlagSet) {
	fl.BoolVar(&cmd.Save, "save", false, "Save the formatted output back to the file")
}

// Parse command line options.
func (cmd *DmfrFormatCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	cmd.Filename = fl.Arg(0)
	return nil
}

// Run this command.
func (cmd *DmfrFormatCommand) Run(ctx context.Context) error {
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
	defer r.Close()
	rr, err := dmfr.ReadRawRegistry(r)
	if err != nil {
		log.For(ctx).Error().Msgf("%s: Error when loading DMFR: %s", filename, err.Error())
		return err
	}
	var buf bytes.Buffer
	if err := rr.Write(&buf); err != nil {
		return err
	}
	byteValue := buf.Bytes()
	if cmd.Save {
		if err := os.WriteFile(filename, byteValue, 0644); err != nil {
			return err
		}
	} else {
		fmt.Println(string(byteValue))
	}
	return nil
}
