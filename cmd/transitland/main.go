package main

import (
	_ "embed"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/cmd/tlcli"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/diff"
	"github.com/interline-io/transitland-lib/dmfr/fetch"
	"github.com/interline-io/transitland-lib/dmfr/format"
	"github.com/interline-io/transitland-lib/dmfr/importer"
	"github.com/interline-io/transitland-lib/dmfr/lint"
	"github.com/interline-io/transitland-lib/dmfr/sync"
	"github.com/interline-io/transitland-lib/dmfr/unimporter"
	"github.com/interline-io/transitland-lib/extract"
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

func (cmd *versionCommand) HelpDesc() (string, string) {
	return "Program version and supported GTFS and GTFS-RT versions", ""
}

func (cmd *versionCommand) Parse(args []string) error {
	return nil
}

func (cmd *versionCommand) Run() error {
	log.Print("transitland-lib version: %s", tl.Version.Tag)
	log.Print("transitland-lib commit: https://github.com/interline-io/transitland-lib/commit/%s (time: %s)", tl.Version.Commit, tl.Version.CommitTime)
	log.Print("GTFS specification version: https://github.com/google/transit/blob/%s/gtfs/spec/en/reference.md", tl.GTFSVERSION)
	log.Print("GTFS Realtime specification version: https://github.com/google/transit/blob/%s/gtfs-realtime/proto/gtfs-realtime.proto", tl.GTFSRTVERSION)
	return nil
}

var rootCmd = &cobra.Command{Use: "transitland"}

func init() {
	dmfrCommand := &cobra.Command{Use: "dmfr", Short: "DMFR subcommands", Long: "DMFR Subcommands. Deprecated. Use dmfr-format, dmfr-lint, etc. instead."}
	dmfrCommand.AddCommand(
		tlcli.CobraHelper(&lint.Command{}, "format"),
		tlcli.CobraHelper(&format.Command{}, "lint"),
	)

	rootCmd.AddCommand(
		tlcli.CobraHelper(&fetch.Command{}, "fetch"),
		tlcli.CobraHelper(&sync.Command{}, "sync"),
		tlcli.CobraHelper(&copier.Command{}, "copy"),
		tlcli.CobraHelper(&validator.Command{}, "validate"),
		tlcli.CobraHelper(&extract.Command{}, "extract"),
		tlcli.CobraHelper(&diff.Command{}, "diff"),
		tlcli.CobraHelper(&fetch.RebuildStatsCommand{}, "rebuild-stats"),
		tlcli.CobraHelper(&importer.Command{}, "import"),
		tlcli.CobraHelper(&unimporter.Command{}, "unimport"),
		tlcli.CobraHelper(&merge.Command{}, "merge"),
		tlcli.CobraHelper(&versionCommand{}, "version"),
		tlcli.CobraHelper(&lint.Command{}, "dmfr-format"),
		tlcli.CobraHelper(&format.Command{}, "dmfr-lint"),
		dmfrCommand,
	)
}

func main() {
	rootCmd.Execute()
}
