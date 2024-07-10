package format

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/cli"
	"github.com/spf13/pflag"
)

// Command formats a DMFR file.
type Command struct {
	Filename string
	Save     bool
}

func (cmd *Command) AddFlags(fl *pflag.FlagSet) {
	fl.BoolVar(&cmd.Save, "save", false, "Save the formatted output back to the file")
}

// Parse command line options.
func (cmd *Command) Parse(args []string) error {
	fl := cli.NewNArgs(args)
	cmd.Filename = fl.Arg(0)
	return nil
}

// Run this command.
func (cmd *Command) Run() error {
	filename := cmd.Filename
	if filename == "" {
		return errors.New("must specify filename")
	}
	// First, validate DMFR
	_, err := dmfr.LoadAndParseRegistry(filename)
	if err != nil {
		log.Errorf("%s: Error when loading DMFR: %s", filename, err.Error())
	}

	// Re-read as raw registry
	r, err := os.Open(filename)
	if err != nil {
		return err
	}
	rr, err := dmfr.ReadRawRegistry(r)
	if err != nil {
		log.Errorf("%s: Error when loading DMFR: %s", filename, err.Error())
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
