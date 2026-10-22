package sync

// The rules that match operators to the agencies of their feeds, as pure
// functions over the rows, operators and agencies a store holds. Database sync
// is one store that applies their plans; any other store that keeps
// operator-in-feed rows can apply them the same way and get the same rows.

import (
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// FeedAgency is an agency of a feed's active version, with the onestop id its
// feed version's stats gave it, if any.
type FeedAgency struct {
	OnestopID tt.String
	gtfs.Agency
}

// FeedAgencies is a feed's id and the agencies of its active version.
type FeedAgencies struct {
	FeedID   int
	Agencies []FeedAgency
}

// OifRow is an operator-in-feed row a plan writes, and the row id of the agency
// it resolved to, or 0, which a store with agency places uses to fill in the
// row's resolved places.
type OifRow struct {
	dmfr.OperatorAssociatedFeed
	AgencyID int
}

// OifPlan is how sync changes a store's operator-in-feed rows. Inserts are new
// rows, in the order sync resolves them, so stores that number rows in insert
// order agree on their ids. Refreshes set the resolved fields of the rows their
// ids name, and Deletes are the ids of rows to remove.
type OifPlan struct {
	Inserts   []OifRow
	Refreshes []OifRow
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
		agencyID := 0
		if len(feed.Agencies) == 1 {
			// The only agency, whatever gtfs_agency_id the association names.
			oif.ResolvedGtfsAgencyID.Set(feed.Agencies[0].AgencyID.Val)
			agencyID = feed.Agencies[0].ID
		} else {
			for _, agency := range feed.Agencies {
				if agency.AgencyID.Val == oif.GtfsAgencyID.Val {
					oif.ResolvedGtfsAgencyID.Set(agency.AgencyID.Val)
					agencyID = agency.ID
				}
			}
		}
		// A matched row is refreshed, not kept as first written: the operator may
		// have a new name, and a row written before the feed had a version has no
		// agency_id, which matches ''.
		if id, ok := lookup[matchOf(oif)]; ok {
			matched[id] = true
			oif.ID = id
			plan.Refreshes = append(plan.Refreshes, OifRow{OperatorAssociatedFeed: oif, AgencyID: agencyID})
		} else {
			plan.Inserts = append(plan.Inserts, OifRow{OperatorAssociatedFeed: oif, AgencyID: agencyID})
		}
	}
	plan.Deletes = unmatched(existing, matched)
	return plan
}

// PlanFeedOifs generates an operator for each agency of a feed's active version
// that no row claims, and deletes the generated rows of agencies the version no
// longer has.
//
// existing holds the feed's current rows. An Atlas operator's row owns its
// agency and is never deleted here, so a generated row for the same agency goes
// unmatched and is deleted. A generated operator takes the agency's onestop id,
// or else one made from the feed's onestop id and the agency's name.
func PlanFeedOifs(feedID int, feedOnestopID string, existing []dmfr.OperatorAssociatedFeed, agencies []FeedAgency) OifPlan {
	lookup := map[oifmatch]int{}
	matched := map[int]bool{}
	for _, oif := range existing {
		key := matchOf(oif)
		if oif.OperatorID.Valid {
			matched[oif.ID] = true
			lookup[key] = oif.ID
		} else if _, ok := lookup[key]; !ok {
			lookup[key] = oif.ID
		}
	}
	var plan OifPlan
	for _, agency := range agencies {
		if id, ok := lookup[oifmatch{feedID: feedID, resolvedGtfsAgencyID: agency.AgencyID.Val}]; ok {
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
		plan.Inserts = append(plan.Inserts, OifRow{OperatorAssociatedFeed: oif, AgencyID: agency.ID})
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
