package format

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/gjson_modifications"
	"github.com/interline-io/transitland-lib/log"
	"github.com/tidwall/gjson"
)

// Command formats a DMFR file.
type Command struct {
	Filename string
	Save bool
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
	if filename != "" {
		// first validate DMFR
		_, err := dmfr.LoadAndParseRegistry(filename)
		if err != nil {
			log.Errorf("%s: Error when loading DMFR: %s", filename, err.Error())
		}
		
		// load JSON
		var jsonFile *os.File
		if (cmd.Save) {
			jsonFile, err = os.OpenFile(filename, os.O_RDWR, 0644)
			if err != nil {
				log.Errorf("%s: Error when loading DMFR JSON: %s", filename, err.Error())
			}
		} else {
			jsonFile, err = os.Open(filename)
			if err != nil {
				log.Errorf("%s: Error when loading DMFR JSON: %s", filename, err.Error())
			}
		}
	
		defer jsonFile.Close()
		byteValue, _ := ioutil.ReadAll(jsonFile)

		// sort feeds by Onestop ID, sort all properties alphabetically, and pretty print with two-space indent
		gjson_modifications.AddSortModifier()
		formattedResult := gjson.Get(string(byteValue[:]), `@sort:{"array":"feeds","orderBy":"id","desc":false}.@pretty:{"sortKeys":true}`).Raw
		if (cmd.Save) {
			jsonFile.Truncate(0)
			jsonFile.Seek(0, 0)
			jsonFile.WriteString(formattedResult)
		} else {
			fmt.Print(formattedResult)
		}
	} else {
		log.Errorf("Must specify a filename")
	}
	return nil
}
