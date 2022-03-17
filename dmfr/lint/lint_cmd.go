package lint

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/gjson_modifications"
	"github.com/interline-io/transitland-lib/log"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/tidwall/gjson"
)

// Command merges two or more DMFR files.
type Command struct {
	Filenames []string
}

// Parse command line options.
func (cmd *Command) Parse(args []string) error {
	fl := flag.NewFlagSet("format", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: lint <one or more filenames...>")
		fl.PrintDefaults()
	}
	fl.Parse(args)
	if fl.NArg() == 0 {
		fl.Usage()
		return nil
	}
	cmd.Filenames = fl.Args()
	return nil
}

// Run this command.
func (cmd *Command) Run() error {
	for _, filename := range cmd.Filenames {
		// first validate DMFR
		_, err := dmfr.LoadAndParseRegistry(filename)
		if err != nil {
			log.Errorf("%s: Error when loading DMFR: %s", filename, err.Error())
		}

		// load JSON
		jsonFile, err := os.Open(filename)
		if err != nil {
			log.Errorf("%s: Error when loading DMFR JSON: %s", filename, err.Error())
		}
		defer jsonFile.Close()
		byteValue, _ := ioutil.ReadAll(jsonFile)
		originalJsonString := string(byteValue[:])

		// sort feeds by Onestop ID, sort all properties alphabetically, and pretty print with two-space indent
		gjson_modifications.AddSortModifier()
		formattedJsonString := gjson.Get(originalJsonString, `@sort:{"array":"feeds","orderBy":"id","desc":false}.@pretty:{"sortKeys":true}`).Raw
		if (formattedJsonString != originalJsonString) {
			log.Errorf("%s: Not formatted properly.", filename)
			// print out diff
			dmp := diffmatchpatch.New()
			diffs := dmp.DiffMain(originalJsonString, formattedJsonString, false)
			fmt.Println(dmp.DiffPrettyText(diffs))
		} else {
			log.Infof("%s: Formatted properly.", filename)
		}
	}
	return nil
}
