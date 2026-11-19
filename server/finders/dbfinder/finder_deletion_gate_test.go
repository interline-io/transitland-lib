package dbfinder

import (
	"context"
	"slices"
	"testing"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A deleted feed, feed version or Atlas operator hides the rows that depend on it from every
// caller.
func TestFinder_DeletionGate(t *testing.T) {
	ctx := context.Background()

	// Runs in a transaction that is always rolled back, since the test database is shared
	// with the other server packages that go test runs concurrently.
	db := testutil.MustOpenTestDB(t)
	tx, err := db.Beginx()
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	f := NewFinder(dbutil.WithQueryLogger(tx, false, 0))

	// Caltrain, from the standard test fixtures: the CT feed, its one feed version, and its
	// Atlas operator, which the CT~rt feed names too.
	feedOnestopID := "CT"
	sha1 := "d2813c293bcfd7a97dde599527ae6c62c98e66c6"
	operatorOnestopID := "o-9q9-caltrain"
	feeds, err := f.FindFeeds(ctx, nil, nil, nil, &model.FeedFilter{OnestopID: &feedOnestopID})
	require.NoError(t, err)
	require.Len(t, feeds, 1, "expected the caltrain test feed")
	feedID := feeds[0].ID

	// counts returns how much of Caltrain each finder sees.
	counts := func(t *testing.T) map[string]int {
		t.Helper()
		out := map[string]int{}
		feeds, err := f.FindFeeds(ctx, nil, nil, nil, &model.FeedFilter{OnestopID: &feedOnestopID})
		require.NoError(t, err)
		out["feeds"] = len(feeds)
		fvs, err := f.FindFeedVersions(ctx, nil, nil, nil, &model.FeedVersionFilter{Sha1: &sha1})
		require.NoError(t, err)
		out["feed versions"] = len(fvs)
		stops, err := f.FindStops(ctx, nil, nil, nil, &model.StopFilter{FeedVersionSha1: &sha1})
		require.NoError(t, err)
		out["stops"] = len(stops)
		feedOperators, err := f.OperatorsByFeedIDs(ctx, nil, nil, []int{feedID})
		require.NoError(t, err)
		out["feed operators"] = len(feedOperators[0])
		operators, err := f.FindOperators(ctx, nil, nil, nil, &model.OperatorFilter{OnestopID: &operatorOnestopID})
		require.NoError(t, err)
		out["operators"] = len(operators)
		return out
	}

	// Baseline: with nothing deleted, every finder sees Caltrain.
	for k, v := range counts(t) {
		require.Greater(t, v, 0, "%s should see caltrain before any deletion", k)
	}

	for _, tc := range []struct {
		name   string
		delete string // marks the row that key names as deleted
		key    string
		hidden []string
	}{
		{
			name:   "feed version deleted",
			delete: `UPDATE feed_versions SET deleted_at = now() WHERE sha1 = $1`,
			key:    sha1,
			hidden: []string{"feed versions", "stops"},
		},
		{
			// The operator stays visible through CT~rt, whose feed isn't deleted.
			name:   "feed deleted",
			delete: `UPDATE current_feeds SET deleted_at = now() WHERE onestop_id = $1`,
			key:    feedOnestopID,
			hidden: []string{"feeds", "feed versions", "stops", "feed operators"},
		},
		{
			name:   "atlas operator deleted",
			delete: `UPDATE current_operators SET deleted_at = now() WHERE onestop_id = $1`,
			key:    operatorOnestopID,
			hidden: []string{"feed operators", "operators"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tx.ExecContext(ctx, `SAVEPOINT deletion`)
			require.NoError(t, err)
			t.Cleanup(func() {
				_, err := tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT deletion`)
				require.NoError(t, err)
			})
			_, err = tx.ExecContext(ctx, tc.delete, tc.key)
			require.NoError(t, err)
			for k, v := range counts(t) {
				if slices.Contains(tc.hidden, k) {
					assert.Equal(t, 0, v, "%s should be hidden", k)
				} else {
					assert.Greater(t, v, 0, "%s should stay visible", k)
				}
			}
		})
	}
}
