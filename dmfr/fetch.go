package dmfr

import (
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/gtcsv"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/log"
)

// MainFetchFeed .
// Fetch errors are logged to Feed LastFetchError and saved.
// An error return from this function is a serious failure.
// This should run inside a transaction.
func MainFetchFeed(atx gtdb.Adapter, feedid int, outpath string) (gotransit.FeedVersion, bool, error) {
	fv := gotransit.FeedVersion{}
	tlfeed := Feed{}
	tlfeed.ID = feedid
	if err := atx.Find(&tlfeed); err != nil {
		log.Info("Fetching feed: %d not found")
		return fv, false, err
	}
	log.Debug("Fetching feed: %d (%s) url: %s", tlfeed.ID, tlfeed.FeedID, tlfeed.URLs.StaticCurrent)
	fetchtime := gotransit.OptionalTime{Time: time.Now().UTC(), Valid: true}
	tlfeed.LastFetchedAt = fetchtime
	tlfeed.LastFetchError = ""
	// Immediately save LastFetchedAt to obtain lock
	if err := atx.Update(&tlfeed, "last_fetched_at", "last_fetch_error"); err != nil {
		return fv, false, err
	}
	// Start fetching
	fv, found, err := FetchAndCreateFeedVersion(atx, feedid, tlfeed.URLs.StaticCurrent, fetchtime.Time, outpath)
	if err != nil {
		log.Info("Fetched feed: %d (%s) url: %s error: %s", tlfeed.ID, tlfeed.FeedID, tlfeed.URLs.StaticCurrent, err.Error())
		tlfeed.LastFetchError = err.Error()
	} else if found {
		log.Info("Fetched feed: %d (%s) url: %s exists: %d (%s)", tlfeed.ID, tlfeed.FeedID, tlfeed.URLs.StaticCurrent, fv.ID, fv.SHA1)
		tlfeed.LastFetchError = ""
		tlfeed.LastSuccessfulFetchAt = fetchtime
	} else {
		log.Info("Fetched feed: %d (%s) url: %s new: %d (%s)", tlfeed.ID, tlfeed.FeedID, tlfeed.URLs.StaticCurrent, fv.ID, fv.SHA1)
		tlfeed.LastFetchError = ""
		tlfeed.LastSuccessfulFetchAt = fetchtime
	}
	// Save updated timestamps
	if err := atx.Update(&tlfeed, "last_fetched_at", "last_fetch_error", "last_successful_fetch_at"); err != nil {
		return fv, false, err
	}
	if fv.File != "" {
		log.Info("Saved feed to %s", fv.File)
	}
	return fv, found, nil
}

// FetchAndCreateFeedVersion from a URL.
// Returns error if the source cannot be loaded or is invalid GTFS.
// Returns no error if the SHA1 is already present, or a FeedVersion is created.
func FetchAndCreateFeedVersion(atx gtdb.Adapter, feedid int, url string, fetchtime time.Time, outpath string) (gotransit.FeedVersion, bool, error) {
	fv := gotransit.FeedVersion{}
	// Download feed
	reader, err := gtcsv.NewReader(url)
	if err != nil {
		return fv, false, err
	}
	if err := reader.Open(); err != nil {
		return fv, false, err
	}
	defer reader.Close()
	//
	fv, err = gotransit.NewFeedVersionFromReader(reader)
	if err != nil {
		return fv, false, err
	}
	fv.URL = url
	fv.FeedID = feedid
	fv.FetchedAt = fetchtime
	// Is this SHA1 already present?
	checkfvid := gotransit.FeedVersion{}
	err = atx.Get(&checkfvid, "SELECT * FROM feed_versions WHERE sha1 = ?", fv.SHA1)
	if err == nil {
		// Already present
		return checkfvid, true, nil
	} else if err == sql.ErrNoRows {
		// Not present, create
		fv.ID, err = atx.Insert(&fv)
	}
	// Copy file to output directory
	if outpath != "" && fv.File != "" {
		outfn := filepath.Join(outpath, fv.SHA1+".zip")
		// fmt.Printf("COPY %s -> %s\n", fv.File, outfn)
		err := copyFileContents(fv.File, outfn)
		if err != nil {
			return fv, false, err
		}
		fv.File = outfn
	}
	// Return any query error or insert error
	return fv, false, err
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
