package dmfr

import (
	"database/sql"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/gtdb"
)

// MainSync .
func MainSync(atx gtdb.Adapter, filenames []string) ([]string, error) {
	found := []string{}
	// Load
	regs := []*Registry{}
	for _, fn := range filenames {
		reg, err := LoadAndParseRegistry(fn)
		if err != nil {
			return found, err
		}
		regs = append(regs, reg)
	}
	// Import
	for _, registry := range regs {
		fids, err := ImportRegistry(atx, registry)
		if err != nil {
			return found, err
		}
		found = append(found, fids...)
	}
	// Hide
	if err := HideUnusedFeeds(atx, found); err != nil {
		return found, err
	}
	return found, nil
}

// ImportRegistry .
func ImportRegistry(atx gtdb.Adapter, registry *Registry) ([]string, error) {
	// Update feeds from DMFR
	feedids := []string{}
	for _, rfeed := range registry.Feeds {
		feedid, err := ImportFeed(atx, rfeed)
		if err != nil {
			return []string{}, err
		}
		feedids = append(feedids, feedid)
	}
	return feedids, nil
}

// ImportFeed .
func ImportFeed(atx gtdb.Adapter, rfeed Feed) (string, error) {
	// Check if we have the existing Feed
	var errTx error
	dbfeed := Feed{}
	err := atx.Get(&dbfeed, "SELECT * FROM current_feeds WHERE onestop_id = ?", rfeed.FeedID)
	if err == nil {
		// Exists, update key values
		rfeed.ID = dbfeed.ID
		rfeed.LastFetchedAt = dbfeed.LastFetchedAt
		rfeed.LastSuccessfulFetchAt = dbfeed.LastSuccessfulFetchAt
		rfeed.LastFetchError = dbfeed.LastFetchError
		rfeed.LastImportedAt = dbfeed.LastImportedAt
		rfeed.CreatedAt = dbfeed.CreatedAt
		errTx = atx.Update(&rfeed)
	} else if err == sql.ErrNoRows {
		_, errTx = atx.Insert(&rfeed)
	} else {
		// Error
		errTx = err
	}
	if errTx != nil {
		return "", errTx
	}
	return rfeed.FeedID, nil
}

// HideUnusedFeeds .
func HideUnusedFeeds(atx gtdb.Adapter, found []string) error {
	// Delete unreferenced feeds
	t := gotransit.OptionalTime{Time: time.Now(), Valid: true}
	_, err := atx.Sqrl().
		Update("current_feeds").
		Where(sq.NotEq{"onestop_id": found}).
		Set("deleted_at", t).
		Exec()
	return err
}
