package cmds

import (
	"context"
	"errors"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/spf13/pflag"
)

// CopyCommand
type CopyCommand struct {
	copier.Options
	fvid                    int
	create                  bool
	extensionDefs           []string
	readerPath              string
	writerPath              string
	writeExtraColumns       bool
	standardizedSort        string
	standardizedSortColumns []string
}

func (cmd *CopyCommand) HelpDesc() (string, string) {
	a := "Copy performs a basic copy from a reader to a writer."
	b := `By default, any entity with errors will be skipped and not written to output.
This can be ignored with --allow-entity-errors to ignore simple errors and
--allow-reference-errors to ignore entity relationship errors, such as a
reference to a non-existent stop.

By default, the output order is determined by transitland-lib's streaming
architecture. It generally preserves the input order, although some records
may be reordered to maintain associations (such as ensuring parent stops are
processed before child stops).

Output can be automatically sorted using --standardized-sort (asc or desc).
This is an optional feature and is off by default. When enabled, it applies
an opinionated, standardized sort order to CSV files, which is useful for
consistent diffing and human readability. By default, it uses logical primary
GTFS columns (e.g., stop_id for stops.txt), but specific columns can be
provided with --standardized-sort-columns.`
	return a, b
}

func (cmd *CopyCommand) HelpExample() string {
	return `
% {{.ParentCommand}} {{.Command}} --allow-entity-errors "https://www.bart.gov/dev/schedules/google_transit.zip" output.zip

% unzip -p output.zip agency.txt
agency_id,agency_name,agency_url,agency_timezone,agency_lang,agency_phone,agency_fare_url,agency_email
BART,Bay Area Rapid Transit,https://www.bart.gov/,America/Los_Angeles,,510-464-6000,,
`
}

func (cmd *CopyCommand) HelpArgs() string {
	return "[flags] <reader> <writer>"
}

func (cmd *CopyCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringSliceVar(&cmd.extensionDefs, "ext", nil, "Include GTFS Extension")
	fl.IntVar(&cmd.fvid, "fvid", 0, "Specify FeedVersionID when writing to a database")
	fl.BoolVar(&cmd.create, "create", false, "Create a basic database schema if none exists")
	fl.BoolVar(&cmd.CopyExtraFiles, "write-extra-files", false, "Copy additional files found in source to destination")
	fl.BoolVar(&cmd.writeExtraColumns, "write-extra-columns", false, "Include extra columns in output")
	fl.BoolVar(&cmd.AllowEntityErrors, "allow-entity-errors", false, "Allow entities with errors to be copied")
	fl.BoolVar(&cmd.AllowReferenceErrors, "allow-reference-errors", false, "Allow entities with reference errors to be copied")
	fl.IntVar(&cmd.Options.ErrorLimit, "error-limit", 1000, "Max number of detailed errors per error group")
	fl.StringVar(&cmd.standardizedSort, "standardized-sort", "", "Standardized sort order for CSV files (asc, desc, or none)")
	fl.StringSliceVar(&cmd.standardizedSortColumns, "standardized-sort-columns", nil, "Comma-separated list of columns to sort by (optional; if empty, defaults are used)")
}

func (cmd *CopyCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	if fl.NArg() < 2 {
		return errors.New("requires input reader and output writer")
	}
	cmd.readerPath = fl.Arg(0)
	cmd.writerPath = fl.Arg(1)
	return nil
}

func (cmd *CopyCommand) Run(ctx context.Context) error {
	// Reader / Writer
	reader, err := ext.OpenReader(cmd.readerPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	writer, err := ext.OpenWriter(cmd.writerPath, cmd.create)
	if err != nil {
		return err
	}
	if cmd.writeExtraColumns {
		if v, ok := writer.(adapters.WriterWithExtraColumns); ok {
			v.WriteExtraColumns(true)
		} else {
			return errors.New("writer does not support extra output columns")
		}
	}
	if cmd.standardizedSort != "" {
		if v, ok := writer.(adapters.WriterWithStandardizedSort); ok {
			v.SetStandardizedSortOptions(adapters.StandardizedSortOptions{
				StandardizedSort:        cmd.standardizedSort,
				StandardizedSortColumns: cmd.standardizedSortColumns,
			})
		} else {
			return errors.New("writer does not support standardized sort")
		}
	}

	defer writer.Close()

	// Setup copier
	cmd.Options.ExtensionDefs = cmd.extensionDefs
	_, err = copier.CopyWithOptions(ctx, reader, writer, cmd.Options)
	return err
}
