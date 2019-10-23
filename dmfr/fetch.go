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
// An error return from this function is a serious failure, and should abort the txn.
func MainFetchFeed(atx gtdb.Adapter, feedid int, outpath string) (gotransit.FeedVersion, bool, error) {
	fv := gotransit.FeedVersion{}
	tlfeed := Feed{}
	tlfeed.ID = feedid
	if err := atx.Find(&tlfeed); err != nil {
		log.Info("Fetching feed: %d not found")
		return fv, false, err
	}
	url := tlfeed.URLs.StaticCurrent
	fetchtime := gotransit.OptionalTime{Time: time.Now().UTC(), Valid: true}
	tlfeed.LastFetchedAt = fetchtime
	tlfeed.LastFetchError = ""
	log.Debug("Fetching feed: %d (%s) url: %s", tlfeed.ID, tlfeed.FeedID, url)
	// Immediately save LastFetchedAt to obtain lock
	if err := atx.Update(&tlfeed, "last_fetched_at", "last_fetch_error"); err != nil {
		return fv, false, err
	}
	// Start fetching; keep fetchErr
	fv, fetchErr := FetchFeedVersion(feedid, url, fetchtime.Time)
	// Serious errors:
	var saveError error
	var copyError error
	var findError error
	var updateError error
	// Is this SHA1 already present?
	checkfvid := gotransit.FeedVersion{}
	found := false
	findError = atx.Get(&checkfvid, "SELECT * FROM feed_versions WHERE sha1 = ?", fv.SHA1)
	if findError == nil {
		// Already present
		found = true
	} else if findError == sql.ErrNoRows {
		// Not present, create
		findError = nil
		fv.ID, saveError = atx.Insert(&fv)
		if saveError != nil {
			// Serious error
			return fv, false, saveError
		}
	}
	// Copy file to output directory
	if outpath != "" && fv.File != "" {
		outfn := filepath.Join(outpath, fv.SHA1+".zip")
		// fmt.Printf("COPY %s -> %s\n", fv.File, outfn)
		copyError = copyFileContents(fv.File, outfn)
		if copyError != nil {
			return fv, false, copyError
		}
		fv.File = outfn
	}
	// Serious errors
	if findError != nil {
		return fv, false, findError
	}
	if copyError != nil {
		return fv, false, copyError
	}
	// Save Feed
	if fetchErr != nil {
		log.Info("Fetched feed: %d (%s) url: %s error: %s", tlfeed.ID, tlfeed.FeedID, url, fetchErr.Error())
		tlfeed.LastFetchError = fetchErr.Error()
	} else if found {
		log.Info("Fetched feed: %d (%s) url: %s exists: %d (%s)", tlfeed.ID, tlfeed.FeedID, url, fv.ID, fv.SHA1)
		tlfeed.LastFetchError = ""
		tlfeed.LastSuccessfulFetchAt = fetchtime
	} else {
		log.Info("Fetched feed: %d (%s) url: %s new: %d (%s)", tlfeed.ID, tlfeed.FeedID, url, fv.ID, fv.SHA1)
		tlfeed.LastFetchError = ""
		tlfeed.LastSuccessfulFetchAt = fetchtime
	}
	updateError = atx.Update(&tlfeed, "last_fetched_at", "last_fetch_error", "last_successful_fetch_at")
	if updateError != nil {
		return fv, false, updateError
	}
	// Done
	return fv, found, nil
}

func FetchFeedVersion(feedid int, url string, fetchtime time.Time) (gotransit.FeedVersion, error) {
	fv := gotransit.FeedVersion{}
	// Download feed
	reader, fetchErr := gtcsv.NewReader(url)
	if fetchErr != nil {
		return fv, fetchErr
	}
	fetchErr = reader.Open()
	if fetchErr != nil {
		return fv, fetchErr
	}
	defer reader.Close()
	//
	fv, fetchErr = gotransit.NewFeedVersionFromReader(reader)
	if fetchErr != nil {
		return fv, fetchErr
	}
	fv.URL = url
	fv.FeedID = feedid
	fv.FetchedAt = fetchtime
	return fv, fetchErr
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
