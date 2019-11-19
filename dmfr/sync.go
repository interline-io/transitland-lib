package dmfr

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/log"
)

// SyncOptions sets options for a sync operation.
type SyncOptions struct {
	Filenames  []string
	HideUnseen bool
}

// SyncResult is the result of a sync operation.
type SyncResult struct {
	FeedIDs     []int
	Errors      []error
	HiddenCount int
}

// MainSync .
func MainSync(atx gtdb.Adapter, opts SyncOptions) (SyncResult, error) {
	// Load
	sr := SyncResult{}
	feedids := []int{}
	errs := []error{}
	for _, fn := range opts.Filenames {
		reg, err := LoadAndParseRegistry(fn)
		if err != nil {
			log.Info("%s: Error parsing DMFR: %s", fn, err.Error())
			errs = append(errs, err)
			continue
		}
		for _, rfeed := range reg.Feeds {
			rfeed.File = filepath.Base(fn)
			feedid, found, err := ImportFeed(atx, rfeed)
			if found {
				log.Info("%s: updated feed %s (id:%d)", fn, rfeed.FeedID, feedid)
			} else {
				log.Info("%s: new feed %s (id:%d)", fn, rfeed.FeedID, feedid)
			}
			if err != nil {
				log.Info("%s: error on feed %s: %s", fn, feedid, err)
				errs = append(errs, err)
			}
			feedids = append(feedids, feedid)
		}
	}
	sr.FeedIDs = feedids
	sr.Errors = errs
	if len(errs) > 0 {
		log.Info("Rollback due to one or more failures")
		return sr, fmt.Errorf("Failed: %s", errs[0].Error())
	}
	// Hide
	if opts.HideUnseen {
		count, err := HideUnseedFeeds(atx, sr.FeedIDs)
		if err != nil {
			log.Info("Error soft-deleting feeds: %s", err.Error())
			return sr, err
		}
		sr.HiddenCount = count
		if count > 0 {
			log.Info("Soft-deleted %d feeds", count)
		}
	}
	return sr, nil
}

// ImportFeed .
func ImportFeed(atx gtdb.Adapter, rfeed Feed) (int, bool, error) {
	// Check if we have the existing Feed
	feedid := 0
	found := false
	var errTx error
	dbfeed := Feed{}
	err := atx.Get(&dbfeed, "SELECT * FROM current_feeds WHERE onestop_id = ?", rfeed.FeedID)
	if err == nil {
		// Exists, update key values
		found = true
		feedid = dbfeed.ID
		rfeed.ID = dbfeed.ID
		rfeed.CreatedAt = dbfeed.CreatedAt
		rfeed.DeletedAt = gotransit.OptionalTime{Valid: false}
		errTx = atx.Update(&rfeed)
	} else if err == sql.ErrNoRows {
		feedid, errTx = atx.Insert(&rfeed)
	} else {
		// Error
		errTx = err
	}
	if errTx != nil {
		return 0, found, errTx
	}
	return feedid, found, nil
}

// HideUnseedFeeds .
func HideUnseedFeeds(atx gtdb.Adapter, found []int) (int, error) {
	// Delete unreferenced feeds
	t := gotransit.OptionalTime{Time: time.Now(), Valid: true}
	r, err := atx.Sqrl().
		Update("current_feeds").
		Where(sq.NotEq{"id": found}).
		Set("deleted_at", t).
		Exec()
	if err != nil {
		return 0, err
	}
	c, err := r.RowsAffected()
	return int(c), err
}
