package main

import (
	"flag"
	"fmt"

	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/validator"
)

// validateCommand
type validateCommand struct {
	validateExtensions arrayFlags
}

func (cmd *validateCommand) Run(args []string) error {
	fl := flag.NewFlagSet("validate", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: validate <reader>")
		fl.PrintDefaults()
	}
	fl.Var(&cmd.validateExtensions, "ext", "Include GTFS Extension")
	err := fl.Parse(args)
	if err != nil || fl.NArg() < 1 {
		fl.Usage()
		log.Exit("Requires input reader")
	}
	//
	reader := MustGetReader(fl.Arg(0))
	defer reader.Close()
	v, err := validator.NewValidator(reader)
	if err != nil {
		return err
	}
	for _, extName := range cmd.validateExtensions {
		e, err := ext.GetExtension(extName)
		if err != nil {
			return fmt.Errorf("No extension for: %s", extName)
		}
		v.Copier.AddExtension(e)
	}
	v.Validate()
	return nil
}
