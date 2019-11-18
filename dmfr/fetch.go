package dmfr

import (
	"database/sql"
	"errors"
	"io"
	"os"
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
	FetchTime               time.Time
}

// FetchResult contains results of a fetch operation.
type FetchResult struct {
	FeedVersion  gotransit.FeedVersion
	Path         string
	FoundSHA1    bool
	FoundDirSHA1 bool
	FetchError   error
}

// MainFetchFeed fetches and creates a new FeedVersion for a given Feed.
// Fetch errors are logged to Feed LastFetchError and saved.
// An error return from this function is a serious failure.
func MainFetchFeed(atx gtdb.Adapter, opts FetchOptions) (FetchResult, error) {
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
	if opts.FetchTime.IsZero() {
		opts.FetchTime = time.Now().UTC()
	}
	tlstate.LastFetchedAt = gotransit.OptionalTime{Time: opts.FetchTime, Valid: true}
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
		tlstate.LastSuccessfulFetchAt = gotransit.OptionalTime{Time: opts.FetchTime, Valid: true}
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
	reader, err := gtcsv.NewReader(opts.FeedURL)
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
	fv.FetchedAt = opts.FetchTime
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
	// Copy file to output directory
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
