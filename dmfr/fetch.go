package dmfr

import (
	"database/sql"
	"errors"
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
	fvid2, found, sha1, err := FetchAndCreateFeedVersion(atx, feedid, tlfeed.URL, fetchtime.Time)
	if err != nil {
		log.Info("Fetched feed: %d (%s) url: %s error: %s", tlfeed.ID, tlfeed.FeedID, tlfeed.URLs.StaticCurrent, err.Error())
		tlfeed.LastFetchError = err.Error()
	} else if found {
		log.Info("Fetched feed: %d (%s) url: %s exists: %d (%s)", tlfeed.ID, tlfeed.FeedID, tlfeed.URLs.StaticCurrent, fvid, sha1)
		tlfeed.LastFetchError = ""
		tlfeed.LastSuccessfulFetchAt = fetchtime
		fvid = fvid2
	} else {
		log.Info("Fetched feed: %d (%s) url: %s new: %d (%s)", tlfeed.ID, tlfeed.FeedID, tlfeed.URLs.StaticCurrent, fvid, sha1)
		tlfeed.LastFetchError = ""
		tlfeed.LastSuccessfulFetchAt = fetchtime
		fvid = fvid2
	}
	// Save updated timestamps
	if err := atx.Update(&tlfeed, "last_fetched_at", "last_fetch_error", "last_successful_fetch_at"); err != nil {
		return fvid, found, "", err
	}
	return fvid, found, sha1, nil
}

// FetchAndCreateFeedVersion from a URL.
// Returns error if the source cannot be loaded or is invalid GTFS.
// Returns no error if the SHA1 is already present, or a FeedVersion is created.
func FetchAndCreateFeedVersion(atx gtdb.Adapter, feedid int, url string, fetchtime time.Time) (int, bool, string, error) {
	fvid := 0
	fv, err := NewFeedVersionFromURL(url)
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
		fvid, err = atx.Insert(&fv)
	}
	// Return any query error or insert error
	return fvid, false, fv.SHA1, err
}

// NewFeedVersionFromURL returns a new FeedVersion initialized from the given URL.
func NewFeedVersionFromURL(url string) (gotransit.FeedVersion, error) {
	// Init FV
	fv := gotransit.FeedVersion{}
	fv.URL = url
	fv.FeedType = "gtfs"
	// Download feed
	reader, err := gtcsv.NewReader(url)
	if err != nil {
		return fv, err
	}
	if err := reader.Open(); err != nil {
		return fv, err
	}
	defer reader.Close()
	// fv.FetchedAt = &fetchtime
	// Are we a zip archive? Can we read the SHA1?
	if h, err := getSHA1(reader); err == nil {
		fv.SHA1 = h
	} else {
		return fv, err
	}
	// Perform basic GTFS validity checks
	if errs := reader.ValidateStructure(); len(errs) > 0 {
		return fv, errs[0]
	}
	// Get service dates
	start, end, err := servicePeriod(reader)
	if err != nil {
		return fv, err
	}
	fv.EarliestCalendarDate = start
	fv.LatestCalendarDate = end
	return fv, nil
}

type canSHA1 interface {
	SHA1() (string, error)
}

func getSHA1(reader gotransit.Reader) (string, error) {
	ad, ok := reader.(canSHA1)
	if !ok {
		return "", errors.New("not a zip source")
	}
	h, err := ad.SHA1()
	if err != nil {
		return "", err
	}
	return h, nil
}

func servicePeriod(reader gotransit.Reader) (time.Time, time.Time, error) {
	var start time.Time
	var end time.Time
	for c := range reader.Calendars() {
		if start.IsZero() || c.StartDate.Before(start) {
			start = c.StartDate
		}
		if end.IsZero() || c.EndDate.After(end) {
			end = c.EndDate
		}
	}
	for cd := range reader.CalendarDates() {
		if cd.ExceptionType != 1 {
			continue
		}
		if start.IsZero() || cd.Date.Before(start) {
			start = cd.Date
		}
		if end.IsZero() || cd.Date.After(end) {
			end = cd.Date
		}
	}
	if start.IsZero() || end.IsZero() {
		return start, end, errors.New("start or end dates were empty")
	}
	if end.Before(start) {
		return start, end, errors.New("end before start")
	}
	return start, end, nil
}
