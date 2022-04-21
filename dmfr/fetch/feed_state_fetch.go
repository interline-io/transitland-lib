package fetch

import (
	"database/sql"
	"errors"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
)

// FeedStateFetch fetches the specified feed and updates FeedState.
// An error return from this function is a serious failure, e.g. db or disk error.
func FeedStateFetch(atx tldb.Adapter, opts Options) (tl.FeedVersion, Result, error) {
	fr := Result{}
	fv := tl.FeedVersion{}
	// Get feed, create if not present and FeedCreate is specified
	tlfeed := tl.Feed{}
	if err := atx.Get(&tlfeed, `SELECT * FROM current_feeds WHERE onestop_id = ?`, opts.FeedID); err == sql.ErrNoRows && opts.FeedCreate {
		tlfeed.FeedID = opts.FeedID
		tlfeed.Spec = "gtfs"
		if tlfeed.ID, err = atx.Insert(&tlfeed); err != nil {
			return fv, fr, err
		}
	} else if err != nil {
		return fv, fr, errors.New("feed does not exist")
	}
	opts.URLType = "manual"
	if opts.FeedURL == "" {
		opts.URLType = "static_current"
		opts.FeedURL = tlfeed.URLs.StaticCurrent
	}
	if opts.FetchedAt.IsZero() {
		opts.FetchedAt = time.Now().UTC()
	}
	// Get state, create if necessary
	tlstate := dmfr.FeedState{FeedID: tlfeed.ID}
	if err := atx.Get(&tlstate, `SELECT * FROM feed_states WHERE feed_id = ?`, tlfeed.ID); err == sql.ErrNoRows {
		tlstate.ID, err = atx.Insert(&tlstate)
		if err != nil {
			return fv, fr, err
		}
	} else if err != nil {
		return fv, fr, err
	}
	// Immediately save LastFetchedAt
	tlstate.LastFetchedAt = tl.NewTime(opts.FetchedAt)
	tlstate.LastFetchError = ""
	tlstate.UpdateTimestamps()
	if err := atx.Update(&tlstate, "last_fetched_at", "last_fetch_error"); err != nil {
		return fv, fr, err
	}
	// Start fetching
	fv, fr, err := StaticFetch(atx, tlfeed, opts)
	if err != nil {
		return fv, fr, err
	}
	if fr.FetchError != nil {
		tlstate.LastFetchError = fr.FetchError.Error()
	} else {
		tlstate.LastSuccessfulFetchAt = tl.NewTime(opts.FetchedAt)
	}
	// Save updated timestamps
	tlstate.UpdateTimestamps()
	if err := atx.Update(&tlstate, "last_fetched_at", "last_fetch_error", "last_successful_fetch_at"); err != nil {
		return fv, fr, err
	}
	return fv, fr, nil
}
