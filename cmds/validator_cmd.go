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
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
)

// ValidatorCommand
type ValidatorCommand struct {
	Options                 validator.Options
	rtFiles                 []string
	OutputFile              string
	Quiet                   bool
	DBURL                   string
	FVID                    int
	extensionDefs           []string
	SaveValidationReport    bool
	ValidationReportStorage string
	readerPath              string
	errorThresholds         []string
}

func (cmd *ValidatorCommand) HelpDesc() (string, string) {
	return "Validate a GTFS feed", "The validate command performs a basic validation on a data source and writes the results to standard out."
}

func (cmd *ValidatorCommand) HelpExample() string {
	return `% {{.ParentCommand}} {{.Command}} "https://www.bart.gov/dev/schedules/google_transit.zip"
% {{.ParentCommand}} {{.Command}} -o - --include-entities "http://developer.trimet.org/schedule/gtfs.zip"`
}

func (cmd *ValidatorCommand) HelpArgs() string {
	return "[flags] <reader>"
}

// shouldShowLogs returns true if logs should be displayed
func (cmd *ValidatorCommand) shouldShowLogs() bool {
	return !cmd.Quiet
}

func (cmd *ValidatorCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringSliceVar(&cmd.extensionDefs, "ext", nil, "Include GTFS Extension")
	fl.StringVarP(&cmd.OutputFile, "out", "o", "", "Write validation report as JSON to file; use '-' for stdout (implies -q)")
	fl.BoolVarP(&cmd.Quiet, "quiet", "q", false, "Suppress log output")
	fl.BoolVar(&cmd.Options.BestPractices, "best-practices", false, "Include Best Practices validations")
	fl.BoolVar(&cmd.Options.IncludeEntities, "include-entities", false, "Include GTFS entities in JSON output")
	fl.BoolVar(&cmd.Options.IncludeRouteGeometries, "include-route-geometries", false, "Include route geometries in JSON output")
	fl.BoolVar(&cmd.Options.IncludeServiceLevels, "include-service-levels", false, "Include service levels in JSON output")
	fl.BoolVar(&cmd.Options.IncludeRealtimeJson, "rt-json", false, "Include GTFS-RT proto messages as JSON in validation report")
	fl.BoolVar(&cmd.SaveValidationReport, "validation-report", false, "Save static validation report in database")
	fl.StringVar(&cmd.ValidationReportStorage, "validation-report-storage", "", "Storage path for saving validation report JSON")
	fl.IntVar(&cmd.FVID, "save-fvid", 0, "Save report to feed version ID")
	fl.StringSliceVar(&cmd.rtFiles, "rt", nil, "Include GTFS-RT proto message in validation report")
	fl.IntVar(&cmd.Options.ErrorLimit, "error-limit", 1000, "Max number of detailed errors per error group")
	fl.StringSliceVar(&cmd.errorThresholds, "error-threshold", nil, "Fail validation if file exceeds error percentage; format: 'filename:percent' or '*:percent' for default (e.g., 'stops.txt:5' or '*:10')")
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
	if len(cmd.errorThresholds) > 0 {
		thresholds, err := parseErrorThresholds(cmd.errorThresholds)
		if err != nil {
			return err
		}
		cmd.Options.ErrorThreshold = thresholds
	}

	// Output to stdout implies quiet mode
	if cmd.OutputFile == "-" {
		cmd.Quiet = true
	}

	// Suppress logs when quiet mode is enabled
	// TODO: Remove direct zerolog import once log package exports level constants
	if cmd.Quiet {
		log.SetLevel(zerolog.FatalLevel)
	}

	return nil
}

func (cmd *ValidatorCommand) Run(ctx context.Context) error {
	// Only log if not outputting JSON to stdout
	if cmd.shouldShowLogs() {
		log.For(ctx).Info().Msgf("Validating: %s", cmd.readerPath)
	}
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

	// Write JSON output if -o flag specified
	if cmd.OutputFile != "" {
		b, err := json.MarshalIndent(snakejson.SnakeMarshaller{Value: result}, "", "  ")
		if err != nil {
			return err
		}

		outf := os.Stdout
		if cmd.OutputFile != "-" {
			var err error
			outf, err = os.Create(cmd.OutputFile)
			if err != nil {
				return err
			}
			defer outf.Close()
		}
		if _, err := outf.Write(b); err != nil {
			return err
		}
		if _, err := outf.Write([]byte("\n")); err != nil {
			return err
		}
	}

	// Save to database
	if cmd.SaveValidationReport {
		if cmd.shouldShowLogs() {
			log.For(ctx).Info().Msgf("Saving validation report to feed version: %d", cmd.FVID)
		}
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
