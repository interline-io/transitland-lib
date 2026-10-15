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

// Paging through a list sorted by distance from a focus point returns every row once, in
// the order of the unpaged list. That holds only if the cursor compares the sort key.
func TestFinder_FocusCursorPaging(t *testing.T) {
	db := testutil.MustOpenTestDB(t)
	f := NewFinder(dbutil.WithQueryLogger(db, false, 0))
	ctx := context.Background()
	materialized := model.WithConfig(ctx, model.Config{UseMaterialized: true})

	// 19TH and 19TH_N share a location, as do MCAR and MCAR_S, so each pair ties on
	// distance. From the second point, a sphere and the spheroid order some BART stops
	// differently.
	at19th := &model.FocusPoint{Lon: -122.268602, Lat: 37.80835}
	southEast := &model.FocusPoint{Lon: -122.0, Lat: 37.6}
	ba := "BA"
	// Agencies without a geometry have no distance, so the bbox leaves them out.
	us := &model.BoundingBox{MinLon: -125, MinLat: 24, MaxLon: -66, MaxLat: 50}

	stopRow := func(e *model.Stop) cursorRow { return cursorRow{e.ID, e.FeedVersionID} }
	routeRow := func(e *model.Route) cursorRow { return cursorRow{e.ID, e.FeedVersionID} }
	stops := func(focus *model.FocusPoint) func(int, *model.Cursor) ([]cursorRow, error) {
		return func(limit int, after *model.Cursor) ([]cursorRow, error) {
			where := &model.StopFilter{FeedOnestopID: &ba, Location: &model.StopLocationFilter{Focus: focus}}
			return cursorRows(stopRow)(f.FindStops(ctx, &limit, after, nil, where))
		}
	}
	routes := func(routeCtx context.Context) func(int, *model.Cursor) ([]cursorRow, error) {
		return func(limit int, after *model.Cursor) ([]cursorRow, error) {
			where := &model.RouteFilter{FeedOnestopID: &ba, Location: &model.RouteLocationFilter{Focus: southEast}}
			return cursorRows(routeRow)(f.FindRoutes(routeCtx, &limit, after, nil, where))
		}
	}
	checks := []struct {
		name string
		page func(limit int, after *model.Cursor) ([]cursorRow, error)
	}{
		{"stops tied on distance", stops(at19th)},
		{"stops by distance", stops(southEast)},
		{"routes by distance", routes(ctx)},
		{"materialized routes by distance", routes(materialized)},
		{"agencies by distance", func(limit int, after *model.Cursor) ([]cursorRow, error) {
			where := &model.AgencyFilter{Location: &model.AgencyLocationFilter{Bbox: us, Focus: southEast}}
			return cursorRows(func(e *model.Agency) cursorRow { return cursorRow{e.ID, e.FeedVersionID} })(f.FindAgencies(ctx, &limit, after, nil, where))
		}},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			all, err := check.page(1000, nil)
			require.NoError(t, err)
			require.Greater(t, len(all), 1, "need more than one row to page through")

			// One row per page, so every adjacent pair straddles a page boundary.
			var paged []cursorRow
			var after *model.Cursor
			for range all {
				rows, err := check.page(1, after)
				require.NoError(t, err)
				if len(rows) == 0 {
					break
				}
				paged = append(paged, rows[0])
				next := model.NewCursor(0, rows[0].id)
				after = &next
			}
			assert.Equal(t, all, paged, "paging must return the unpaged list")
		})
	}
}
