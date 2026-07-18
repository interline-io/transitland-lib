package dbfinder

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/stretchr/testify/require"
)

// Exercises the active+materialized focus-pagination path, which had no coverage --
// letting a broken cursor subquery (text route_id vs int id, and a missing geometry
// column) survive. Asserts only that the query executes: it is the first test to run
// routeSelect in materialized mode.
func TestRouteSelect_MaterializedFocusCursor(t *testing.T) {
	ctx := context.Background()
	db := testutil.MustOpenTestDB(t)

	var afterID int
	if err := db.Get(&afterID, "select id from tl_materialized_active_routes limit 1"); err != nil {
		t.Skipf("no materialized routes in test db: %v", err)
	}

	q := routeSelect(
		nil,
		&model.Cursor{Valid: true, ID: afterID},
		nil,
		&UseActive{active: true, materialized: true},
		&model.PermFilter{IsGlobalAdmin: true},
		&model.RouteFilter{Location: &model.RouteLocationFilter{Focus: &model.FocusPoint{Lon: -122.4, Lat: 37.8}}},
	)
	var ents []*model.Route
	require.NoError(t, dbutil.Select(ctx, db, q, &ents))
}
