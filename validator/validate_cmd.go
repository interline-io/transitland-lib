package validator

import (
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/internal/cli"
	"github.com/interline-io/transitland-lib/internal/snakejson"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/spf13/pflag"
)

// Command
type Command struct {
	Options                 Options
	rtFiles                 []string
	OutputFile              string
	DBURL                   string
	FVID                    int
	extensions              []string
	SaveValidationReport    bool
	ValidationReportStorage string
	readerPath              string
}

func (cmd *Command) AddFlags(fl *pflag.FlagSet) {
	fl.StringSliceVar(&cmd.extensions, "ext", nil, "Include GTFS Extension")
	fl.StringVar(&cmd.OutputFile, "o", "", "Write validation report as JSON to file")
	fl.BoolVar(&cmd.Options.BestPractices, "best-practices", false, "Include Best Practices validations")
	fl.BoolVar(&cmd.Options.IncludeRealtimeJson, "rt-json", false, "Include GTFS-RT proto messages as JSON in validation report")
	fl.BoolVar(&cmd.SaveValidationReport, "validation-report", false, "Save static validation report in database")
	fl.StringVar(&cmd.ValidationReportStorage, "validation-report-storage", "", "Storage path for saving validation report JSON")
	fl.IntVar(&cmd.FVID, "save-fvid", 0, "Save report to feed version ID")
	fl.StringSliceVar(&cmd.rtFiles, "rt", nil, "Include GTFS-RT proto message in validation report")
	fl.IntVar(&cmd.Options.ErrorLimit, "error-limit", 1000, "Max number of detailed errors per error group")
}

func (cmd *Command) Parse(args []string) error {
	fl := cli.NewNArgs(args)
	if fl.NArg() < 1 {
		return errors.New("requires input reader")
	}
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	cmd.readerPath = fl.Arg(0)
	cmd.Options.ValidateRealtimeMessages = cmd.rtFiles
	cmd.Options.Extensions = cmd.extensions
	cmd.Options.EvaluateAt = time.Now().In(time.UTC)
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
	// result.DisplayErrors()
	// result.DisplayWarnings()
	// result.DisplaySummary()

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
	if cmd.SaveValidationReport {
		log.Infof("Saving validation report to feed version: %d", cmd.FVID)
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return err
		}
		atx := writer.Adapter
		defer atx.Close()
		if err := SaveValidationReport(atx, result, cmd.FVID, cmd.ValidationReportStorage); err != nil {
			return err
		}
	}
	return nil
}
