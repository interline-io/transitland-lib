package sync

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
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

// refreshOifSQL rewrites a matched operator-in-feed row's resolved fields where
// any changed. Each value is bound twice, to set it and to compare it, so an
// unchanged row is left alone.
const refreshOifSQL = `update current_operators_in_feed set
	resolved_onestop_id = ?,
	resolved_name = ?,
	resolved_short_name = ?,
	resolved_places = ?,
	resolved_gtfs_agency_id = ?
where id = ? and (
	resolved_onestop_id is distinct from ?
	or resolved_name is distinct from ?
	or resolved_short_name is distinct from ?
	or resolved_places is distinct from ?
	or resolved_gtfs_agency_id is distinct from ?
)`

// deleteHiddenOifsSQL removes the rows of operators that sync has soft-deleted.
const deleteHiddenOifsSQL = `delete from current_operators_in_feed
where operator_id in (select id from current_operators where deleted_at is not null)`

var nameTilde = "[-:&@/]"
var nameFilter = "[^[:alnum:]~><]"

// filterName .
func filterName(name string) string {
	re1 := regexp.MustCompile(nameTilde)
	re2 := regexp.MustCompile(nameFilter)
	return strings.ToLower(re2.ReplaceAllString(re1.ReplaceAllString(name, "~"), ""))
}

func getPlaces(ctx context.Context, atx tldb.Adapter, id int) (string, error) {
	agencyPlaces := []agencyPlace{}
	if err := atx.Select(ctx, &agencyPlaces, "select name,adm0name,adm1name from tl_agency_places where agency_id = ? AND rank > 0.2 order by rank desc, id", id); err != nil {
		return "", err
	}
	// Places stay in rank order, each listed once, so the same rows always give
	// the same string.
	places := []string{}
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
		place := strings.Join(suba, ", ")
		if len(suba) > 0 && !slices.Contains(places, place) {
			places = append(places, place)
		}
	}
	return strings.Join(places, " / "), nil
}

func updateOifs(ctx context.Context, atx tldb.Adapter, operator dmfr.Operator) (bool, error) {
	// Update OIFs that belong to this operator
	updated := false
	oiflookup := map[oifmatch]int{}
	oifmatches := map[int]bool{}
	oifexisting := []dmfr.OperatorAssociatedFeed{}
	if err := atx.Select(ctx, &oifexisting, "select * from current_operators_in_feed where operator_id = ?", operator.ID); err != nil {
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
		if err := atx.Get(ctx, &oif.FeedID, "select id from current_feeds where onestop_id = ?", oif.FeedOnestopID.Val); err == sql.ErrNoRows {
			log.For(ctx).Info().Msgf("Warning: no feed for '%s'", oif.FeedOnestopID.Val)
			continue
		} else if err != nil {
			return false, err
		}
		// Get agencies
		agencies := []gtfs.Agency{}
		if err := atx.Select(ctx, &agencies, "select gtfs_agencies.* from gtfs_agencies inner join feed_states on feed_states.materialized_feed_version_id = gtfs_agencies.feed_version_id where feed_states.feed_id = ?", oif.FeedID); err != nil {
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
		places, err := getPlaces(ctx, atx, agencyID)
		if err != nil {
			return false, err
		}
		oif.ResolvedPlaces.Set(places)
		// Match or insert. A matched row is refreshed, not kept as first written:
		// the operator may have a new name or the agency new places, and a NULL
		// agency id, written before the feed had a version, matches ''.
		check := oifmatch{feedID: oif.FeedID, resolvedGtfsAgencyID: oif.ResolvedGtfsAgencyID.Val}
		if match, ok := oiflookup[check]; ok {
			oifmatches[match] = true
			changed, err := refreshOif(ctx, atx, match, oif)
			if err != nil {
				return false, err
			}
			updated = updated || changed
		} else {
			updated = true
			if _, err := atx.Insert(ctx, &oif); err != nil {
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
		if _, err := atx.Sqrl().Delete("current_operators_in_feed").Where(sq.Eq{"id": deleteoifs}).ExecContext(ctx); err != nil {
			return false, err
		}
	}
	return updated, nil
}

// refreshOif writes oif's resolved fields to row id where they differ from what
// the row holds, and says whether any did.
func refreshOif(ctx context.Context, atx tldb.Adapter, id int, oif dmfr.OperatorAssociatedFeed) (bool, error) {
	vals := []any{oif.ResolvedOnestopID, oif.ResolvedName, oif.ResolvedShortName, oif.ResolvedPlaces, oif.ResolvedGtfsAgencyID}
	r, err := atx.DBX().ExecContext(ctx, atx.DBX().Rebind(refreshOifSQL), slices.Concat(vals, []any{id}, vals)...)
	if err != nil {
		return false, err
	}
	n, err := r.RowsAffected()
	return n > 0, err
}

// deleteHiddenOifs removes the rows of soft-deleted operators, which the finders
// already hide, so none of them keeps an agency from its generated row.
func deleteHiddenOifs(ctx context.Context, atx tldb.Adapter) error {
	_, err := atx.DBX().ExecContext(ctx, atx.DBX().Rebind(deleteHiddenOifsSQL))
	return err
}

func feedUpdateOifs(ctx context.Context, atx tldb.Adapter, feed dmfr.Feed) (bool, error) {
	// Update OIFs that do not have an operator
	updated := false
	feedid := feed.ID
	oiflookup := map[oifmatch]int{}
	oifmatches := map[int]bool{}
	oifexisting := []dmfr.OperatorAssociatedFeed{}
	if err := atx.Select(ctx, &oifexisting, "select * from current_operators_in_feed where feed_id = ?", feedid); err != nil {
		return false, err
	}
	for _, oif := range oifexisting {
		key := oifmatch{feedID: oif.FeedID, resolvedGtfsAgencyID: oif.ResolvedGtfsAgencyID.Val}
		if oif.OperatorID.Valid {
			// An Atlas row is never deleted here, and owns its key, so a generated row
			// for the same agency goes unmatched and is deleted below.
			oifmatches[oif.ID] = true
			oiflookup[key] = oif.ID
		} else if _, ok := oiflookup[key]; !ok {
			oiflookup[key] = oif.ID
		}
	}
	agencies := []agencyOnestop{}
	agencyQuery := atx.Sqrl().
		Select("gtfs_agencies.*", "feed_version_agency_onestop_ids.onestop_id as onestop_id").
		From("gtfs_agencies").
		Join("feed_states on feed_states.materialized_feed_version_id = gtfs_agencies.feed_version_id").
		Join("current_feeds on current_feeds.id = feed_states.feed_id").
		JoinClause("left join feed_version_agency_onestop_ids on feed_version_agency_onestop_ids.entity_id = gtfs_agencies.agency_id and feed_version_agency_onestop_ids.feed_version_id = gtfs_agencies.feed_version_id").
		Where(sq.Eq{"current_feeds.id": feedid})
	qstr, qargs, err := agencyQuery.ToSql()
	if err != nil {
		return false, err
	}
	if err := atx.Select(ctx, &agencies, qstr, qargs...); err != nil {
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
			if places, err := getPlaces(ctx, atx, agency.ID); err != nil {
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
			if _, err := atx.Insert(ctx, &oif); err != nil {
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
		if _, err := atx.Sqrl().Delete("current_operators_in_feed").Where(sq.Eq{"id": deleteoifs}).ExecContext(ctx); err != nil {
			return false, err
		}
	}
	return updated, nil
}
