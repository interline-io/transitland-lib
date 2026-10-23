package sync

// The rules sync uses to match operators to the agencies of their feeds, as pure
// functions over operator-in-feed rows, operators and agencies.

import (
	"cmp"
	"slices"
	"strings"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/ext/builders"
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
// onestop id; a feed missing from feeds is skipped. Each resolved row refreshes
// the existing row it matches, by feed and resolved agency_id, or else is
// inserted. Associations that resolve to the same match make one row, and of
// existing rows that share a match, only the lowest id can match. The plan
// deletes every existing row left unmatched.
func PlanOperatorOifs(operator dmfr.Operator, existing []dmfr.OperatorAssociatedFeed, feeds map[string]FeedAgencies) OifPlan {
	existing = byID(existing)
	lookup := map[oifmatch]int{}
	for _, oif := range existing {
		key := matchOf(oif)
		if _, ok := lookup[key]; !ok {
			lookup[key] = oif.ID
		}
	}
	matched := map[int]bool{}
	planned := map[oifmatch]bool{}
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
		key := matchOf(oif)
		if planned[key] {
			// A repeated association: the first one's row stands.
			continue
		}
		planned[key] = true
		// A matched row is refreshed, not kept as first written: the operator may
		// have a new name, and a row written before the feed had a version has no
		// agency_id, which matches ''.
		if id, ok := lookup[key]; ok {
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
// agency_id, and deletedOperators the ids of soft-deleted operators. An Atlas
// operator's row owns its agency and is never deleted here, unless its operator
// is deleted: then the row is deleted and claims no agency. A generated row for
// an agency an Atlas row owns is deleted, and of generated rows that share an
// agency, the lowest id is kept. Agencies that share an agency_id get one
// generated row, and inserts follow the agencies' row ids. A generated
// operator takes the agency's onestop id, or else one made from the feed's
// onestop id and the agency's name.
func PlanFeedOifs(feedID int, feedOnestopID string, existing []dmfr.OperatorAssociatedFeed, agencies []FeedAgency, deletedOperators []int) OifPlan {
	existing = byID(existing)
	deleted := map[int]bool{}
	for _, id := range deletedOperators {
		deleted[id] = true
	}
	lookup := map[string]int{} // resolved agency_id -> row id
	matched := map[int]bool{}
	for _, oif := range existing {
		key := oif.ResolvedGtfsAgencyID.Val
		switch {
		case !oif.OperatorID.Valid:
			if _, ok := lookup[key]; !ok {
				lookup[key] = oif.ID
			}
		case !deleted[oif.OperatorID.Int()]:
			matched[oif.ID] = true
			lookup[key] = oif.ID
		}
	}
	planned := map[string]bool{}
	var plan OifPlan
	for _, agency := range byID(agencies) {
		key := agency.AgencyID.Val
		if planned[key] {
			// A repeated agency_id: the agency with the lowest row id makes the row.
			continue
		}
		planned[key] = true
		if id, ok := lookup[key]; ok {
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

// FormatAgencyPlaces returns the resolved places of an operator-in-feed row from
// its agency's places.
//
// It lists the places ranked over 0.2, highest rank first, each once, joined
// with " / ". Places that tie in rank keep their order in places.
func FormatAgencyPlaces(places []builders.AgencyPlace) string {
	ranked := slices.DeleteFunc(slices.Clone(places), func(p builders.AgencyPlace) bool { return p.Rank <= 0.2 })
	slices.SortStableFunc(ranked, func(a, b builders.AgencyPlace) int { return cmp.Compare(b.Rank, a.Rank) })
	var out []string
	for _, p := range ranked {
		var parts []string
		for _, v := range []tt.String{p.Name, p.Adm1name, p.Adm0name} {
			if v.Valid {
				parts = append(parts, v.Val)
			}
		}
		place := strings.Join(parts, ", ")
		if len(parts) > 0 && !slices.Contains(out, place) {
			out = append(out, place)
		}
	}
	return strings.Join(out, " / ")
}

// byID returns a copy of ents in id order, so a plan doesn't depend on read
// order.
func byID[T interface{ GetID() int }](ents []T) []T {
	return slices.SortedFunc(slices.Values(ents), func(a, b T) int { return cmp.Compare(a.GetID(), b.GetID()) })
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
