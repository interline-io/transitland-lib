package sync

import (
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

func feedAgency(id int, agencyID, name, onestopID string) FeedAgency {
	a := FeedAgency{Agency: gtfs.Agency{AgencyID: tt.NewString(agencyID), AgencyName: tt.NewString(name)}}
	a.ID = id
	if onestopID != "" {
		a.OnestopID = tt.NewString(onestopID)
	}
	return a
}

func association(feedOnestopID, gtfsAgencyID string) dmfr.OperatorAssociatedFeed {
	return dmfr.OperatorAssociatedFeed{FeedOnestopID: tt.NewString(feedOnestopID), GtfsAgencyID: tt.NewString(gtfsAgencyID)}
}

func oifRow(id, feedID, operatorID int, resolvedAgencyID string) dmfr.OperatorAssociatedFeed {
	oif := dmfr.OperatorAssociatedFeed{FeedID: feedID, ResolvedGtfsAgencyID: tt.NewString(resolvedAgencyID)}
	oif.ID = id
	if operatorID > 0 {
		oif.OperatorID = tt.NewInt(operatorID)
	}
	return oif
}

func testOperator(id int, feeds ...dmfr.OperatorAssociatedFeed) dmfr.Operator {
	op := dmfr.Operator{OnestopID: tt.NewString("o-test"), Name: tt.NewString("Test Transit"), ShortName: tt.NewString("TT"), AssociatedFeeds: feeds}
	op.ID = id
	return op
}

// The resolved agency_id of each row a plan writes, in order.
func resolvedAgencyIDs(rows []OifRow) []string {
	var ids []string
	for _, row := range rows {
		ids = append(ids, row.ResolvedGtfsAgencyID.Val)
	}
	return ids
}

func TestPlanOperatorOifs_Resolve(t *testing.T) {
	feeds := map[string]FeedAgencies{
		"f-one":  {FeedID: 1, Agencies: []FeedAgency{feedAgency(10, "A", "Agency A", "")}},
		"f-many": {FeedID: 2, Agencies: []FeedAgency{feedAgency(20, "B", "Agency B", ""), feedAgency(21, "C", "Agency C", "")}},
	}
	op := testOperator(5,
		// The only agency matches whatever agency_id the association names.
		association("f-one", "other"),
		// With several, the association's agency_id picks one, or none.
		association("f-many", "C"),
		association("f-many", "missing"),
		// A feed the store doesn't have is skipped.
		association("f-absent", ""),
	)
	plan := PlanOperatorOifs(op, nil, feeds)
	assert.Equal(t, []string{"A", "C", ""}, resolvedAgencyIDs(plan.Inserts))
	assert.Equal(t, []int{10, 21, 0}, []int{plan.Inserts[0].AgencyID, plan.Inserts[1].AgencyID, plan.Inserts[2].AgencyID})
	assert.Empty(t, plan.Refreshes)
	assert.Empty(t, plan.Deletes)
	first := plan.Inserts[0]
	assert.Equal(t, 1, first.FeedID)
	assert.Equal(t, 5, first.OperatorID.Int())
	assert.Equal(t, "o-test", first.ResolvedOnestopID.Val)
	assert.Equal(t, "Test Transit", first.ResolvedName.Val)
	assert.Equal(t, "TT", first.ResolvedShortName.Val)
}

func TestPlanOperatorOifs_Existing(t *testing.T) {
	feeds := map[string]FeedAgencies{
		"f-one": {FeedID: 1, Agencies: []FeedAgency{feedAgency(10, "A", "Agency A", "")}},
		"f-two": {FeedID: 2, Agencies: []FeedAgency{feedAgency(20, "B", "Agency B", "")}},
	}
	existing := []dmfr.OperatorAssociatedFeed{
		oifRow(100, 1, 5, "A"), // still resolves: refreshed
		oifRow(101, 3, 5, "Z"), // a feed the operator dropped: deleted
	}
	plan := PlanOperatorOifs(testOperator(5, association("f-one", ""), association("f-two", "")), existing, feeds)
	if assert.Len(t, plan.Refreshes, 1) {
		assert.Equal(t, 100, plan.Refreshes[0].ID)
		assert.Equal(t, "Test Transit", plan.Refreshes[0].ResolvedName.Val)
	}
	assert.Equal(t, []string{"B"}, resolvedAgencyIDs(plan.Inserts))
	assert.Equal(t, []int{101}, plan.Deletes)
}

func TestPlanFeedOifs(t *testing.T) {
	agencies := []FeedAgency{
		feedAgency(10, "A", "Agency A", "o-9q9-a"), // has a generated row: kept
		feedAgency(11, "B", "Agency B", ""),        // an Atlas row owns it
		feedAgency(12, "C", "Agency C", "o-9q9-c"), // new, with a onestop id
		feedAgency(13, "D", "Agency D", ""),        // new, without one
	}
	existing := []dmfr.OperatorAssociatedFeed{
		oifRow(100, 1, 0, "A"),
		oifRow(101, 1, 0, "B"), // a generated row beside the Atlas row: deleted
		oifRow(102, 1, 7, "B"),
		oifRow(103, 1, 0, "X"), // an agency the version no longer has: deleted
		oifRow(104, 1, 8, "Y"), // an Atlas row is never deleted here
	}
	plan := PlanFeedOifs(1, "f-9q9-test", existing, agencies)
	assert.Equal(t, []string{"C", "D"}, resolvedAgencyIDs(plan.Inserts))
	if assert.Len(t, plan.Inserts, 2) {
		assert.Equal(t, "o-9q9-c", plan.Inserts[0].ResolvedOnestopID.Val)
		assert.Equal(t, GeneratedOperatorOnestopID("f-9q9-test", "Agency D"), plan.Inserts[1].ResolvedOnestopID.Val)
		assert.Equal(t, "Agency D", plan.Inserts[1].ResolvedName.Val)
		assert.False(t, plan.Inserts[1].OperatorID.Valid)
		assert.Equal(t, 13, plan.Inserts[1].AgencyID)
	}
	assert.Empty(t, plan.Refreshes)
	assert.Equal(t, []int{101, 103}, plan.Deletes)
}
