package dmfr

import (
	"database/sql"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/log"
)

// MainSync .
func MainSync(atx gtdb.Adapter, filenames []string) ([]int, error) {
	// Load
	regs := []*Registry{}
	for _, fn := range filenames {
		log.Info("Loading DMFR: %s", fn)
		reg, err := LoadAndParseRegistry(fn)
		if err != nil {
			return []int{}, err
		}
		// TODO: Validate
		regs = append(regs, reg)
	}
	// Import
	feedids := []int{}
	errs := []error{}
	for _, registry := range regs {
		fids, ferrs := MainImportRegistry(atx, registry)
		feedids = append(feedids, fids...)
		errs = append(errs, ferrs...)
	}
	if len(errs) > 0 {
		log.Info("Rollback due to one or more failures")
		return []int{}, errors.New("failed")
	}
	// Hide
	count, err := HideUnseedFeeds(atx, feedids)
	if err != nil {
		return []int{}, err
	}
	if count > 0 {
		log.Info("Soft-deleted %d feeds", count)
	}
	return feedids, nil
}

// MainImportRegistry .
func MainImportRegistry(atx gtdb.Adapter, registry *Registry) ([]int, []error) {
	errs := []error{}
	feedids := []int{}
	log.Info("Syncing %d feeds in registry", len(registry.Feeds))
	for _, rfeed := range registry.Feeds {
		feedid, found, err := ImportFeed(atx, rfeed)
		if found {
			log.Info("\t%s: updated id = %d", rfeed.FeedID, feedid)
		} else {
			log.Info("\t%s: new id = %d", rfeed.FeedID, feedid)
		}
		if err != nil {
			log.Info("\t%s: error: %s", feedid, err)
			errs = append(errs, err)
		}
		feedids = append(feedids, feedid)
	}
	return feedids, errs
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
		feedid = dbfeed.ID
		found = true
		rfeed.ID = dbfeed.ID
		rfeed.LastFetchedAt = dbfeed.LastFetchedAt
		rfeed.LastSuccessfulFetchAt = dbfeed.LastSuccessfulFetchAt
		rfeed.LastFetchError = dbfeed.LastFetchError
		rfeed.LastImportedAt = dbfeed.LastImportedAt
		rfeed.CreatedAt = dbfeed.CreatedAt
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
