package sync

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
)

type oifmatch struct {
	feedID               int
	resolvedGtfsAgencyID string
}

type agencyOnestop struct {
	OnestopID tt.String
	gtfs.Agency
}

type agencyPlace struct {
	Name     tt.String
	Adm1name tt.String
	Adm0name tt.String
}

var nameTilde = "[-:&@/]"
var nameFilter = "[^[:alnum:]~><]"

// filterName .
func filterName(name string) string {
	re1 := regexp.MustCompile(nameTilde)
	re2 := regexp.MustCompile(nameFilter)
	return strings.ToLower(re2.ReplaceAllString(re1.ReplaceAllString(name, "~"), ""))
}

func getPlaces(atx tldb.Adapter, id int) (string, error) {
	agencyPlaces := []agencyPlace{}
	if err := atx.Select(&agencyPlaces, "select name,adm0name,adm1name from tl_agency_places where agency_id = ? AND rank > 0.2 order by rank desc", id); err != nil {
		return "", err
	}
	uniquePlaces := map[string]bool{}
	for _, a := range agencyPlaces {
		suba := []string{}
		if a.Name.Valid {
			suba = append(suba, a.Name.Val)
		}
		if a.Adm1name.Valid {
			suba = append(suba, a.Adm1name.Val)
		}
		if a.Adm0name.Valid {
			suba = append(suba, a.Adm0name.Val)
		}
		if len(suba) > 0 {
			uniquePlaces[strings.Join(suba, ", ")] = true
		}
	}
	places := []string{}
	for k := range uniquePlaces {
		places = append(places, k)
	}
	return strings.Join(places, " / "), nil
}

func updateOifs(atx tldb.Adapter, operator dmfr.Operator) (bool, error) {
	// Update OIFs that belong to this operator
	updated := false
	oiflookup := map[oifmatch]int{}
	oifmatches := map[int]bool{}
	oifexisting := []dmfr.OperatorAssociatedFeed{}
	if err := atx.Select(&oifexisting, "select * from current_operators_in_feed where operator_id = ?", operator.ID); err != nil {
		return false, err
	}
	for _, oif := range oifexisting {
		oiflookup[oifmatch{feedID: oif.FeedID, resolvedGtfsAgencyID: oif.ResolvedGtfsAgencyID.Val}] = oif.ID
	}
	for _, oif := range operator.AssociatedFeeds {
		// Get feed id
		oif.ResolvedOnestopID.Set(operator.OnestopID.Val)
		oif.ResolvedName.Set(operator.Name.Val)
		oif.ResolvedShortName.Set(operator.ShortName.Val)
		oif.OperatorID.SetInt(operator.ID)
		if err := atx.Get(&oif.FeedID, "select id from current_feeds where onestop_id = ?", oif.FeedOnestopID.Val); err == sql.ErrNoRows {
			log.Infof("Warning: no feed for '%s'", oif.FeedOnestopID.Val)
			continue
		} else if err != nil {
			return false, err
		}
		// Get agencies
		agencies := []gtfs.Agency{}
		if err := atx.Select(&agencies, "select gtfs_agencies.* from gtfs_agencies inner join feed_states using(feed_version_id) where feed_states.feed_id = ?", oif.FeedID); err != nil {
			return false, err
		}
		agencyID := 0
		if len(agencies) == 1 {
			// match regardless of gtfs_agency_id
			oif.ResolvedGtfsAgencyID.Set(agencies[0].AgencyID.Val)
			agencyID = agencies[0].ID
		} else if len(agencies) > 1 {
			// match on gtfs_agency_id
			for _, agency := range agencies {
				if agency.AgencyID.Val == oif.GtfsAgencyID.Val {
					oif.ResolvedGtfsAgencyID.Set(agency.AgencyID.Val)
					agencyID = agency.ID
				}
			}
		}
		// Match or insert
		check := oifmatch{feedID: oif.FeedID, resolvedGtfsAgencyID: oif.ResolvedGtfsAgencyID.Val}
		if match, ok := oiflookup[check]; ok {
			oifmatches[match] = true
		} else {
			updated = true
			if places, err := getPlaces(atx, agencyID); err != nil {
				return false, err
			} else {
				oif.ResolvedPlaces.Set(places)
			}
			if _, err := atx.Insert(&oif); err != nil {
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

func feedUpdateOifs(atx tldb.Adapter, feed dmfr.Feed) (bool, error) {
	// Update OIFs that do not have an operator
	updated := false
	feedid := feed.ID
	oiflookup := map[oifmatch]int{}
	oifmatches := map[int]bool{}
	oifexisting := []dmfr.OperatorAssociatedFeed{}
	if err := atx.Select(&oifexisting, "select * from current_operators_in_feed where feed_id = ?", feedid); err != nil {
		return false, err
	}
	for _, oif := range oifexisting {
		oiflookup[oifmatch{feedID: oif.FeedID, resolvedGtfsAgencyID: oif.ResolvedGtfsAgencyID.Val}] = oif.ID
		if oif.OperatorID.Valid {
			oifmatches[oif.ID] = true // allow matching on operator associated oifs, but do not delete them
		}
	}
	agencies := []agencyOnestop{}
	agencyQuery := atx.Sqrl().
		Select("gtfs_agencies.*", "feed_version_agency_onestop_ids.onestop_id as onestop_id").
		From("gtfs_agencies").
		Join("feed_states using(feed_version_id)").
		Join("current_feeds on current_feeds.id = feed_states.feed_id").
		JoinClause("left join feed_version_agency_onestop_ids on feed_version_agency_onestop_ids.entity_id = gtfs_agencies.agency_id and feed_version_agency_onestop_ids.feed_version_id = gtfs_agencies.feed_version_id").
		Where(sq.Eq{"current_feeds.id": feedid})
	qstr, qargs, err := agencyQuery.ToSql()
	if err != nil {
		return false, err
	}
	if err := atx.Select(&agencies, qstr, qargs...); err != nil {
		return false, err
	}
	for _, agency := range agencies {
		check := oifmatch{feedID: feedid, resolvedGtfsAgencyID: agency.AgencyID.Val}
		if match, ok := oiflookup[check]; ok {
			oifmatches[match] = true
		} else {
			updated = true
			// Generate OnestopID
			oif := dmfr.OperatorAssociatedFeed{
				FeedID:               feedid,
				ResolvedGtfsAgencyID: tt.NewString(agency.AgencyID.Val),
				ResolvedName:         tt.NewString(agency.AgencyName.Val),
			}
			if places, err := getPlaces(atx, agency.ID); err != nil {
				return false, err
			} else {
				oif.ResolvedPlaces.Set(places)
			}
			if agency.OnestopID.Valid {
				oif.ResolvedOnestopID = agency.OnestopID
			} else {
				fsid := "unknown"
				if strings.HasPrefix(feed.FeedID, "f-") && len(feed.FeedID) > 2 {
					fsid = feed.FeedID[2:]
				}
				oif.ResolvedOnestopID.Set(fmt.Sprintf("o-%s-%s", fsid, filterName(agency.AgencyName.Val)))
			}
			// Save
			if _, err := atx.Insert(&oif); err != nil {
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
