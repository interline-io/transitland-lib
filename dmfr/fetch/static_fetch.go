package fetch

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
)

// DatabaseFetch fetches the specified feed and updates FeedState.
// An error return from this function is a serious failure, e.g. db or disk error.
func DatabaseFetch(atx tldb.Adapter, opts Options) (Result, error) {
	fr := Result{}
	// Get feed, create if not present and FeedCreate is specified
	tlfeed := tl.Feed{}
	if err := atx.Get(&tlfeed, `SELECT * FROM current_feeds WHERE onestop_id = ?`, opts.FeedID); err == sql.ErrNoRows && opts.FeedCreate {
		tlfeed.FeedID = opts.FeedID
		tlfeed.Spec = "gtfs"
		if tlfeed.ID, err = atx.Insert(&tlfeed); err != nil {
			return fr, err
		}
	} else if err != nil {
		return fr, errors.New("feed does not exist")
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
			return fr, err
		}
	} else if err != nil {
		return fr, err
	}
	// Immediately save LastFetchedAt
	tlstate.LastFetchedAt = tl.NewTime(opts.FetchedAt)
	tlstate.LastFetchError = ""
	tlstate.UpdateTimestamps()
	if err := atx.Update(&tlstate, "last_fetched_at", "last_fetch_error"); err != nil {
		return fr, err
	}
	// Start fetching
	fr, err := StaticFetch(atx, tlfeed, opts)
	if err != nil {
		return fr, err
	}
	if fr.FetchError != nil {
		tlstate.LastFetchError = fr.FetchError.Error()
	} else {
		tlstate.LastSuccessfulFetchAt = tl.NewTime(opts.FetchedAt)
	}
	// Save updated timestamps
	tlstate.UpdateTimestamps()
	if err := atx.Update(&tlstate, "last_fetched_at", "last_fetch_error", "last_successful_fetch_at"); err != nil {
		return fr, err
	}
	return fr, nil
}

// StaticFetch from a URL. Creates FeedVersion and FeedFetch records.
// Returns an error if a serious failure occurs, such as database or filesystem access.
// Sets Result.FetchError if a regular failure occurs, such as a 404.
// feed is an argument to provide the ID, File, and Authorization.
func StaticFetch(atx tldb.Adapter, feed tl.Feed, opts Options) (Result, error) {
	cb := func(tmpfilepath string) (validationResponse, error) {
		vr := validationResponse{}
		// Open reader
		if a := strings.SplitN(opts.FeedURL, "#", 2); len(a) > 1 {
			tmpfilepath = tmpfilepath + "#" + a[1]
		}
		reader, err := tlcsv.NewReaderFromAdapter(tlcsv.NewZipAdapter(tmpfilepath))
		if err != nil {
			vr.Error = err
			return vr, nil
		}
		if err := reader.Open(); err != nil {
			vr.Error = err
			return vr, nil
		}
		defer reader.Close()
		// Get initialized FeedVersion
		fv, err := tl.NewFeedVersionFromReader(reader)
		if err != nil {
			return vr, err
		}
		fv.URL = opts.FeedURL
		fv.FeedID = feed.ID
		fv.FetchedAt = opts.FetchedAt
		fv.CreatedBy = opts.CreatedBy
		fv.Name = opts.Name
		fv.Description = opts.Description
		fv.File = fmt.Sprintf("%s.zip", fv.SHA1)
		// Is this SHA1 already present?
		checkfvid := tl.FeedVersion{}
		err = atx.Get(&checkfvid, "SELECT * FROM feed_versions WHERE sha1 = ? OR sha1_dir = ?", fv.SHA1, fv.SHA1Dir)
		if err == nil {
			// Already present
			vr.FeedVersion = checkfvid
			vr.Found = true
			return vr, nil
		} else if err == sql.ErrNoRows {
			// Not present, create below
		} else if err != nil {
			// Serious error
			return vr, err
		}
		// Return fv
		fv.UpdateTimestamps()
		fv.ID, err = atx.Insert(&fv)
		if err != nil {
			return vr, err
		}
		// Update stats records
		if err := createFeedStats(atx, reader, fv.ID); err != nil {
			return vr, err
		}
		vr.FeedVersion = fv
		vr.Filename = reader.Path()
		vr.UploadFilename = fv.File
		return vr, nil
	}
	return ffetch(atx, feed, opts, cb)
}

func createFeedStats(atx tldb.Adapter, reader *tlcsv.Reader, fvid int) error {
	// Get FeedVersionFileInfos
	fvfis, err := dmfr.NewFeedVersionFileInfosFromReader(reader)
	if err != nil {
		return err
	}
	for _, fvfi := range fvfis {
		fvfi.UpdateTimestamps()
		fvfi.FeedVersionID = fvid
		if _, err := atx.Insert(&fvfi); err != nil {
			return err
		}
	}
	// Get service statistics
	fvsls, err := dmfr.NewFeedVersionServiceInfosFromReader(reader)
	if err != nil {
		return err
	}
	// Batch insert
	bt := make([]interface{}, len(fvsls))
	for i := range fvsls {
		fvsls[i].FeedVersionID = fvid
		bt[i] = &fvsls[i]
	}
	if err := atx.CopyInsert(bt); err != nil {
		return err
	}

	return nil
}
