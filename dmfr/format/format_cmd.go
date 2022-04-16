package format

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/log"
)

// Command formats a DMFR file.
type Command struct {
	Filename string
	Save     bool
}

// Parse command line options.
func (cmd *Command) Parse(args []string) error {
	fl := flag.NewFlagSet("format", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: format <local filename>")
		fl.PrintDefaults()
	}
	fl.BoolVar(&cmd.Save, "save", false, "Save the formatted output back to the file")
	fl.Parse(args)
	if fl.NArg() == 0 {
		fl.Usage()
		return nil
	}
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
