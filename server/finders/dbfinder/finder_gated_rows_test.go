package dbfinder

import (
	"context"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A cursor that names a row the caller can't see starts no page, the same as an id that
// doesn't exist, so it can't reveal where the hidden row sorts.
func TestFinder_CursorGate(t *testing.T) {
	// Runs in a transaction that is always rolled back: each check hides a feed, and the
	// test database is shared with the other server packages that go test runs concurrently.
	db := testutil.MustOpenTestDB(t)
	tx, err := db.Beginx()
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	f := NewFinder(dbutil.WithQueryLogger(tx, false, 0))
	ctx := context.Background()
	materialized := model.WithConfig(ctx, model.Config{UseMaterialized: true})

	limit := 10
	focus := &model.FocusPoint{Lon: -122.27, Lat: 37.80}
	stopRow := func(e *model.Stop) cursorRow { return cursorRow{e.ID, e.FeedVersionID} }
	routeRow := func(e *model.Route) cursorRow { return cursorRow{e.ID, e.FeedVersionID} }
	agencyRow := func(e *model.Agency) cursorRow { return cursorRow{e.ID, e.FeedVersionID} }
	checks := []struct {
		name string
		page func(after *model.Cursor) ([]cursorRow, error)
	}{
		{"stops", func(after *model.Cursor) ([]cursorRow, error) {
			return cursorRows(stopRow)(f.FindStops(ctx, &limit, after, nil, nil))
		}},
		{"stops by distance", func(after *model.Cursor) ([]cursorRow, error) {
			where := &model.StopFilter{Location: &model.StopLocationFilter{Focus: focus}}
			return cursorRows(stopRow)(f.FindStops(ctx, &limit, after, nil, where))
		}},
		{"routes", func(after *model.Cursor) ([]cursorRow, error) {
			return cursorRows(routeRow)(f.FindRoutes(ctx, &limit, after, nil, nil))
		}},
		{"routes by distance", func(after *model.Cursor) ([]cursorRow, error) {
			where := &model.RouteFilter{Location: &model.RouteLocationFilter{Focus: focus}}
			return cursorRows(routeRow)(f.FindRoutes(ctx, &limit, after, nil, where))
		}},
		{"materialized routes by distance", func(after *model.Cursor) ([]cursorRow, error) {
			where := &model.RouteFilter{Location: &model.RouteLocationFilter{Focus: focus}}
			return cursorRows(routeRow)(f.FindRoutes(materialized, &limit, after, nil, where))
		}},
		{"agencies", func(after *model.Cursor) ([]cursorRow, error) {
			return cursorRows(agencyRow)(f.FindAgencies(ctx, &limit, after, nil, nil))
		}},
		{"agencies by distance", func(after *model.Cursor) ([]cursorRow, error) {
			where := &model.AgencyFilter{Location: &model.AgencyLocationFilter{Focus: focus}}
			return cursorRows(agencyRow)(f.FindAgencies(ctx, &limit, after, nil, where))
		}},
		{"trips", func(after *model.Cursor) ([]cursorRow, error) {
			return cursorRows(func(e *model.Trip) cursorRow { return cursorRow{e.ID, e.FeedVersionID} })(f.FindTrips(ctx, &limit, after, nil, nil))
		}},
		{"feed versions", func(after *model.Cursor) ([]cursorRow, error) {
			return cursorRows(func(e *model.FeedVersion) cursorRow { return cursorRow{e.ID, e.ID} })(f.FindFeedVersions(ctx, &limit, after, nil, nil))
		}},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			first, err := check.page(nil)
			require.NoError(t, err)
			require.NotEmpty(t, first, "found no rows to aim a cursor at")
			after := model.NewCursor(0, first[0].id)

			// Every other row sorts after the first one, so while its feed is visible the
			// cursor starts a page.
			rows, err := check.page(&after)
			require.NoError(t, err)
			require.NotEmpty(t, rows, "a cursor at the first row should start a page")

			setFeedPublic(t, tx, first[0].fvid, false)
			t.Cleanup(func() { setFeedPublic(t, tx, first[0].fvid, true) })
			rows, err = check.page(&after)
			require.NoError(t, err)
			assert.Empty(t, rows, "a cursor at a hidden row must start no page")
		})
	}
}

// A stop's observations come only from feed versions the caller can see, whichever feed
// version the request names.
func TestFinder_StopObservationGate(t *testing.T) {
	db := testutil.MustOpenTestDB(t)
	tx, err := db.Beginx()
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	f := NewFinder(dbutil.WithQueryLogger(tx, false, 0))
	ctx := context.Background()

	// The fixtures record an observation at BART's FTVL against BART's active feed version.
	ba, ftvlID := "BA", "FTVL"
	stops, err := f.FindStops(ctx, nil, nil, nil, &model.StopFilter{FeedOnestopID: &ba, StopID: &ftvlID})
	require.NoError(t, err)
	require.Len(t, stops, 1, "expected the bart FTVL stop")
	ftvl := stops[0]
	day, err := time.Parse("2006-01-02", "2023-03-09")
	require.NoError(t, err)
	filters := []struct {
		name  string
		where *model.StopObservationFilter
	}{
		{"named feed version", &model.StopObservationFilter{Source: "TripUpdate", FeedVersionID: ftvl.FeedVersionID, TripStartDate: tt.NewDate(day)}},
		{"any feed version", nil},
	}
	count := func(t *testing.T, where *model.StopObservationFilter) int {
		t.Helper()
		obs, err := f.StopObservationsByStopIDs(ctx, nil, where, []int{ftvl.ID})
		require.NoError(t, err)
		return len(obs[0])
	}

	for _, filter := range filters {
		require.NotZero(t, count(t, filter.where), "%s: expected the fixture observation while BART is public", filter.name)
	}
	setFeedPublic(t, tx, ftvl.FeedVersionID, false)
	for _, filter := range filters {
		t.Run(filter.name, func(t *testing.T) {
			assert.Zero(t, count(t, filter.where), "observations of a hidden feed version must not be returned")
		})
	}
}

// cursorRow is what a cursor check needs of a row: its id, to aim a cursor at, and its
// feed version, to find the feed to hide.
type cursorRow struct{ id, fvid int }

// cursorRows adapts a finder's results to cursorRows.
func cursorRows[T any](row func(T) cursorRow) func([]T, error) ([]cursorRow, error) {
	return func(ents []T, err error) ([]cursorRow, error) {
		rows := make([]cursorRow, 0, len(ents))
		for _, ent := range ents {
			rows = append(rows, row(ent))
		}
		return rows, err
	}
}

// setFeedPublic sets whether the feed of feed version fvid is public.
func setFeedPublic(t *testing.T, tx *sqlx.Tx, fvid int, public bool) {
	t.Helper()
	_, err := tx.Exec(`UPDATE feed_states SET public = $1 WHERE feed_id = (SELECT feed_id FROM feed_versions WHERE id = $2)`, public, fvid)
	require.NoError(t, err)
}
