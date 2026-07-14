package dbfinder

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A feed version whose import is incomplete must not return entities. This is what lets
// import and unimport write without an enclosing transaction: they flip in_progress and
// the rows become invisible immediately, whatever state they are left in.
func TestFinder_ImportGate(t *testing.T) {
	ctx := context.Background()
	db := testutil.MustOpenTestDB(t)
	f := NewFinder(dbutil.WithQueryLogger(db, false, 0))

	// BART, from the standard test fixtures.
	sha1 := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
	fvs, err := f.FindFeedVersions(ctx, nil, nil, nil, &model.FeedVersionFilter{Sha1: &sha1})
	require.NoError(t, err)
	require.Len(t, fvs, 1, "expected the bart test feed version")
	fvid := fvs[0].ID

	// Restore the import record no matter how the test exits; other tests share this db.
	setImport := func(t *testing.T, success bool, inProgress bool) {
		t.Helper()
		_, err := db.ExecContext(ctx,
			`UPDATE feed_version_gtfs_imports SET success = $1, in_progress = $2 WHERE feed_version_id = $3`,
			success, inProgress, fvid)
		require.NoError(t, err)
	}
	t.Cleanup(func() { setImport(t, true, false) })

	// counts returns the number of entities visible for this feed version across every
	// finder that reads a raw imported table.
	counts := func(t *testing.T) map[string]int {
		t.Helper()
		out := map[string]int{}
		stops, err := f.FindStops(ctx, nil, nil, nil, &model.StopFilter{FeedVersionSha1: &sha1})
		require.NoError(t, err)
		out["stops"] = len(stops)
		routes, err := f.FindRoutes(ctx, nil, nil, nil, &model.RouteFilter{FeedVersionSha1: &sha1})
		require.NoError(t, err)
		out["routes"] = len(routes)
		trips, err := f.FindTrips(ctx, nil, nil, nil, &model.TripFilter{FeedVersionSha1: &sha1})
		require.NoError(t, err)
		out["trips"] = len(trips)
		agencies, err := f.FindAgencies(ctx, nil, nil, nil, &model.AgencyFilter{FeedVersionSha1: &sha1})
		require.NoError(t, err)
		out["agencies"] = len(agencies)
		return out
	}

	// Baseline: a completed import is visible.
	setImport(t, true, false)
	for k, v := range counts(t) {
		assert.Greater(t, v, 0, "%s should be visible for a completed import", k)
	}

	// Both conditions are load-bearing, and neither subsumes the other: a feed version
	// being unimported still has success = true, and a failed import has in_progress = false.
	for _, tc := range []struct {
		name       string
		success    bool
		inProgress bool
	}{
		{"import in flight", false, true},
		{"unimport in flight", true, true},
		{"import failed", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setImport(t, tc.success, tc.inProgress)
			for k, v := range counts(t) {
				assert.Equal(t, 0, v, "%s must be hidden while: %s", k, tc.name)
			}
		})
	}

	// And visible again once the import completes.
	setImport(t, true, false)
	for k, v := range counts(t) {
		assert.Greater(t, v, 0, "%s should be visible again after the import completes", k)
	}
}
