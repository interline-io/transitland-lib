package cmds

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/spf13/pflag"
)

// DmfrFromDirCommand generates a DMFR file from a directory of GTFS files.
type DmfrFromDirCommand struct {
	Dir       string
	Prefix    string
	Recursive bool
}

func (cmd *DmfrFromDirCommand) HelpDesc() (string, string) {
	return "Generate DMFR from a directory of GTFS files", "Scans a directory for .zip files and outputs a DMFR registry JSON to stdout."
}

func (cmd *DmfrFromDirCommand) HelpArgs() string {
	return "[flags] <directory>"
}

func (cmd *DmfrFromDirCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringVar(&cmd.Prefix, "prefix", "", "Prefix for generated feed IDs")
	fl.BoolVar(&cmd.Recursive, "recursive", false, "Search subdirectories recursively")
}

func (cmd *DmfrFromDirCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	if fl.NArg() == 0 {
		return errors.New("directory argument required")
	}
	cmd.Dir = fl.Arg(0)
	return nil
}

func (cmd *DmfrFromDirCommand) Run(ctx context.Context) error {
	// Get absolute path for the directory
	absDir, err := filepath.Abs(cmd.Dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Find .zip files
	var matches []string
	if cmd.Recursive {
		err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, ".zip") {
				matches = append(matches, path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}
	} else {
		pattern := filepath.Join(absDir, "*.zip")
		matches, err = filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("failed to glob directory: %w", err)
		}
	}

	// Build feeds
	var feeds []dmfr.Feed
	for _, match := range matches {
		base := filepath.Base(match)
		feedID := strings.TrimSuffix(base, ".zip")
		if cmd.Prefix != "" {
			feedID = cmd.Prefix + feedID
		}
		feed := dmfr.Feed{
			FeedID: feedID,
			Spec:   "gtfs",
			URLs: dmfr.FeedUrls{
				StaticCurrent: "file://" + match,
			},
		}
		feeds = append(feeds, feed)
	}

	// Build registry and output
	reg := dmfr.Registry{
		Feeds: feeds,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(reg); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}
