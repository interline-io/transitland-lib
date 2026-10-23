package sync

import (
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// resolvedAgencyIDs returns the resolved agency_id of each row, in order.
func resolvedAgencyIDs(rows []OifRow) []string {
	var ids []string
	for _, row := range rows {
		ids = append(ids, row.ResolvedGtfsAgencyID.Val)
	}
	return ids
}

// agencyRowIDs returns the agency row id of each row, in order.
func agencyRowIDs(rows []OifRow) []int {
	var ids []int
	for _, row := range rows {
		ids = append(ids, row.AgencyRowID)
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
		// The store has no feed f-absent, so the planner skips the association.
		association("f-absent", ""),
	)
	plan := PlanOperatorOifs(op, nil, feeds)
	require.Len(t, plan.Inserts, 3)
	assert.Equal(t, []string{"A", "C", ""}, resolvedAgencyIDs(plan.Inserts))
	assert.Equal(t, []int{10, 21, 0}, agencyRowIDs(plan.Inserts))
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
		oifRow(100, 1, 5, "A"), // still resolves, so the plan refreshes it
		oifRow(101, 3, 5, "Z"), // on a feed the operator dropped, so the plan deletes it
	}
	plan := PlanOperatorOifs(testOperator(5, association("f-one", ""), association("f-two", "")), existing, feeds)
	require.Len(t, plan.Refreshes, 1)
	assert.Equal(t, 100, plan.Refreshes[0].ID)
	assert.Equal(t, "Test Transit", plan.Refreshes[0].ResolvedName.Val)
	assert.Equal(t, []string{"B"}, resolvedAgencyIDs(plan.Inserts))
	assert.Equal(t, []int{101}, plan.Deletes)
}

// A feed whose active version has no agencies resolves to no agency_id.
func TestPlanOperatorOifs_NoAgencies(t *testing.T) {
	feeds := map[string]FeedAgencies{"f-empty": {FeedID: 1}}
	plan := PlanOperatorOifs(testOperator(5, association("f-empty", "A")), nil, feeds)
	require.Len(t, plan.Inserts, 1)
	assert.Equal(t, "", plan.Inserts[0].ResolvedGtfsAgencyID.Val)
	assert.Equal(t, 0, plan.Inserts[0].AgencyRowID)
}

// Two associations that resolve to the same agency are both inserted, as
// database sync has done.
func TestPlanOperatorOifs_RepeatedAssociation(t *testing.T) {
	feeds := map[string]FeedAgencies{"f-one": {FeedID: 1, Agencies: []FeedAgency{feedAgency(10, "A", "Agency A", "")}}}
	plan := PlanOperatorOifs(testOperator(5, association("f-one", "1"), association("f-one", "2")), nil, feeds)
	assert.Equal(t, []string{"A", "A"}, resolvedAgencyIDs(plan.Inserts))
}

// Of an operator's rows that share a feed and agency, the last one is refreshed
// and the others are deleted.
func TestPlanOperatorOifs_RowsShareKey(t *testing.T) {
	feeds := map[string]FeedAgencies{"f-one": {FeedID: 1, Agencies: []FeedAgency{feedAgency(10, "A", "Agency A", "")}}}
	existing := []dmfr.OperatorAssociatedFeed{oifRow(100, 1, 5, "A"), oifRow(101, 1, 5, "A")}
	plan := PlanOperatorOifs(testOperator(5, association("f-one", "")), existing, feeds)
	require.Len(t, plan.Refreshes, 1)
	assert.Equal(t, 101, plan.Refreshes[0].ID)
	assert.Empty(t, plan.Inserts)
	assert.Equal(t, []int{100}, plan.Deletes)
}

func TestPlanFeedOifs(t *testing.T) {
	agencies := []FeedAgency{
		feedAgency(10, "A", "Agency A", "o-9q9-a"), // has a generated row, which the plan keeps
		feedAgency(11, "B", "Agency B", ""),        // owned by an Atlas row
		feedAgency(12, "C", "Agency C", "o-9q9-c"), // new, with a onestop id
		feedAgency(13, "D", "Agency D", ""),        // new, without one
	}
	existing := []dmfr.OperatorAssociatedFeed{
		oifRow(100, 1, 0, "A"),
		oifRow(101, 1, 0, "B"), // a generated row beside the Atlas row, so the plan deletes it
		oifRow(102, 1, 7, "B"),
		oifRow(103, 1, 0, "X"), // for an agency the version no longer has, so the plan deletes it
		oifRow(104, 1, 8, "Y"), // an Atlas row, which this plan never deletes
	}
	plan := PlanFeedOifs(1, "f-9q9-test", existing, agencies)
	require.Len(t, plan.Inserts, 2)
	assert.Equal(t, []string{"C", "D"}, resolvedAgencyIDs(plan.Inserts))
	assert.Equal(t, "o-9q9-c", plan.Inserts[0].ResolvedOnestopID.Val)
	assert.Equal(t, GeneratedOperatorOnestopID("f-9q9-test", "Agency D"), plan.Inserts[1].ResolvedOnestopID.Val)
	assert.Equal(t, "Agency D", plan.Inserts[1].ResolvedName.Val)
	assert.False(t, plan.Inserts[1].OperatorID.Valid)
	assert.Equal(t, []int{12, 13}, agencyRowIDs(plan.Inserts))
	assert.Empty(t, plan.Refreshes)
	assert.Equal(t, []int{101, 103}, plan.Deletes)
}

// A feed version with no agencies keeps the feed's Atlas rows and deletes its
// generated ones.
func TestPlanFeedOifs_NoAgencies(t *testing.T) {
	existing := []dmfr.OperatorAssociatedFeed{oifRow(100, 1, 0, "A"), oifRow(101, 1, 7, "B")}
	plan := PlanFeedOifs(1, "f-9q9-test", existing, nil)
	assert.Empty(t, plan.Inserts)
	assert.Equal(t, []int{100}, plan.Deletes)
}

// Of a feed's generated rows that share an agency, the first one is kept, and
// an Atlas row owns the agency wherever it comes in the rows.
func TestPlanFeedOifs_RowsShareAgency(t *testing.T) {
	agencies := []FeedAgency{feedAgency(10, "A", "Agency A", ""), feedAgency(11, "B", "Agency B", "")}
	existing := []dmfr.OperatorAssociatedFeed{
		oifRow(100, 1, 0, "A"),
		oifRow(101, 1, 0, "A"),
		oifRow(102, 1, 7, "B"),
		oifRow(103, 1, 0, "B"),
	}
	plan := PlanFeedOifs(1, "f-9q9-test", existing, agencies)
	assert.Empty(t, plan.Inserts)
	assert.Equal(t, []int{101, 103}, plan.Deletes)
}

// Rows match agencies by resolved agency_id alone, so rows that don't carry
// the feed's id still match.
func TestPlanFeedOifs_RowsWithoutFeedID(t *testing.T) {
	agencies := []FeedAgency{feedAgency(10, "A", "Agency A", "")}
	plan := PlanFeedOifs(1, "f-9q9-test", []dmfr.OperatorAssociatedFeed{oifRow(100, 0, 0, "A")}, agencies)
	assert.Empty(t, plan.Inserts)
	assert.Empty(t, plan.Deletes)
}
