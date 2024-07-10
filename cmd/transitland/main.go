package main

import (
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/diff"
	"github.com/interline-io/transitland-lib/dmfr/fetch"
	"github.com/interline-io/transitland-lib/dmfr/format"
	"github.com/interline-io/transitland-lib/dmfr/importer"
	"github.com/interline-io/transitland-lib/dmfr/lint"
	"github.com/interline-io/transitland-lib/dmfr/sync"
	"github.com/interline-io/transitland-lib/dmfr/unimporter"
	"github.com/interline-io/transitland-lib/extract"
	"github.com/interline-io/transitland-lib/internal/cli"
	"github.com/interline-io/transitland-lib/merge"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/validator"

	_ "github.com/interline-io/transitland-lib/ext/plus"
	_ "github.com/interline-io/transitland-lib/filters"
	_ "github.com/interline-io/transitland-lib/tlcsv"
	_ "github.com/interline-io/transitland-lib/tldb"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type versionCommand struct{}

func (cmd *versionCommand) AddFlags(fl *pflag.FlagSet) {}

func (cmd *versionCommand) Parse(args []string) error {
	return nil
}

func (cmd *versionCommand) Run() error {
	log.Print("transitland-lib version: %s", tl.VERSION)
	log.Print("GTFS specification version: https://github.com/google/transit/blob/%s/gtfs/spec/en/reference.md", tl.GTFSVERSION)
	log.Print("GTFS Realtime specification version: https://github.com/google/transit/blob/%s/gtfs-realtime/proto/gtfs-realtime.proto", tl.GTFSRTVERSION)
	return nil
}

var rootCmd = &cobra.Command{Use: "transitland"}

func init() {
	dmfrCommand := &cobra.Command{Use: "dmfr"}
	dmfrCommand.AddCommand(
		cli.CobraHelper(&lint.Command{}, "format"),
		cli.CobraHelper(&format.Command{}, "lint"),
	)

	rootCmd.AddCommand(
		cli.CobraHelper(&fetch.Command{}, "fetch"),
		cli.CobraHelper(&sync.Command{}, "sync"),
		cli.CobraHelper(&copier.Command{}, "copy"),
		cli.CobraHelper(&validator.Command{}, "validate"),
		cli.CobraHelper(&extract.Command{}, "extract"),
		cli.CobraHelper(&diff.Command{}, "diff"),
		cli.CobraHelper(&fetch.RebuildStatsCommand{}, "rebuild-stats"),
		cli.CobraHelper(&importer.Command{}, "import"),
		cli.CobraHelper(&unimporter.Command{}, "unimport"),
		cli.CobraHelper(&merge.Command{}, "merge"),
		cli.CobraHelper(&versionCommand{}, "version"),
		cli.CobraHelper(&lint.Command{}, "dmfr-format"),
		cli.CobraHelper(&format.Command{}, "dmfr-lint"),
		dmfrCommand,
	)
}

func main() {
	rootCmd.Execute()
}
