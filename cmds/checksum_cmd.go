package cmds

import (
	"context"
	"errors"
	"fmt"

	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/spf13/pflag"
)

// ChecksumCommand calculates the SHA1 checksum of a static GTFS feed.
// The SHA1 checksum uniquely identifies a feed version and is used by Transitland
// to detect when new feed versions are available. This command calculates the
// SHA1 from the contents of the feed archive, not the zip archive as a whole
type ChecksumCommand struct {
	FeedPath string
	Raw      bool
}

func (cmd *ChecksumCommand) HelpDesc() (string, string) {
	return "Calculate the SHA1 checksum of a static GTFS feed's contents", `Calculate the SHA1 checksum of a GTFS feed archive and provide a link to look for a matching feed version in Transitland's online archive.

This checksum uniquely identifies the feed version and is used by Transitland to detect when new feed versions are available. The SHA1 is calculated from the contents of the feed archive, not the zip archive as a whole, so it won't change if only the zip packaging changes.

Example:
  transitland checksum myfeed.zip
  transitland checksum --raw http://example.com/myfeed.zip  # Output only the SHA1 hash

This command is useful for verifying feed integrity and looking up feed versions on Transitland. Use --raw for scripting scenarios where only the hash is needed.`
}

func (cmd *ChecksumCommand) HelpArgs() string {
	return "<feed-path>"
}

func (cmd *ChecksumCommand) AddFlags(fl *pflag.FlagSet) {
	fl.BoolVar(&cmd.Raw, "raw", false, "Output only the SHA1 hash (useful for scripting)")
}

// Parse command line options.
func (cmd *ChecksumCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	cmd.FeedPath = fl.Arg(0)
	if cmd.FeedPath == "" {
		return errors.New("must specify feed path")
	}
	return nil
}

// Run this command.
func (cmd *ChecksumCommand) Run(ctx context.Context) error {
	// Create reader for the feed
	reader, err := tlcsv.NewReader(cmd.FeedPath)
	if err != nil {
		return fmt.Errorf("could not create reader for %s: %w", cmd.FeedPath, err)
	}

	// Open the reader
	if err := reader.Open(); err != nil {
		return fmt.Errorf("could not open feed at %s: %w", cmd.FeedPath, err)
	}
	defer reader.Close()

	// Get feed version info including SHA1
	fv, err := stats.NewFeedVersionFromReader(reader)
	if err != nil {
		return fmt.Errorf("could not analyze feed at %s: %w", cmd.FeedPath, err)
	}

	// Print the SHA1 hash
	if fv.SHA1 == "" {
		return fmt.Errorf("could not calculate SHA1 checksum for GTFS feed at %s", cmd.FeedPath)
	}

	if cmd.Raw {
		fmt.Printf("%s\n", fv.SHA1)
	} else {
		fmt.Printf("SHA1 checksum of feed contents: %s\n", fv.SHA1)
		fmt.Printf("Find via Transitland: https://www.transit.land/feed-versions/%s\n", fv.SHA1)
	}

	return nil
}
