package lint

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/cmd/tlcli"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/pflag"
)

// Command lints a DMFR file.
type Command struct {
	Filenames []string
}

func (cmd *Command) HelpDesc() (string, string) {
	return "Lint DMFR files", ""
}

func (cmd *Command) HelpArgs() string {
	return "[flags] <filenames...>"
}

func (cmd *Command) AddFlags(fl *pflag.FlagSet) {
}

// Parse command line options.
func (cmd *Command) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	if fl.NArg() == 0 {
		return errors.New("at least one file required")
	}
	cmd.Filenames = fl.Args()
	return nil
}

// Run this command.
func (cmd *Command) Run() error {
	var fileErrors []string
	for _, filename := range cmd.Filenames {
		// first validate DMFR
		_, err := dmfr.LoadAndParseRegistry(filename)
		if err != nil {
			log.Errorf("%s: Error when loading DMFR: %s", filename, err.Error())
		}

		// Now load again as raw dmfr
		rawJson, err := ioutil.ReadFile(filename)
		rr, err := dmfr.ReadRawRegistry(bytes.NewBuffer(rawJson))
		if err != nil {
			log.Errorf("%s: Error when loading DMFR: %s", filename, err.Error())
		}
		var buf bytes.Buffer
		if err := rr.Write(&buf); err != nil {
			return err
		}

		// load JSON
		originalJsonString := string(rawJson)
		formattedJsonString := string(buf.Bytes())

		// Compare against input json
		if formattedJsonString != originalJsonString {
			err := fmt.Errorf("%s: not formatted correctly", filename)
			log.Errorf(err.Error())
			fileErrors = append(fileErrors, filename)
			// print out diff
			dmp := diffmatchpatch.New()
			diffs := dmp.DiffMain(originalJsonString, formattedJsonString, false)
			fmt.Println(dmp.DiffPrettyText(diffs))
		} else {
			log.Infof("%s: Formatted properly.", filename)
		}
	}
	if len(fileErrors) > 0 {
		return fmt.Errorf("Incorrectly formatted files: %s", strings.Join(fileErrors, ", "))

	}
	return nil
}
