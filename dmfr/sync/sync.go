package sync

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
)

// Options sets options for a sync operation.
type Options struct {
	Filenames           []string
	HideUnseen          bool
	HideUnseenOperators bool
}

// Result is the result of a sync operation.
type Result struct {
	FeedIDs         []int
	OperatorIDs     []int
	Errors          []error
	HiddenCount     int
	HiddenOperators int
}

// MainSync .
func MainSync(atx tldb.Adapter, opts Options) (Result, error) {
	// Load
	sr := Result{}
	feedids := []int{}
	operatorids := []int{}
	errs := []error{}
	for _, fn := range opts.Filenames {
		reg, err := dmfr.LoadAndParseRegistry(fn)
		if err != nil {
			log.Error("%s: Error parsing DMFR: %s", fn, err.Error())
			errs = append(errs, err)
			continue
		}
		for _, rfeed := range reg.Feeds {
			fsid := rfeed.FeedID
			rfeed.File = filepath.Base(fn)
			feedid, found, err := UpdateFeed(atx, rfeed)
			if found {
				log.Info("%s: updated feed %s (id:%d)", fn, fsid, feedid)
			} else {
				log.Info("%s: new feed %s (id:%d)", fn, fsid, feedid)
			}
			if err != nil {
				log.Error("%s: error on feed %s: %s", fn, feedid, err)
				errs = append(errs, err)
			}
			feedids = append(feedids, feedid)
		}
		for _, operator := range reg.Operators {
			osid := operator.OnestopID.String
			operatorid, found, err := UpdateOperator(atx, operator)
			if found {
				log.Info("%s: updated operator %s (id:%d)", fn, osid, operatorid)
			} else {
				log.Info("%s: new operator %s (id:%d)", fn, osid, operatorid)
			}
			if err != nil {
				log.Error("%s: error on feed %s: %s", fn, osid, err)
				errs = append(errs, err)
			}
			operatorids = append(operatorids, operatorid)
		}
	}
	sr.FeedIDs = feedids
	sr.OperatorIDs = operatorids
	sr.Errors = errs
	if len(errs) > 0 {
		log.Error("Rollback due to one or more failures")
		return sr, fmt.Errorf("failed: %s", errs[0].Error())
	}
	// Hide
	if opts.HideUnseen {
		var err error
		sr.HiddenCount, err = HideUnseedFeeds(atx, sr.FeedIDs)
		if err != nil {
			log.Error("Error soft-deleting feeds: %s", err.Error())
			return sr, err
		}
		if sr.HiddenCount > 0 {
			log.Info("Soft-deleted %d feeds", sr.HiddenCount)
		}
	}
	if opts.HideUnseenOperators {
		var err error
		sr.HiddenOperators, err = HideUnseedOperators(atx, sr.OperatorIDs)
		if err != nil {
			log.Error("Error soft-deleting operators: %s", err.Error())
			return sr, err
		}
		if sr.HiddenOperators > 0 {
			log.Info("Soft-deleted %d operators", sr.HiddenOperators)
		}
	}
	return sr, nil
}

// UpdateFeed .
func UpdateFeed(atx tldb.Adapter, rfeed tl.Feed) (int, bool, error) {
	// Check if we have the existing Feed
	feedid := 0
	found := false
	var errTx error
	dbfeed := tl.Feed{}
	err := atx.Get(&dbfeed, "SELECT * FROM current_feeds WHERE onestop_id = ?", rfeed.FeedID)
	if err == nil {
		// Exists, update key values
		found = true
		feedid = dbfeed.ID
		rfeed.ID = dbfeed.ID
		rfeed.CreatedAt = dbfeed.CreatedAt
		rfeed.DeletedAt = tl.OTime{Valid: false}
		rfeed.UpdateTimestamps()
		errTx = atx.Update(&rfeed)
	} else if err == sql.ErrNoRows {
		rfeed.UpdateTimestamps()
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
func HideUnseedFeeds(atx tldb.Adapter, found []int) (int, error) {
	// Delete unreferenced feeds
	t := tl.OTime{Time: time.Now(), Valid: true}
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

// UpdateOperator updates or inserts a single operator, as well as managing associated operator-in-feed records
func UpdateOperator(atx tldb.Adapter, operator tl.Operator) (int, bool, error) {
	// Check if we have the existing operator
	id := 0
	found := false
	var errTx error
	ent := tl.Operator{}
	err := atx.Get(&ent, "SELECT * FROM current_operators WHERE onestop_id = ?", operator.OnestopID)
	if err == nil {
		// Exists, update key values
		found = true
		id = ent.ID
		operator.ID = ent.ID
		operator.CreatedAt = ent.CreatedAt
		operator.DeletedAt = tl.OTime{Valid: false}
		operator.UpdateTimestamps()
		errTx = atx.Update(&operator)
	} else if err == sql.ErrNoRows {
		// Insert
		operator.UpdateTimestamps()
		id, errTx = atx.Insert(&operator)
		operator.ID = id
	} else {
		// Error
		errTx = err
	}
	if errTx != nil {
		return 0, false, errTx
	}
	// Update operator in feeds
	if err := updateOifs(atx, operator); err != nil {
		return 0, false, err
	}
	return id, found, nil
}

func updateOifs(atx tldb.Adapter, operator tl.Operator) error {
	id := operator.ID
	type oifmatch struct {
		feedid   int
		agencyid string
	}
	oiflookup := map[oifmatch]int{}
	oifmatches := map[int]bool{}
	oifexisting := []tl.OperatorAssociatedFeed{}
	fsids := map[string]int{} // cache for entire run?
	if err := atx.Select(&oifexisting, "select * from current_operators_in_feed where operator_id = ?", id); err != nil {
		return err
	}
	for _, oif := range oifexisting {
		oiflookup[oifmatch{feedid: oif.FeedID, agencyid: oif.AgencyID.String}] = oif.ID
	}
	for _, oif := range operator.AssociatedFeeds {
		fsid := oif.FeedOnestopID.String
		feedid, ok := fsids[fsid]
		if !ok {
			if err := atx.Get(&feedid, "select id from current_feeds where onestop_id = ?", fsid); err != nil {
				return err
			}
			fsids[fsid] = feedid
		}
		if match, ok := oiflookup[oifmatch{feedid: feedid, agencyid: oif.AgencyID.String}]; ok {
			// ok
			oifmatches[match] = true
		} else {
			if _, err := atx.Sqrl().Insert("current_operators_in_feed").Columns("operator_id", "feed_id", "gtfs_agency_id").Values(id, feedid, oif.AgencyID).Exec(); err != nil {
				return err
			}
		}
	}
	deleteoifs := []int{}
	for _, oif := range oifexisting {
		if _, ok := oifmatches[oif.ID]; !ok {
			deleteoifs = append(deleteoifs, oif.ID)
		}
	}
	if len(deleteoifs) > 0 {
		if _, err := atx.Sqrl().Delete("current_operators_in_feed").Where(sq.Eq{"id": deleteoifs}).Exec(); err != nil {
			return err
		}
	}
	return nil
}

// HideUnseedOperators .
func HideUnseedOperators(atx tldb.Adapter, found []int) (int, error) {
	// Delete unreferenced feeds
	t := tl.OTime{Time: time.Now(), Valid: true}
	r, err := atx.Sqrl().
		Update("current_operators").
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
