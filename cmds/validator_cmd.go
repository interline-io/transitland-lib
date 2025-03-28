package cmds

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/internal/snakejson"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/validator"
	"github.com/spf13/pflag"
)

// ValidatorCommand
type ValidatorCommand struct {
	Options                 validator.Options
	rtFiles                 []string
	OutputFile              string
	DBURL                   string
	FVID                    int
	extensionDefs           []string
	SaveValidationReport    bool
	ValidationReportStorage string
	readerPath              string
}

func (cmd *ValidatorCommand) HelpDesc() (string, string) {
	return "Validate a GTFS feed", "The validate command performs a basic validation on a data source and writes the results to standard out."
}

func (cmd *ValidatorCommand) HelpExample() string {
	return `% {{.ParentCommand}} {{.Command}} "https://www.bart.gov/dev/schedules/google_transit.zip"`
}

func (cmd *ValidatorCommand) HelpArgs() string {
	return "[flags] <reader>"
}

func (cmd *ValidatorCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringSliceVar(&cmd.extensionDefs, "ext", nil, "Include GTFS Extension")
	fl.StringVar(&cmd.OutputFile, "o", "", "Write validation report as JSON to file")
	fl.BoolVar(&cmd.Options.BestPractices, "best-practices", false, "Include Best Practices validations")
	fl.BoolVar(&cmd.Options.IncludeRealtimeJson, "rt-json", false, "Include GTFS-RT proto messages as JSON in validation report")
	fl.BoolVar(&cmd.SaveValidationReport, "validation-report", false, "Save static validation report in database")
	fl.StringVar(&cmd.ValidationReportStorage, "validation-report-storage", "", "Storage path for saving validation report JSON")
	fl.IntVar(&cmd.FVID, "save-fvid", 0, "Save report to feed version ID")
	fl.StringSliceVar(&cmd.rtFiles, "rt", nil, "Include GTFS-RT proto message in validation report")
	fl.IntVar(&cmd.Options.ErrorLimit, "error-limit", 1000, "Max number of detailed errors per error group")
}

func (cmd *ValidatorCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	if fl.NArg() < 1 {
		return errors.New("requires input reader")
	}
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	cmd.readerPath = fl.Arg(0)
	cmd.Options.ValidateRealtimeMessages = cmd.rtFiles
	cmd.Options.ExtensionDefs = cmd.extensionDefs
	cmd.Options.EvaluateAt = time.Now().In(time.UTC)
	return nil
}

func (cmd *ValidatorCommand) Run(ctx context.Context) error {
	log.For(ctx).Info().Msgf("Validating: %s", cmd.readerPath)
	reader, err := ext.OpenReader(cmd.readerPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	v, err := validator.NewValidator(reader, cmd.Options)
	if err != nil {
		return err
	}
	result, err := v.Validate(ctx)
	if err != nil {
		return err
	}

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
		log.For(ctx).Info().Msgf("Saving validation report to feed version: %d", cmd.FVID)
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return err
		}
		atx := writer.Adapter
		defer atx.Close()
		if err := validator.SaveValidationReport(ctx, atx, result, cmd.FVID, cmd.ValidationReportStorage); err != nil {
			return err
		}
	}
	return nil
}
