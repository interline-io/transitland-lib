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
	Options    validator.Options
	rtFiles    cli.ArrayFlags
	OutputFile string
	extensions cli.ArrayFlags
}

func (cmd *validateCommand) Run(args []string) error {
	fl := flag.NewFlagSet("validate", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: validate <reader>")
		fl.PrintDefaults()
	}
	fl.Var(&cmd.extensions, "ext", "Include GTFS Extension")
	fl.StringVar(&cmd.OutputFile, "o", "", "Write validation report as JSON to file")
	fl.BoolVar(&cmd.Options.BestPractices, "best-practices", false, "Include Best Practices validations")
	fl.Var(&cmd.rtFiles, "rt", "Include GTFS-RT proto message in validation report")
	err := fl.Parse(args)
	if err != nil || fl.NArg() < 1 {
		fl.Usage()
		log.Exit("Requires input reader")
	}
	reader := ext.MustGetReader(fl.Arg(0))
	defer reader.Close()
	cmd.Options.ValidateRealtimeMessages = cmd.rtFiles
	cmd.Options.Extensions = cmd.extensions
	v, err := validator.NewValidator(reader, cmd.Options)
	if err != nil {
		return err
	}
	log.Info("Validating: %s", fl.Arg(0))
	result, err := v.Validate()
	if err != nil {
		return err
	}
	result.DisplayErrors()
	result.DisplayWarnings()
	result.DisplaySummary()

	// Write output
	if cmd.OutputFile != "" {
		f, err := os.Create(cmd.OutputFile)
		if err != nil {
			return err
		}
		b, err := json.MarshalIndent(snakejson.SnakeMarshaller{Value: result}, "", "  ")
		if err != nil {
			return err
		}
		f.Write(b)
		f.Close()
	}
	return nil
}
