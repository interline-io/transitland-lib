package main

import (
	"flag"
	"fmt"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/validator"
)

// validateCommand
type validateCommand struct {
	validateExtensions arrayFlags
}

func (cmd *validateCommand) Run(args []string) error {
	fl := flag.NewFlagSet("validate", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: validate <input>")
		fl.PrintDefaults()
	}
	fl.Var(&cmd.validateExtensions, "ext", "Include GTFS Extension")
	fl.Parse(args)
	if fl.NArg() < 1 {
		fl.Usage()
		exit("requires input reader")
	}
	//
	reader := MustGetReader(fl.Arg(0))
	defer reader.Close()
	v, err := validator.NewValidator(reader)
	if err != nil {
		return err
	}
	for _, ext := range cmd.validateExtensions {
		e, err := gotransit.GetExtension(ext)
		if err != nil {
			return fmt.Errorf("No extension for: %s", ext)
		}
		v.Copier.AddExtension(e)
	}
	v.Validate()
	return nil
}
