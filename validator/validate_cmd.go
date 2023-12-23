package validator

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/internal/cli"
	"github.com/interline-io/transitland-lib/internal/snakejson"
	"github.com/interline-io/transitland-lib/tldb"
)

// Command
type Command struct {
	Options    Options
	rtFiles    cli.ArrayFlags
	OutputFile string
	DBURL      string
	FVID       int
	extensions cli.ArrayFlags
	readerPath string
}

func (cmd *Command) Parse(args []string) error {
	fl := flag.NewFlagSet("validate", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: validate <reader>")
		fl.PrintDefaults()
	}
	fl.Var(&cmd.extensions, "ext", "Include GTFS Extension")
	fl.StringVar(&cmd.OutputFile, "o", "", "Write validation report as JSON to file")
	fl.BoolVar(&cmd.Options.BestPractices, "best-practices", false, "Include Best Practices validations")
	fl.BoolVar(&cmd.Options.IncludeRealtimeJson, "rt-json", false, "Include GTFS-RT proto messages as JSON in validation report")
	fl.BoolVar(&cmd.Options.SaveStaticValidationReport, "save-static-report", false, "Save static validation report in database")
	fl.BoolVar(&cmd.Options.SaveRealtimeValidationReport, "save-rt-report", false, "Save RT validation report in database")
	fl.IntVar(&cmd.FVID, "save-fvid", 0, "Save report to feed version ID")
	fl.Var(&cmd.rtFiles, "rt", "Include GTFS-RT proto message in validation report")
	fl.IntVar(&cmd.Options.ErrorLimit, "error-limit", 1000, "Max number of detailed errors per error group")
	err := fl.Parse(args)
	if err != nil || fl.NArg() < 1 {
		fl.Usage()
		return errors.New("requires input reader")
	}
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	cmd.readerPath = fl.Arg(0)
	cmd.Options.ValidateRealtimeMessages = cmd.rtFiles
	cmd.Options.Extensions = cmd.extensions
	cmd.Options.EvaluateAt = time.Now()
	return nil
}

func (cmd *Command) Run() error {
	log.Infof("Validating: %s", cmd.readerPath)
	reader, err := ext.OpenReader(cmd.readerPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	v, err := NewValidator(reader, cmd.Options)
	if err != nil {
		return err
	}
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

	// Save to database
	if cmd.Options.SaveRealtimeValidationReport || cmd.Options.SaveStaticValidationReport {
		log.Infof("Saving validation report to feed version: %d", cmd.FVID)
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return err
		}
		atx := writer.Adapter
		defer atx.Close()
		if err := SaveValidationReport(atx, result, time.Now(), cmd.FVID, true, true); err != nil {
			return err
		}
	}
	return nil
}
