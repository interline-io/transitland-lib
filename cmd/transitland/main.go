package main

import (
	_ "embed"
	"os"
	_ "time/tzdata"

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
)

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
		tlcli.CobraHelper(&ServerCommand{}, pc, "server"),
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
