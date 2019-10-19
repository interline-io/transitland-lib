package main

import (
	"flag"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/validator"
)

// validateCommand
type validateCommand struct {
	validateExtensions arrayFlags
	args               []string
}

func (cmd *validateCommand) run(args []string) {
	fl := flag.NewFlagSet("validate", flag.ExitOnError)
	fl.Var(&cmd.validateExtensions, "ext", "Include GTFS Extension")
	fl.Parse(args)
	cmd.args = fl.Args()
	//
	reader := MustGetReader(cmd.args[0])
	defer reader.Close()
	v, err := validator.NewValidator(reader)
	if err != nil {
		panic(err)
	}
	for _, ext := range cmd.validateExtensions {
		e, err := gotransit.GetExtension(ext)
		if err != nil {
			exit("No extension for: %s", ext)
		}
		v.Copier.AddExtension(e)
	}
	v.Validate()
}
