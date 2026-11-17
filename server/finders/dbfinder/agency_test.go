package dbfinder

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/stretchr/testify/assert"
)

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
	onestopIDs := func(agencies []*model.Agency) []string {
		var out []string
		for _, a := range agencies {
			out = append(out, a.OnestopID)
		}
		return out
	}

	agencies, err := f.FindAgencies(ctx, nil, nil, nil, &model.AgencyFilter{AgencyID: ptr("caltrain-ca-us")})
	if err != nil {
		t.Fatal(err)
	}
	if !assert.Equal(t, []string{"o-9q9-caltrain"}, onestopIDs(agencies)) {
		return
	}

	byVersion, err := f.AgenciesByFeedVersionIDs(ctx, nil, nil, []int{agencies[0].FeedVersionID})
	if err != nil {
		t.Fatal(err)
	}
	if assert.Len(t, byVersion, 1) {
		assert.Equal(t, []string{"o-9q9-caltrain"}, onestopIDs(byVersion[0]))
	}

	named, err := f.FindAgencies(ctx, nil, nil, nil, &model.AgencyFilter{OnestopID: ptr("o-9q9-caltrain~second")})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"o-9q9-caltrain~second"}, onestopIDs(named))

	byOperator, err := f.AgenciesByOnestopIDs(ctx, nil, nil, []string{"o-9q9-caltrain", "o-9q9-caltrain~second"})
	if err != nil {
		t.Fatal(err)
	}
	var grouped [][]string
	for _, group := range byOperator {
		grouped = append(grouped, onestopIDs(group))
	}
	assert.Equal(t, [][]string{{"o-9q9-caltrain"}, {"o-9q9-caltrain~second"}}, grouped)
}
