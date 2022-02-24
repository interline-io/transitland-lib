package sync

import (
	"database/sql"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
)

// UpdateFeed .
func UpdateFeed(atx tldb.Adapter, rfeed tl.Feed) (int, bool, bool, error) {
	// Check if we have the existing Feed
	feedid := 0
	found := false
	updated := false
	var errTx error
	dbfeed := tl.Feed{}
	if err := atx.Get(&dbfeed, "select * from current_feeds where onestop_id = ?", rfeed.FeedID); err == nil {
		// Exists, update key values
		found = true
		feedid = dbfeed.ID
		rfeed.ID = dbfeed.ID
		if !dbfeed.Equal(&rfeed) {
			updated = true
			rfeed.CreatedAt = dbfeed.CreatedAt
			rfeed.DeletedAt = tl.Time{Valid: false}
			rfeed.UpdateTimestamps()
			errTx = atx.Update(&rfeed)
		}
	} else if err == sql.ErrNoRows {
		rfeed.UpdateTimestamps()
		feedid, errTx = atx.Insert(&rfeed)
	} else {
		// Error
		errTx = err
	}
	return feedid, found, updated, errTx
}

// HideUnseedFeeds .
func HideUnseedFeeds(atx tldb.Adapter, found []int) (int, error) {
	// Delete unreferenced feeds
	t := tl.NewTime(time.Now())
	r, err := atx.Sqrl().
		Update("current_feeds").
		Where(sq.NotEq{"id": found}).
		Where(sq.Eq{"deleted_at": nil}).
		Set("deleted_at", t).
		Exec()
	if err != nil {
		return 0, err
	}
	c, err := r.RowsAffected()
	return int(c), err
}

// UpdateFeedGeneratedOperators creates OperatorInFeed records for agencies that are not associated with an operator
func UpdateFeedGeneratedOperators(atx tldb.Adapter, found []int) error {
	for _, id := range found {
		feed := tl.Feed{}
		if err := atx.Get(&feed, "select * from current_feeds where id = ?", id); err != nil {
			return err
		}
		if _, err := feedUpdateOifs(atx, feed); err != nil {
			return err
		}
	}
	return nil
}
