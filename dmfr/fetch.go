package dmfr

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/gtcsv"
	"github.com/interline-io/gotransit/gtdb"
)

// FetchOptions sets options for a fetch operation.
type FetchOptions struct {
	FeedID                  int
	FeedURL                 string
	IgnoreDuplicateContents bool
	Directory               string
	S3                      string
	FetchedAt               time.Time
}

// FetchResult contains results of a fetch operation.
type FetchResult struct {
	FeedVersion  gotransit.FeedVersion
	Path         string
	FoundSHA1    bool
	FoundDirSHA1 bool
	FetchError   error
}

// DatabaseFetch fetches and creates a new FeedVersion for a given Feed.
// Fetch errors are logged to Feed LastFetchError and saved.
// An error return from this function is a serious failure.
func DatabaseFetch(atx gtdb.Adapter, opts FetchOptions) (FetchResult, error) {
	fr := FetchResult{}
	// Get url
	tlfeed := Feed{ID: opts.FeedID}
	if err := atx.Find(&tlfeed); err != nil {
		return fr, err
	}
	if opts.FeedURL == "" {
		opts.FeedURL = tlfeed.URLs.StaticCurrent
	}
	// Get state
	tlstate := FeedState{FeedID: opts.FeedID}
	if err := atx.Get(&tlstate, `SELECT * FROM feed_states WHERE feed_id = ?`, opts.FeedID); err == sql.ErrNoRows {
		tlstate.ID, err = atx.Insert(&tlstate)
		if err != nil {
			return fr, err
		}
	} else if err != nil {
		return fr, err
	}
	if opts.FetchedAt.IsZero() {
		opts.FetchedAt = time.Now().UTC()
	}
	tlstate.LastFetchedAt = gotransit.OptionalTime{Time: opts.FetchedAt, Valid: true}
	tlstate.LastFetchError = ""
	// Immediately save LastFetchedAt to obtain lock
	if err := atx.Update(&tlstate, "last_fetched_at", "last_fetch_error"); err != nil {
		return fr, err
	}
	// Start fetching
	fr, err := FetchAndCreateFeedVersion(atx, opts)
	if err != nil {
		return fr, err
	}
	if fr.FetchError != nil {
		tlstate.LastFetchError = fr.FetchError.Error()
	} else {
		tlstate.LastSuccessfulFetchAt = gotransit.OptionalTime{Time: opts.FetchedAt, Valid: true}
	}
	// else if fr.FoundSHA1 || fr.FoundDirSHA1 {}
	// Save updated timestamps
	if err := atx.Update(&tlstate, "last_fetched_at", "last_fetch_error", "last_successful_fetch_at"); err != nil {
		return fr, err
	}

	return fr, nil
}

// FetchAndCreateFeedVersion from a URL.
// Returns error if the source cannot be loaded or is invalid GTFS.
// Returns no error if the SHA1 is already present, or a FeedVersion is created.
func FetchAndCreateFeedVersion(atx gtdb.Adapter, opts FetchOptions) (FetchResult, error) {
	fr := FetchResult{}
	if opts.FeedURL == "" {
		fr.FetchError = errors.New("no url")
		return fr, nil
	}
	// Download feed
	secret := Secret{}                    // TODO
	auth := gotransit.FeedAuthorization{} // TODO
	tmpfile, err := AuthenticatedRequest(opts.FeedURL, secret, auth)
	if err != nil {
		fr.FetchError = err
		return fr, nil
	}
	// Open
	reader, err := gtcsv.NewReader(tmpfile)
	if err != nil {
		fr.FetchError = err
		return fr, nil
	}
	if err := reader.Open(); err != nil {
		fr.FetchError = err
		return fr, nil
	}
	defer reader.Close()
	// Get initialized FeedVersion
	fv, err := gotransit.NewFeedVersionFromReader(reader)
	if err != nil {
		fr.FetchError = err
		return fr, nil
	}
	fv.URL = opts.FeedURL
	fv.FeedID = opts.FeedID
	fv.FetchedAt = opts.FetchedAt
	// Is this SHA1 already present?
	checkfvid := gotransit.FeedVersion{}
	err = atx.Get(&checkfvid, "SELECT * FROM feed_versions WHERE sha1 = ? OR sha1_dir = ?", fv.SHA1, fv.SHA1Dir)
	if err == nil {
		// Already present
		fr.FeedVersion = checkfvid
		fr.FoundSHA1 = (checkfvid.SHA1 == fv.SHA1)
		fr.FoundDirSHA1 = (checkfvid.SHA1Dir == fv.SHA1Dir)
		return fr, nil
	} else if err == sql.ErrNoRows {
		// Not present, create below
	} else if err != nil {
		// Serious error
		return fr, err
	}
	// Upload file or copy to output directory
	if opts.S3 != "" {
		awscmd := exec.Command(
			"aws",
			"s3",
			"cp",
			reader.Path(),
			fmt.Sprintf("%s/%s.zip", opts.S3, fv.SHA1),
		)
		if output, err := awscmd.Output(); err != nil {
			return fr, fmt.Errorf("upload error: %s: %s", err, output)
		}
	}
	if opts.Directory != "" {
		fn := fv.SHA1 + ".zip"
		outfn := filepath.Join(opts.Directory, fn)
		if err := copyFileContents(reader.Path(), outfn); err != nil {
			return fr, err
		}
		fv.File = fn
		fr.Path = fv.File // TODO: remove
	}
	// Return fv
	fv.ID, err = atx.Insert(&fv)
	fr.FeedVersion = fv
	if err == nil {
		return fr, err
	}
	return fr, nil
}

func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}
