package main

import (
	"context"
	_ "embed"
	"os"
	_ "time/tzdata"

	"github.com/interline-io/log"
	tl "github.com/interline-io/transitland-lib"
	"github.com/interline-io/transitland-lib/cmds"
	"github.com/interline-io/transitland-lib/diff"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/tlxy"

	_ "github.com/interline-io/transitland-lib/ext/plus"
	_ "github.com/interline-io/transitland-lib/filters"
	_ "github.com/interline-io/transitland-lib/tlcsv"
	_ "github.com/interline-io/transitland-lib/tldb"
	_ "github.com/interline-io/transitland-lib/tldb/postgres"
	_ "github.com/interline-io/transitland-lib/tldb/sqlite"

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

func (cmd *versionCommand) Run(ctx context.Context) error {
	log.Print("transitland-lib version: %s", tl.Version.Tag)
	log.Print("transitland-lib commit: https://github.com/interline-io/transitland-lib/commit/%s (time: %s)", tl.Version.Commit, tl.Version.CommitTime)
	log.Print("GTFS specification version: https://github.com/google/transit/blob/%s/gtfs/spec/en/reference.md", tl.GTFSVERSION)
	log.Print("GTFS Realtime specification version: https://github.com/google/transit/blob/%s/gtfs-realtime/proto/gtfs-realtime.proto", tl.GTFSRTVERSION)
	return nil
}

var rootCmd = &cobra.Command{
	Use:   "transitland",
	Short: "transitland-lib utilities",
}

func init() {
	pc := "transitland"
	dmfrCommand := &cobra.Command{
		Use:    "dmfr",
		Short:  "DMFR subcommands",
		Long:   "DMFR Subcommands. Deprecated. Use dmfr-format, dmfr-lint, etc. instead.",
		Hidden: true,
	}
	dmfrCommand.AddCommand(
		tlcli.CobraHelper(&cmds.LintCommand{}, pc, "lint"),
		tlcli.CobraHelper(&cmds.FormatCommand{}, pc, "format"),
	)

	genDocCommand := tlcli.CobraHelper(&tlcli.GenDocCommand{Command: rootCmd}, pc, "gendoc")
	genDocCommand.Hidden = true

	rootCmd.AddCommand(
		tlcli.CobraHelper(&cmds.CopyCommand{}, pc, "copy"),
		tlcli.CobraHelper(&cmds.ExtractCommand{}, pc, "extract"),
		tlcli.CobraHelper(&cmds.FetchCommand{}, pc, "fetch"),
		tlcli.CobraHelper(&cmds.FormatCommand{}, pc, "dmfr-format"),
		tlcli.CobraHelper(&cmds.ImportCommand{}, pc, "import"),
		tlcli.CobraHelper(&cmds.ChecksumCommand{}, pc, "checksum"),
		tlcli.CobraHelper(&cmds.LintCommand{}, pc, "dmfr-lint"),
		tlcli.CobraHelper(&cmds.MergeCommand{}, pc, "merge"),
		tlcli.CobraHelper(&cmds.RebuildStatsCommand{}, pc, "rebuild-stats"),
		tlcli.CobraHelper(&cmds.SyncCommand{}, pc, "sync"),
		tlcli.CobraHelper(&cmds.UnimportCommand{}, pc, "unimport"),
		tlcli.CobraHelper(&cmds.DeleteCommand{}, pc, "delete"),
		tlcli.CobraHelper(&cmds.ValidatorCommand{}, pc, "validate"),
		tlcli.CobraHelper(&cmds.RTConvertCommand{}, pc, "rt-convert"),
		tlcli.CobraHelper(&diff.Command{}, pc, "diff"),
		tlcli.CobraHelper(&tlxy.PolylinesCommand{}, pc, "polylines-create"),
		tlcli.CobraHelper(&versionCommand{}, pc, "version"),
		tlcli.CobraHelper(&cmds.DBMigrateCommand{}, pc, "dbmigrate"),
		genDocCommand,
		dmfrCommand,
	)

}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
