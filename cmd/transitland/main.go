package main

import (
	"context"
	_ "embed"
	"log"
	"os"
	"runtime/debug"
	"strings"
	_ "time/tzdata"

	tl "github.com/interline-io/transitland-lib"
	"github.com/interline-io/transitland-lib/cmds"
	"github.com/interline-io/transitland-lib/diff"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/tlxy"

	_ "github.com/interline-io/transitland-lib/ext/filters"
	_ "github.com/interline-io/transitland-lib/ext/plus"
	_ "github.com/interline-io/transitland-lib/tlcsv"
	_ "github.com/interline-io/transitland-lib/tldb"
	_ "github.com/interline-io/transitland-lib/tldb/postgres"
	_ "github.com/interline-io/transitland-lib/tldb/sqlite"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var rootCmd = &cobra.Command{
	Use:               "transitland",
	Short:             "transitland-lib utilities",
	DisableAutoGenTag: true,
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
		tlcli.CobraHelper(&cmds.ServerCommand{}, pc, "server"),
		tlcli.CobraHelper(&versionCommand{}, pc, "version"),
		tlcli.CobraHelper(&cmds.DBMigrateCommand{}, pc, "dbmigrate"),
		tlcli.CobraHelper(&cmds.FeedStateManagerCommand{}, pc, "feed-state"),
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

////////////

// Read version from compiled in git details
var Version VersionInfo

type VersionInfo struct {
	Tag        string
	Commit     string
	CommitTime string
}

func getVersion() VersionInfo {
	ret := VersionInfo{}
	info, _ := debug.ReadBuildInfo()
	tagPrefix := "main.tag="
	for _, kv := range info.Settings {
		switch kv.Key {
		case "vcs.revision":
			ret.Commit = kv.Value
		case "vcs.time":
			ret.CommitTime = kv.Value
		case "-ldflags":
			for _, ss := range strings.Split(kv.Value, " ") {
				if strings.HasPrefix(ss, tagPrefix) {
					ret.Tag = strings.TrimPrefix(ss, tagPrefix)
				}
			}
		}
	}
	return ret
}

type versionCommand struct{}

func (cmd *versionCommand) AddFlags(fl *pflag.FlagSet) {}

func (cmd *versionCommand) HelpDesc() (string, string) {
	return "Program version and supported GTFS and GTFS-RT versions", ""
}

func (cmd *versionCommand) Parse(args []string) error {
	return nil
}

func (cmd *versionCommand) Run(context.Context) error {
	vi := getVersion()
	log.Printf("transitland-lib version: %s\n", vi.Tag)
	log.Printf("transitland-lib commit: https://github.com/interline-io/transitland-lib/commit/%s (time: %s)\n", vi.Commit, vi.CommitTime)
	log.Printf("GTFS specification version: https://github.com/google/transit/blob/%s/gtfs/spec/en/reference.md\n", tl.GTFSVERSION)
	log.Printf("GTFS Realtime specification version: https://github.com/google/transit/blob/%s/gtfs-realtime/proto/gtfs-realtime.proto\n", tl.GTFSRTVERSION)
	return nil
}
