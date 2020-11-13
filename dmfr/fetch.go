package dmfr

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
)

// FetchOptions sets options for a fetch operation.
type FetchOptions struct {
	FeedURL                 string
	FeedID                  string
	FeedCreate              bool
	IgnoreDuplicateContents bool
	Directory               string
	S3                      string
	FetchedAt               time.Time
	Secrets                 Secrets
}

// FetchResult contains results of a fetch operation.
type FetchResult struct {
	FeedVersion  tl.FeedVersion
	Path         string
	FoundSHA1    bool
	FoundDirSHA1 bool
	FetchError   error
}

// DatabaseFetch fetches and creates a new FeedVersion for a given Feed.
// An error return from this function is a serious failure.
// Saves FeedState.LastFetchError for regular failures.
func DatabaseFetch(atx tldb.Adapter, opts FetchOptions) (FetchResult, error) {
	fr := FetchResult{}
	// Get feed, create if not present and FeedCreate is specified
	tlfeed := Feed{}
	if err := atx.Get(&tlfeed, `SELECT * FROM current_feeds WHERE onestop_id = ?`, opts.FeedID); err == sql.ErrNoRows && opts.FeedCreate {
		tlfeed.FeedID = opts.FeedID
		tlfeed.Spec = "gtfs"
		if tlfeed.ID, err = atx.Insert(&tlfeed); err != nil {
			return fr, err
		}
	} else if err != nil {
		return fr, errors.New("feed does not exist")
	}
	if opts.FeedURL == "" {
		opts.FeedURL = tlfeed.URLs.StaticCurrent
	}
	if opts.FetchedAt.IsZero() {
		opts.FetchedAt = time.Now().UTC()
	}
	// Get state, create if necessary
	tlstate := FeedState{FeedID: tlfeed.ID}
	if err := atx.Get(&tlstate, `SELECT * FROM feed_states WHERE feed_id = ?`, tlfeed.ID); err == sql.ErrNoRows {
		tlstate.ID, err = atx.Insert(&tlstate)
		if err != nil {
			return fr, err
		}
	} else if err != nil {
		return fr, err
	}
	tlstate.LastFetchedAt = tl.OptionalTime{Time: opts.FetchedAt, Valid: true}
	tlstate.LastFetchError = ""
	// Immediately save LastFetchedAt
	if err := atx.Update(&tlstate, "last_fetched_at", "last_fetch_error"); err != nil {
		return fr, err
	}
	// Start fetching
	fr, err := fetchAndCreateFeedVersion(atx, tlfeed, opts)
	if err != nil {
		return fr, err
	}
	if fr.FetchError != nil {
		tlstate.LastFetchError = fr.FetchError.Error()
	} else {
		tlstate.LastSuccessfulFetchAt = tl.OptionalTime{Time: opts.FetchedAt, Valid: true}
	}
	// else if fr.FoundSHA1 || fr.FoundDirSHA1 {}
	// Save updated timestamps
	if err := atx.Update(&tlstate, "last_fetched_at", "last_fetch_error", "last_successful_fetch_at"); err != nil {
		return fr, err
	}
	return fr, nil
}

// fetchAndCreateFeedVersion from a URL.
// Returns an error if a serious failure occurs, such as database or filesystem access.
// Sets FetchResult.FetchError if a regular failure occurs, such as a 404.
// feed is an argument to provide the ID, File, and Authorization.
func fetchAndCreateFeedVersion(atx tldb.Adapter, feed tl.Feed, opts FetchOptions) (FetchResult, error) {
	fr := FetchResult{}
	if opts.FeedURL == "" {
		fr.FetchError = errors.New("no url")
		return fr, nil
	}
	// Handle fragments
	u, err := url.Parse(opts.FeedURL)
	if err != nil {
		fr.FetchError = errors.New("cannot parse url")
		return fr, nil
	}
	// Get secret
	secret := Secret{}
	if a, err := opts.Secrets.MatchFeed(opts.FeedID); err == nil {
		secret = a
	} else if a, err := opts.Secrets.MatchFilename(feed.File); err == nil {
		secret = a
	} else if feed.Authorization.Type != "" {
		fr.FetchError = errors.New("no secret found")
		return fr, nil
	}
	// Check reader type
	reader, err := tlcsv.NewReader(opts.FeedURL)
	if err != nil {
		fr.FetchError = err
		return fr, nil
	}
	// Override the default URLAdapter
	if u.Scheme == "http" || u.Scheme == "https" || u.Scheme == "ftp" || u.Scheme == "s3" {
		aa := AuthenticatedURLAdapter{}
		if err := aa.Download(opts.FeedURL, feed.Authorization, secret); err != nil {
			fr.FetchError = err
			return fr, nil
		}
		reader.Adapter = &aa
	}
	// Open
	if err := reader.Open(); err != nil {
		fr.FetchError = err
		return fr, nil
	}
	defer reader.Close()
	// Get initialized FeedVersion
	fv, err := tl.NewFeedVersionFromReader(reader)
	if err != nil {
		fr.FetchError = err
		return fr, nil
	}
	fv.URL = opts.FeedURL
	fv.FeedID = feed.ID
	fv.FetchedAt = opts.FetchedAt
	// Is this SHA1 already present?
	checkfvid := tl.FeedVersion{}
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
	if err != nil {
		return fr, err
	}
	// Update stats records
	if err := createFeedStats(atx, reader, fv.ID); err != nil {
		return fr, err
	}
	return fr, nil
}

func createFeedStats(atx tldb.Adapter, reader *tlcsv.Reader, fvid int) error {
	// Get FeedVersionFileInfos
	fvfis, err := NewFeedVersionFileInfosFromReader(reader)
	if err != nil {
		return err
	}
	for _, fvfi := range fvfis {
		fvfi.FeedVersionID = fvid
		if _, err := atx.Insert(&fvfi); err != nil {
			return err
		}
	}
	// Get service statistics
	fvsls, err := NewFeedVersionServiceInfosFromReader(reader)
	if err != nil {
		return err
	}
	// Use batch insert?
	for _, fvsl := range fvsls {
		fvsl.FeedVersionID = fvid
		if _, err := atx.Insert(&fvsl); err != nil {
			return err
		}
	}
	return nil
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

// dmfrGetReaderURL helps load a file from an S3 or Directory location
func dmfrGetReaderURL(s3 string, directory string, url string) string {
	if s3 != "" {
		url = s3 + "/" + url
	} else if directory != "" {
		url = filepath.Join(directory, url)
	}
	urlsplit := strings.SplitN(url, "#", 2)
	if len(urlsplit) > 1 {
		url = url + "#" + urlsplit[1]
	}
	return url
}
