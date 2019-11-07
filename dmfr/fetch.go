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

// FetchResult contains results of a fetch operation.
type FetchResult struct {
	FeedVersion gotransit.FeedVersion
	OnestopID   string
	Path        string
	Found       bool
	FetchError  error
}

// MainFetchFeed fetches and creates a new FeedVersion for a given Feed.
// Fetch errors are logged to Feed LastFetchError and saved.
// An error return from this function is a serious failure.
func MainFetchFeed(atx gtdb.Adapter, feedid int, outpath string) (FetchResult, error) {
	fr := FetchResult{}
	tlfeed := Feed{ID: feedid}
	if err := atx.Find(&tlfeed); err != nil {
		return fr, err
	}
	fetchtime := gotransit.OptionalTime{Time: time.Now().UTC(), Valid: true}
	tlfeed.LastFetchedAt = fetchtime
	tlfeed.LastFetchError = ""
	// Immediately save LastFetchedAt to obtain lock
	if err := atx.Update(&tlfeed, "last_fetched_at", "last_fetch_error"); err != nil {
		return fr, err
	}
	// Start fetching
	url := tlfeed.URLs.StaticCurrent
	fr, err := FetchAndCreateFeedVersion(atx, feedid, url, fetchtime.Time, outpath)
	if err != nil {
		return fr, err
	}
	if fr.FetchError != nil {
		tlfeed.LastFetchError = fr.FetchError.Error()
	} else if fr.Found {
		tlfeed.LastFetchError = ""
		tlfeed.LastSuccessfulFetchAt = fetchtime
	}
	// Save updated timestamps
	if err := atx.Update(&tlfeed, "last_fetched_at", "last_fetch_error", "last_successful_fetch_at"); err != nil {
		return fr, err
	}
	return fr, nil
}

// FetchAndCreateFeedVersion from a URL.
// Returns error if the source cannot be loaded or is invalid GTFS.
// Returns no error if the SHA1 is already present, or a FeedVersion is created.
func FetchAndCreateFeedVersion(atx gtdb.Adapter, feedid int, url string, fetchtime time.Time, outpath string) (FetchResult, error) {
	fr := FetchResult{}
	if url == "" {
		fr.FetchError = errors.New("no url")
		return fr, nil
	}
	// Download feed
	reader, err := gtcsv.NewReader(url)
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
	fv.URL = url
	fv.FeedID = feedid
	fv.FetchedAt = fetchtime
	// Is this SHA1 already present?
	checkfvid := gotransit.FeedVersion{}
	err = atx.Get(&checkfvid, "SELECT * FROM feed_versions WHERE sha1 = ?", fv.SHA1)
	if err == nil {
		// Already present
		fr.FeedVersion = checkfvid
		fr.Found = true
		return fr, nil
	} else if err == sql.ErrNoRows {
		// Not present, create below
	} else if err != nil {
		// Serious error
		return fr, err
	}
	// Copy file to output directory
	if outpath != "" {
		fn := fv.SHA1 + ".zip"
		outfn := filepath.Join(outpath, fn)
		// fmt.Printf("COPY %s -> %s\n", fv.File, outfn)
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
