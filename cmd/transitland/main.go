package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"syscall"
	_ "time/tzdata"

	"github.com/interline-io/log"
	tl "github.com/interline-io/transitland-lib"
	"github.com/interline-io/transitland-lib/cmds"
	"github.com/interline-io/transitland-lib/diff"
	neSchema "github.com/interline-io/transitland-lib/schema/ne"
	postgresSchema "github.com/interline-io/transitland-lib/schema/postgres"
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

	genDocCommand := tlcli.CobraHelper(&tlcli.GenDocCommand{Command: rootCmd}, pc, "gendoc")
	genDocCommand.Hidden = true

	// Hidden aliases for backwards compatibility
	dmfrFormatCommand := tlcli.CobraHelper(&cmds.DmfrFormatCommand{}, pc, "dmfr-format")
	dmfrFormatCommand.Hidden = true
	dmfrLintCommand := tlcli.CobraHelper(&cmds.DmfrLintCommand{}, pc, "dmfr-lint")
	dmfrLintCommand.Hidden = true

	rootCmd.AddCommand(
		tlcli.CobraHelper(&cmds.CopyCommand{}, pc, "copy"),
		tlcli.CobraHelper(&cmds.ExtractCommand{}, pc, "extract"),
		tlcli.CobraHelper(&cmds.FetchCommand{}, pc, "fetch"),
		tlcli.CobraHelper(&cmds.ImportCommand{}, pc, "import"),
		tlcli.CobraHelper(&cmds.ChecksumCommand{}, pc, "checksum"),
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
		tlcli.CobraHelper(&postgresSchema.Command{}, pc, "dbmigrate"),
		tlcli.CobraHelper(&neSchema.Command{}, pc, "dbmigrate-natural-earth"),
		tlcli.CobraHelper(&cmds.FeedStateManagerCommand{}, pc, "feed-state"),
		cmds.NewDmfrCommand(pc),
		dmfrFormatCommand,
		dmfrLintCommand,
		genDocCommand,
	)

	// Persistent profiling flags, available on every subcommand. CPU profiling
	// starts before the command runs; the heap profile is written at exit.
	rootCmd.PersistentFlags().StringVar(&cpuProfilePath, "cpuprofile", "", "Write a CPU profile to `file`")
	rootCmd.PersistentFlags().StringVar(&memProfilePath, "memprofile", "", "Write a heap profile to `file` at exit")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Open both profile files up front so a bad/missing path fails in the
		// first second, instead of being discovered only at exit after a long
		// run has already completed.
		if memProfilePath != "" {
			f, err := createProfileFile(memProfilePath)
			if err != nil {
				return fmt.Errorf("could not create memory profile %q: %w", memProfilePath, err)
			}
			memProfileFile = f
			// SIGUSR1 dumps an on-demand heap profile to <memprofile>.sigN so the
			// peak resident set can be captured mid-run; the at-exit profile only
			// sees the post-free heap. Send with: kill -USR1 <pid>
			installHeapSignalDump(memProfilePath)
		}
		if cpuProfilePath != "" {
			f, err := createProfileFile(cpuProfilePath)
			if err != nil {
				return fmt.Errorf("could not create CPU profile %q: %w", cpuProfilePath, err)
			}
			if err := pprof.StartCPUProfile(f); err != nil {
				f.Close()
				return fmt.Errorf("could not start CPU profile: %w", err)
			}
			cpuProfileFile = f
		}
		return nil
	}
}

var (
	cpuProfilePath string
	memProfilePath string
	cpuProfileFile *os.File
	memProfileFile *os.File
)

// installHeapSignalDump writes a heap profile to <base>.sigN each time the
// process receives SIGUSR1, so the peak resident set can be captured mid-run
// (the at-exit --memprofile only sees the post-free heap). Runs for the life of
// the process.
func installHeapSignalDump(base string) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGUSR1)
	go func() {
		n := 0
		for range ch {
			n++
			path := fmt.Sprintf("%s.sig%d", base, n)
			f, err := createProfileFile(path)
			if err != nil {
				log.Error().Err(err).Msgf("could not create heap profile %q", path)
				continue
			}
			runtime.GC() // update in-use statistics before snapshotting the heap
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Error().Err(err).Msg("could not write heap profile")
			}
			f.Close()
			log.Info().Msgf("wrote heap profile to %s", path)
		}
	}()
}

// createProfileFile opens a profile output file, creating parent directories as
// needed so a missing dir never silently costs a long profiling run.
func createProfileFile(path string) (*os.File, error) {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	return os.Create(path)
}

// writeProfiles stops CPU profiling and snapshots the heap into the files opened
// in PersistentPreRunE. Called on both the success and error paths since os.Exit
// skips deferred funcs; a SIGKILL (OOM) still bypasses it, so pair --memprofile
// with GODEBUG=gctrace=1 to catch peaks.
func writeProfiles() {
	if cpuProfileFile != nil {
		pprof.StopCPUProfile()
		cpuProfileFile.Close()
	}
	if memProfileFile != nil {
		runtime.GC() // update in-use statistics before snapshotting the heap
		if err := pprof.WriteHeapProfile(memProfileFile); err != nil {
			log.Error().Err(err).Msg("could not write memory profile")
		}
		memProfileFile.Close()
	}
}

func main() {
	err := rootCmd.Execute()
	writeProfiles()
	if err != nil {
		os.Exit(1)
	}
}

var tag string

func init() {
	if tag != "" {
		tl.Version.Tag = tag
	}
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
	vi := tl.Version
	log.Print("transitland-lib version: %s", vi.Tag)
	log.Print("transitland-lib commit: https://github.com/interline-io/transitland-lib/commit/%s (time: %s)", vi.Commit, vi.CommitTime)
	log.Print("GTFS specification version: https://github.com/google/transit/blob/%s/gtfs/spec/en/reference.md", tl.GTFSVERSION)
	log.Print("GTFS Realtime specification version: https://github.com/google/transit/blob/%s/gtfs-realtime/proto/gtfs-realtime.proto", tl.GTFSRTVERSION)
	return nil
}
