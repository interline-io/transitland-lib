package dmfr

import (
	"database/sql"
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
func MainFetchFeed(atx gtdb.Adapter, feedid int) (int, bool, string, error) {
	tlfeed := Feed{}
	tlfeed.ID = feedid
	fvid := 0
	if err := atx.Find(&tlfeed); err != nil {
		log.Info("Fetching feed: %d not found")
		return fvid, false, "", err
	}
	log.Debug("Fetching feed: %d (%s) url: %s", tlfeed.ID, tlfeed.FeedID, tlfeed.URLs.StaticCurrent)
	fetchtime := gotransit.OptionalTime{Time: time.Now().UTC(), Valid: true}
	tlfeed.LastFetchedAt = fetchtime
	tlfeed.LastFetchError = ""
	// Immediately save LastFetchedAt to obtain lock
	if err := atx.Update(&tlfeed, "last_fetched_at", "last_fetch_error"); err != nil {
		return fvid, false, "", err
	}
	// Start fetching
	fvid2, found, sha1, err := FetchAndCreateFeedVersion(atx, feedid, tlfeed.URLs.StaticCurrent, fetchtime.Time)
	if err != nil {
		log.Info("Fetched feed: %d (%s) url: %s error: %s", tlfeed.ID, tlfeed.FeedID, tlfeed.URLs.StaticCurrent, err.Error())
		tlfeed.LastFetchError = err.Error()
	} else if found {
		log.Info("Fetched feed: %d (%s) url: %s exists: %d (%s)", tlfeed.ID, tlfeed.FeedID, tlfeed.URLs.StaticCurrent, fvid2, sha1)
		tlfeed.LastFetchError = ""
		tlfeed.LastSuccessfulFetchAt = fetchtime
	} else {
		log.Info("Fetched feed: %d (%s) url: %s new: %d (%s)", tlfeed.ID, tlfeed.FeedID, tlfeed.URLs.StaticCurrent, fvid2, sha1)
		tlfeed.LastFetchError = ""
		tlfeed.LastSuccessfulFetchAt = fetchtime
	}
	// Save updated timestamps
	if err := atx.Update(&tlfeed, "last_fetched_at", "last_fetch_error", "last_successful_fetch_at"); err != nil {
		return fvid2, found, "", err
	}
	return fvid2, found, sha1, nil
}

// FetchAndCreateFeedVersion from a URL.
// Returns error if the source cannot be loaded or is invalid GTFS.
// Returns no error if the SHA1 is already present, or a FeedVersion is created.
func FetchAndCreateFeedVersion(atx gtdb.Adapter, feedid int, url string, fetchtime time.Time) (int, bool, string, error) {
	// Download feed
	reader, err := gtcsv.NewReader(url)
	if err != nil {
		return 0, false, "", err
	}
	if err := reader.Open(); err != nil {
		return 0, false, "", err
	}
	defer reader.Close()
	fv, err := gotransit.NewFeedVersionFromReader(reader)
	if err != nil {
		return 0, false, "", err
	}
	fv.URL = url
	fv.FeedID = feedid
	fv.FetchedAt = fetchtime
	// Is this SHA1 already present?
	checkfvid := gotransit.FeedVersion{}
	err = atx.Get(&checkfvid, "SELECT * FROM feed_versions WHERE sha1 = ?", fv.SHA1)
	if err == nil {
		// Already present
		return checkfvid.ID, true, checkfvid.SHA1, nil
	} else if err == sql.ErrNoRows {
		// Not present, create
		fv.ID, err = atx.Insert(&fv)
	}
	// Return any query error or insert error
	return fv.ID, false, fv.SHA1, err
}
