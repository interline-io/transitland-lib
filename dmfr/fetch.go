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
	Feed                    Feed
	FeedURL                 string
	IgnoreDuplicateContents bool
	Directory               string
	S3                      string
	FetchedAt               time.Time
	Secrets                 Secrets
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
// An error return from this function is a serious failure.
// Saves FeedState.LastFetchError for regular failures.
func DatabaseFetch(atx gtdb.Adapter, opts FetchOptions) (FetchResult, error) {
	fr := FetchResult{}
	// Get url
	tlfeed := Feed{ID: opts.Feed.ID}
	if err := atx.Find(&tlfeed); err != nil {
		return fr, err
	}
	if opts.FeedURL == "" {
		opts.FeedURL = tlfeed.URLs.StaticCurrent
	}
	if opts.FetchedAt.IsZero() {
		opts.FetchedAt = time.Now().UTC()
	}
	// Get state
	tlstate := FeedState{FeedID: opts.Feed.ID}
	if err := atx.Get(&tlstate, `SELECT * FROM feed_states WHERE feed_id = ?`, opts.Feed.ID); err == sql.ErrNoRows {
		tlstate.ID, err = atx.Insert(&tlstate)
		if err != nil {
			return fr, err
		}
	} else if err != nil {
		return fr, err
	}
	tlstate.LastFetchedAt = gotransit.OptionalTime{Time: opts.FetchedAt, Valid: true}
	tlstate.LastFetchError = ""
	// Immediately save LastFetchedAt
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
// Returns an error if a serious failure occurs, such as database or filesystem access.
// Sets FetchResult.FetchError if a regular failure occurs, such as a 404.
func FetchAndCreateFeedVersion(atx gtdb.Adapter, opts FetchOptions) (FetchResult, error) {
	fr := FetchResult{}
	if opts.FeedURL == "" {
		fr.FetchError = errors.New("no url")
		return fr, nil
	}
	// Get secret
	secret := Secret{}
	if a, err := opts.Secrets.MatchFeed(opts.Feed.FeedID); err == nil {
		secret = a
	} else if a, err := opts.Secrets.MatchFilename(opts.Feed.File); err == nil {
		secret = a
	} else if opts.Feed.Authorization.Type != "" {
		fr.FetchError = errors.New("no secret found")
		return fr, nil
	}
	// Check reader type
	reader, err := gtcsv.NewReader(opts.FeedURL)
	if err != nil {
		fr.FetchError = err
		return fr, nil
	}
	// This isn't great, but... we only want download/tmpfile/cleanup for URLAdapter
	if _, ok := reader.Adapter.(*gtcsv.URLAdapter); ok {
		// Download feed
		auth := opts.Feed.Authorization
		tmpfile, err := AuthenticatedRequest(opts.FeedURL, secret, auth)
		defer os.Remove(tmpfile)
		if err != nil {
			fr.FetchError = err
			return fr, nil
		}
		reader.Adapter = gtcsv.NewZipAdapter(tmpfile)
	}
	// Open
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
	fv.FeedID = opts.Feed.ID
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
