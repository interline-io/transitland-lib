package dbfinder

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/sync"
	"github.com/interline-io/transitland-lib/tldb/postgres"
	"github.com/stretchr/testify/assert"
)

// agencyOnestopIDs returns each agency's operator onestop id.
func agencyOnestopIDs(agencies []*model.Agency) []string {
	var out []string
	for _, a := range agencies {
		out = append(out, a.OnestopID)
	}
	return out
}

// An agency that two operators claim comes back once, with its first operator,
// unless a query names an operator: then once under each operator it names.
func TestFinder_AgenciesWithTwoOperators(t *testing.T) {
	ctx := context.Background()
	tx, err := testutil.MustOpenTestDB(t).BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	addSecondOperator(ctx, t, tx)
	f := NewFinder(tx)

	agencies, err := f.FindAgencies(ctx, nil, nil, nil, &model.AgencyFilter{AgencyID: ptr("caltrain-ca-us")})
	if err != nil {
		t.Fatal(err)
	}
	if !assert.Equal(t, []string{"o-9q9-caltrain"}, agencyOnestopIDs(agencies)) {
		return
	}

	byVersion, err := f.AgenciesByFeedVersionIDs(ctx, nil, nil, []int{agencies[0].FeedVersionID})
	if err != nil {
		t.Fatal(err)
	}
	if assert.Len(t, byVersion, 1) {
		assert.Equal(t, []string{"o-9q9-caltrain"}, agencyOnestopIDs(byVersion[0]))
	}

	named, err := f.FindAgencies(ctx, nil, nil, nil, &model.AgencyFilter{OnestopID: ptr("o-9q9-caltrain~second")})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"o-9q9-caltrain~second"}, agencyOnestopIDs(named))

	byOperator, err := f.AgenciesByOnestopIDs(ctx, nil, nil, []string{"o-9q9-caltrain", "o-9q9-caltrain~second"})
	if err != nil {
		t.Fatal(err)
	}
	var grouped [][]string
	for _, group := range byOperator {
		grouped = append(grouped, agencyOnestopIDs(group))
	}
	assert.Equal(t, [][]string{{"o-9q9-caltrain"}, {"o-9q9-caltrain~second"}}, grouped)
}

// A deleted Atlas operator's row claims no agency, though it stays until a sync
// of its feed removes it.
func TestFinder_AgenciesWithDeletedOperator(t *testing.T) {
	ctx := context.Background()
	tx, err := testutil.MustOpenTestDB(t).BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	second := addSecondOperator(ctx, t, tx)
	f := NewFinder(tx)
	caltrain := &model.AgencyFilter{AgencyID: ptr("caltrain-ca-us")}

	// Caltrain's operator holds the agency's first row.
	agencies, err := f.FindAgencies(ctx, nil, nil, nil, caltrain)
	if err != nil {
		t.Fatal(err)
	}
	if !assert.Equal(t, []string{"o-9q9-caltrain"}, agencyOnestopIDs(agencies)) {
		return
	}

	// Hiding every operator but the second deletes Caltrain's and leaves its rows.
	if _, err := sync.HideUnseedOperators(postgres.NewPostgresAdapterFromDBX(tx), []int{second}); err != nil {
		t.Fatal(err)
	}

	agencies, err = f.FindAgencies(ctx, nil, nil, nil, caltrain)
	if err != nil {
		t.Fatal(err)
	}
	if !assert.Equal(t, []string{"o-9q9-caltrain~second"}, agencyOnestopIDs(agencies)) {
		return
	}
	ops, errs := f.OperatorsByCOIFs(ctx, []int{*agencies[0].CoifID})
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	if assert.Len(t, ops, 1) && assert.NotNil(t, ops[0]) {
		assert.Equal(t, "o-9q9-caltrain~second", ops[0].OnestopID.Val)
	}

	named, err := f.FindAgencies(ctx, nil, nil, nil, &model.AgencyFilter{OnestopID: ptr("o-9q9-caltrain")})
	if err != nil {
		t.Fatal(err)
	}
	assert.Empty(t, named)
}
