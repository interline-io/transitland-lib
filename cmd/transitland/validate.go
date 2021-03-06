package main

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/internal/cli"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/internal/snakejson"
	"github.com/interline-io/transitland-lib/validator"
)

// validateCommand
type validateCommand struct {
	Options            validator.Options
	OutputFile         string
	validateExtensions cli.ArrayFlags
}

func (cmd *validateCommand) Run(args []string) error {
	fl := flag.NewFlagSet("validate", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: validate <reader>")
		fl.PrintDefaults()
	}
	fl.Var(&cmd.validateExtensions, "ext", "Include GTFS Extension")
	fl.StringVar(&cmd.OutputFile, "o", "", "Write validation report as JSON to file")
	fl.BoolVar(&cmd.Options.BestPractices, "best-practices", false, "Include Best Practices validations")
	err := fl.Parse(args)
	if err != nil || fl.NArg() < 1 {
		fl.Usage()
		log.Exit("Requires input reader")
	}
	reader := ext.MustGetReader(fl.Arg(0))
	defer reader.Close()
	v, err := validator.NewValidator(reader, cmd.Options)
	if err != nil {
		return err
	}
	// TODO
	// for _, extName := range cmd.validateExtensions {
	// 	e, err := ext.GetExtension(extName)
	// 	if err != nil {
	// 		return fmt.Errorf("No extension for: %s", extName)
	// 	}
	// 	v.Copier.AddExtension(e)
	// }
	result, _ := v.Validate()
	result.DisplayErrors()
	result.DisplayWarnings()
	result.DisplaySummary()

	// Write output
	if cmd.OutputFile != "" {
		f, err := os.Create(cmd.OutputFile)
		if err != nil {
			panic(err)
		}
		b, err := json.MarshalIndent(snakejson.SnakeMarshaller{Value: result}, "", "  ")
		if err != nil {
			panic(err)
		}
		f.Write(b)
		f.Close()
	}
	return nil
}
