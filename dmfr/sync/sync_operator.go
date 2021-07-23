package sync

import (
	"database/sql"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
)

// UpdateOperator updates or inserts a single operator, as well as managing associated operator-in-feed records
func UpdateOperator(atx tldb.Adapter, operator tl.Operator) (int, bool, bool, error) {
	// Check if we have the existing operator
	found := false
	updated := false
	var errTx error
	ent := tl.Operator{}
	err := atx.Get(&ent, "SELECT * FROM current_operators WHERE onestop_id = ?", operator.OnestopID)
	if err == nil {
		// Exists, update key values
		found = true
		operator.ID = ent.ID
		if !ent.Equal(&operator) {
			updated = true
			operator.CreatedAt = ent.CreatedAt
			operator.DeletedAt = tl.OTime{Valid: false}
			operator.UpdateTimestamps()
			errTx = atx.Update(&operator)
		}
	} else if err == sql.ErrNoRows {
		// Insert
		operator.UpdateTimestamps()
		operator.ID, errTx = atx.Insert(&operator)
	} else {
		// Error
		errTx = err
	}
	if errTx != nil {
		return 0, false, false, errTx
	}
	// Update operator in feeds
	// This happens even if the entity did not change.
	oifUpdate, err := updateOifs(atx, operator)
	if err != nil {
		return 0, false, false, err
	}
	if oifUpdate {
		updated = true
	}
	return operator.ID, found, updated, nil
}

func updateOifs(atx tldb.Adapter, operator tl.Operator) (bool, error) {
	updated := false
	id := operator.ID
	type oifmatch struct {
		feedID       int
		agencyID     int
		gtfsAgencyID string
	}
	oiflookup := map[oifmatch]int{}
	oifmatches := map[int]bool{}
	oifexisting := []tl.OperatorAssociatedFeed{}
	if err := atx.Select(&oifexisting, "select * from current_operators_in_feed where operator_id = ?", id); err != nil {
		return false, err
	}
	for _, oif := range oifexisting {
		oiflookup[oifmatch{feedID: oif.FeedID, gtfsAgencyID: oif.GtfsAgencyID.String, agencyID: oif.AgencyID.Int}] = oif.ID
	}
	for _, oif := range operator.AssociatedFeeds {
		// Get feed id
		fsid := oif.FeedOnestopID.String
		feedid := 0
		if err := atx.Get(&feedid, "select id from current_feeds where onestop_id = ?", fsid); err == sql.ErrNoRows {
			log.Info("Warning: no feed for '%s'", fsid)
			continue
		} else if err != nil {
			return false, err
		}
		// Get agencies
		agencies := []tl.Agency{}
		if err := atx.Select(&agencies, "select gtfs_agencies.* from gtfs_agencies inner join feed_states using(feed_version_id) where feed_states.feed_id = ?", feedid); err != nil {
			return false, err
		}
		if len(agencies) == 1 {
			// match regardless of gtfs_agency_id
			oif.AgencyID = tl.NewOInt(agencies[0].ID)
		} else if len(agencies) > 1 {
			// match on first gtfs_agency_id
			for _, agency := range agencies {
				if agency.AgencyID == oif.GtfsAgencyID.String {
					oif.AgencyID = tl.NewOInt(agency.ID)
				}
			}
		}
		// Match or insert
		if match, ok := oiflookup[oifmatch{feedID: feedid, gtfsAgencyID: oif.GtfsAgencyID.String, agencyID: oif.AgencyID.Int}]; ok {
			// ok
			oifmatches[match] = true
		} else {
			updated = true
			if _, err := atx.Sqrl().Insert("current_operators_in_feed").Columns("operator_id", "feed_id", "gtfs_agency_id", "agency_id").Values(id, feedid, oif.GtfsAgencyID, oif.AgencyID).Exec(); err != nil {
				return false, err
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
		updated = true
		if _, err := atx.Sqrl().Delete("current_operators_in_feed").Where(sq.Eq{"id": deleteoifs}).Exec(); err != nil {
			return false, err
		}
	}
	return updated, nil
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
