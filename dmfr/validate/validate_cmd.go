package validate

import (
	"errors"
	"flag"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/log"
)

// Command validates a DMFR file.
type Command struct {
	Filenames []string
}

// Parse command line options.
func (cmd *Command) Parse(args []string) error {
	fl := flag.NewFlagSet("validate", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: validate <filenames...>")
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
	errs := []error{}
	for _, filename := range cmd.Filenames {
		log.Info("Loading DMFR: %s", filename)
		registry, err := dmfr.LoadAndParseRegistry(filename)
		if err != nil {
			errs = append(errs, err)
			log.Error("%s: Error when loading DMFR: %s", filename, err.Error())
		} else {
			log.Info("%s: Success loading DMFR with %d feeds", filename, len(registry.Feeds))
		}
	}
	if len(errs) > 0 {
		return errors.New("")
	}
	return nil
}
