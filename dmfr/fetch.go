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
	"github.com/interline-io/gotransit/internal/log"
)

// FetchOptions sets options for a fetch operation.
type FetchOptions struct {
	Feed                    Feed
	FeedURL                 string
	IgnoreDuplicateContents bool
	Directory               string
	S3                      string
	FetchTime               time.Time
	// trying something out...
	secrets Secrets
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
	// Get state
	tlstate := FeedState{FeedID: opts.Feed.ID}
	if err := atx.Get(&tlstate, `SELECT * FROM feed_states WHERE feed_id = ?`, opts.Feed.ID); err == sql.ErrNoRows {
		tlstate.ID, err = atx.Insert(&tlstate)
		if err != nil {
			return fr, err
		}
	} else if err != nil {
		return fr, err // rollback
	}
	if opts.FetchTime.IsZero() {
		opts.FetchTime = time.Now().UTC()
	}
	tlstate.LastFetchedAt = gotransit.OptionalTime{Time: opts.FetchTime, Valid: true}
	tlstate.LastFetchError = ""
	// Immediately save LastFetchedAt
	if err := atx.Update(&tlstate, "last_fetched_at", "last_fetch_error"); err != nil {
		return fr, err // rollback
	}
	// Start fetching
	fr, err := fetchDownload(opts)
	if err != nil {
		return fr, err // rollback
	}
	// Check if this version already exists
	check := gotransit.FeedVersion{}
	if err := atx.Get(&check, "SELECT * FROM feed_versions WHERE sha1 = ? OR sha1_dir = ?", fr.FeedVersion.SHA1, fr.FeedVersion.SHA1Dir); err == nil {
		// Already present
		fr.FeedVersion = check
		fr.FoundSHA1 = (check.SHA1 == fr.FeedVersion.SHA1)
		fr.FoundDirSHA1 = (check.SHA1Dir == fr.FeedVersion.SHA1Dir)
	} else if err == sql.ErrNoRows {
		// Not present
		fr.FoundSHA1 = false
		fr.FoundDirSHA1 = false
	} else if err != nil {
		return fr, err // rollback
	}
	// Upload
	if fr.FetchError == nil && fr.FoundSHA1 == false && fr.FoundDirSHA1 == false {
		path, err := fetchUpload(opts, fr.FeedVersion)
		if err != nil {
			return fr, err // rollback
		}
		fr.FeedVersion.File = path
	}
	// Create FeedVersion record
	if fr.FetchError == nil && fr.FoundSHA1 == false && fr.FoundDirSHA1 == false {
		fvid := 0
		fvid, err = atx.Insert(&fr.FeedVersion)
		if err != nil {
			return fr, err // rollback
		}
		fr.FeedVersion.ID = fvid
	}
	// Update FeedState
	if fr.FetchError != nil {
		tlstate.LastFetchError = fr.FetchError.Error()
	} else {
		tlstate.LastFetchError = ""
		tlstate.LastSuccessfulFetchAt = gotransit.OptionalTime{Time: opts.FetchTime, Valid: true}
	}
	// Save updated timestamps
	if err := atx.Update(&tlstate, "last_fetched_at", "last_fetch_error", "last_successful_fetch_at"); err != nil {
		return fr, err
	}
	return fr, nil
}

// Fetch a feed.
func Fetch(opts FetchOptions) (FetchResult, error) {
	fr, err := fetchDownload(opts)
	if err != nil {
		return fr, err
	}
	path, err := fetchUpload(opts, fr.FeedVersion)
	if err != nil {
		return fr, err
	}
	fr.FeedVersion.File = path
	return fr, nil
}

func fetchDownload(opts FetchOptions) (FetchResult, error) {
	fr := FetchResult{}
	if a := opts.Feed.URLs.StaticCurrent; a != "" {
		opts.FeedURL = a
	}
	if opts.FeedURL == "" {
		fr.FetchError = errors.New("no url")
		return fr, nil
	}
	// Get secret
	secret := Secret{}
	if a, err := opts.secrets.MatchFeed(opts.Feed.FeedID); err == nil {
		secret = a
	} else if a, err := opts.secrets.MatchFilename(opts.Feed.File); err == nil {
		secret = a
	} else if opts.Feed.Authorization.Type != "" {
		fr.FetchError = errors.New("no secret found")
		return fr, nil
	}
	// Download feed
	tmpfile, err := AuthenticatedRequest(opts.FeedURL, secret, opts.Feed.Authorization)
	if err != nil {
		return fr, err
	}
	// Check feed
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
	fv.FeedID = opts.Feed.ID
	fv.FetchedAt = opts.FetchTime
	fr.FeedVersion = fv
	return fr, nil
}

func fetchUpload(opts FetchOptions, fv gotransit.FeedVersion) (string, error) {
	// Upload file or copy to output directory
	path := fv.File
	outpath := fv.File
	if opts.S3 != "" {
		outpath = fv.SHA1 + ".zip"
		log.Debug("Uploading %s -> %s/%s", path, opts.S3, outpath)
		awscmd := exec.Command(
			"aws",
			"s3",
			"cp",
			path,
			fmt.Sprintf("%s/%s", opts.S3, outpath),
		)
		if output, err := awscmd.Output(); err != nil {
			return "", fmt.Errorf("upload error: %s: %s", err, output)
		}
	}
	if opts.Directory != "" {
		outpath := fv.SHA1 + ".zip"
		log.Debug("Copying %s -> %s", path, outpath)
		outfn := filepath.Join(opts.Directory, outpath)
		if err := copyFileContents(path, outfn); err != nil {
			return "", err
		}
	}
	return outpath, nil
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
