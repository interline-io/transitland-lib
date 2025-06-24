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
// SHA1 using the DirSHA1 method, which processes the contents of the feed archive,
// not the zip archive as a whole
type ChecksumCommand struct {
	FeedPath   string
	RawDirSHA1 bool
	RawZipSHA1 bool
}

func (cmd *ChecksumCommand) HelpDesc() (string, string) {
	return "Calculate the SHA1 checksum of a static GTFS feed", `Calculate the SHA1 checksum of a GTFS feed archive and provide a link to look for a matching feed version in Transitland's online archive.

This checksum uniquely identifies the feed version and is used by Transitland to detect when new feed versions are available. By default, this command shows both the zip SHA1 (archive file) and the directory SHA1 (feed contents). 

The directory SHA1 is calculated by:
- Finding all lowercase .txt files in the main directory (agency.txt, stops.txt, routes.txt, etc.)
- Sorting them alphabetically by filename
- Concatenating their contents in sorted order
- Calculating the SHA1 hash of the concatenated data

This approach ensures the hash represents the actual transit data, not the packaging, so it won't change if only the zip compression, file ordering, or metadata changes.

Example:
  transitland checksum myfeed.zip
  transitland checksum --raw-dir-sha1 http://example.com/myfeed.zip  # Output only the directory SHA1 hash, which is used in Transitland APIs
  transitland checksum --raw-zip-sha1 myfeed.zip  # Output only the zip SHA1 hash

This command is useful for verifying feed integrity and looking up feed versions on Transitland. Use --raw-dir-sha1 or --raw-zip-sha1 for scripting scenarios where only a specific hash is needed.`
}

func (cmd *ChecksumCommand) HelpArgs() string {
	return "<feed-path>"
}

func (cmd *ChecksumCommand) AddFlags(fl *pflag.FlagSet) {
	fl.BoolVar(&cmd.RawDirSHA1, "raw-dir-sha1", false, "Output only the directory SHA1 hash")
	fl.BoolVar(&cmd.RawZipSHA1, "raw-zip-sha1", false, "Output only the zip SHA1 hash")
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

	if cmd.RawDirSHA1 {
		fmt.Printf("%s\n", fv.SHA1Dir.Val)
	} else if cmd.RawZipSHA1 {
		fmt.Printf("%s\n", fv.SHA1)
	} else {
		fmt.Printf("Zip SHA1 (archive file): %s\n", fv.SHA1)
		fmt.Printf("Directory SHA1 (feed contents): %s\n", fv.SHA1Dir.Val)
		fmt.Printf("Find via Transitland website: https://www.transit.land/feed-versions/%s\n", fv.SHA1Dir.Val)
		fmt.Printf("Find via Transitland REST API: https://transit.land/api/v2/rest/feed_versions/%s?apikey=YOUR_API_KEY\n", fv.SHA1Dir.Val)
	}

	return nil
}
