package sync

// The rules sync uses to match operators to the agencies of their feeds, as pure
// functions over operator-in-feed rows, operators and agencies.

import (
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// FeedAgency is an agency of a feed's active version, and the onestop id from
// its feed version's stats, when there is one.
type FeedAgency struct {
	OnestopID tt.String
	gtfs.Agency
}

// FeedAgencies is a feed's id and the agencies of its active version.
type FeedAgencies struct {
	FeedID   int
	Agencies []FeedAgency
}

// OifRow is an operator-in-feed row a plan writes. AgencyRowID is the row id of
// its resolved agency, or 0 when it resolved none.
type OifRow struct {
	dmfr.OperatorAssociatedFeed
	AgencyRowID int
}

// OifPlan is how sync changes operator-in-feed rows: refreshes, then inserts,
// then deletes.
//
// Refreshes set the resolved fields of the rows their ids name. Inserts are new
// rows, in the order sync resolves them. Deletes are the ids of rows to remove.
type OifPlan struct {
	Refreshes []OifRow
	Inserts   []OifRow
	Deletes   []int
}

type oifmatch struct {
	feedID               int
	resolvedGtfsAgencyID string
}

func matchOf(oif dmfr.OperatorAssociatedFeed) oifmatch {
	return oifmatch{feedID: oif.FeedID, resolvedGtfsAgencyID: oif.ResolvedGtfsAgencyID.Val}
}

// PlanOperatorOifs resolves an operator to the agency of each of its associated
// feeds' active versions: the feed's only agency, or the one the association's
// gtfs_agency_id names.
//
// existing holds the operator's current rows, and feeds the associated feeds by
// onestop id; a feed missing from feeds is skipped. Each resolved row that
// matches an existing row, by feed and resolved agency_id, refreshes that row,
// and each other one is inserted. Existing rows nothing matches are deleted.
func PlanOperatorOifs(operator dmfr.Operator, existing []dmfr.OperatorAssociatedFeed, feeds map[string]FeedAgencies) OifPlan {
	lookup := map[oifmatch]int{}
	for _, oif := range existing {
		lookup[matchOf(oif)] = oif.ID
	}
	matched := map[int]bool{}
	var plan OifPlan
	for _, oif := range operator.AssociatedFeeds {
		feed, ok := feeds[oif.FeedOnestopID.Val]
		if !ok {
			continue
		}
		oif.ResolvedOnestopID.Set(operator.OnestopID.Val)
		oif.ResolvedName.Set(operator.Name.Val)
		oif.ResolvedShortName.Set(operator.ShortName.Val)
		oif.OperatorID.SetInt(operator.ID)
		oif.FeedID = feed.FeedID
		agencyRowID := 0
		if len(feed.Agencies) == 1 {
			// The only agency, whatever gtfs_agency_id the association names.
			oif.ResolvedGtfsAgencyID.Set(feed.Agencies[0].AgencyID.Val)
			agencyRowID = feed.Agencies[0].ID
		} else {
			for _, agency := range feed.Agencies {
				if agency.AgencyID.Val == oif.GtfsAgencyID.Val {
					oif.ResolvedGtfsAgencyID.Set(agency.AgencyID.Val)
					agencyRowID = agency.ID
				}
			}
		}
		// A matched row is refreshed, not kept as first written: the operator may
		// have a new name, and a row written before the feed had a version has no
		// agency_id, which matches ''.
		if id, ok := lookup[matchOf(oif)]; ok {
			matched[id] = true
			oif.ID = id
			plan.Refreshes = append(plan.Refreshes, OifRow{OperatorAssociatedFeed: oif, AgencyRowID: agencyRowID})
		} else {
			plan.Inserts = append(plan.Inserts, OifRow{OperatorAssociatedFeed: oif, AgencyRowID: agencyRowID})
		}
	}
	plan.Deletes = unmatched(existing, matched)
	return plan
}

// PlanFeedOifs generates an operator for each agency of a feed's active version
// that no row claims, and deletes the generated rows of agencies the version no
// longer has.
//
// existing holds the feed's current rows, matched to agencies by resolved
// agency_id. An Atlas operator's row owns its agency and is never deleted here,
// so a generated row for the same agency goes unmatched and is deleted. A
// generated operator takes the agency's onestop id, or else one made from the
// feed's onestop id and the agency's name.
func PlanFeedOifs(feedID int, feedOnestopID string, existing []dmfr.OperatorAssociatedFeed, agencies []FeedAgency) OifPlan {
	lookup := map[string]int{} // resolved agency_id -> row id
	matched := map[int]bool{}
	for _, oif := range existing {
		key := oif.ResolvedGtfsAgencyID.Val
		if oif.OperatorID.Valid {
			matched[oif.ID] = true
			lookup[key] = oif.ID
		} else if _, ok := lookup[key]; !ok {
			lookup[key] = oif.ID
		}
	}
	var plan OifPlan
	for _, agency := range agencies {
		if id, ok := lookup[agency.AgencyID.Val]; ok {
			matched[id] = true
			continue
		}
		oif := dmfr.OperatorAssociatedFeed{
			FeedID:               feedID,
			ResolvedGtfsAgencyID: tt.NewString(agency.AgencyID.Val),
			ResolvedName:         tt.NewString(agency.AgencyName.Val),
		}
		if agency.OnestopID.Valid {
			oif.ResolvedOnestopID = agency.OnestopID
		} else {
			oif.ResolvedOnestopID.Set(GeneratedOperatorOnestopID(feedOnestopID, agency.AgencyName.Val))
		}
		plan.Inserts = append(plan.Inserts, OifRow{OperatorAssociatedFeed: oif, AgencyRowID: agency.ID})
	}
	plan.Deletes = unmatched(existing, matched)
	return plan
}

// unmatched returns the ids of the existing rows not in matched, in order.
func unmatched(existing []dmfr.OperatorAssociatedFeed, matched map[int]bool) []int {
	var ids []int
	for _, oif := range existing {
		if !matched[oif.ID] {
			ids = append(ids, oif.ID)
		}
	}
	return ids
}
