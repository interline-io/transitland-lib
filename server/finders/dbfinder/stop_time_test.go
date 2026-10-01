package dbfinder

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The materialized departures query is behind a flag, so turning the flag off
// must be a clean rollback: both queries return the same rows for the stops,
// locations and location groups of every test feed version, over a week.
func TestStopDeparturesMaterializedSelect(t *testing.T) {
	ctx := context.Background()
	db := testutil.MustOpenTestDB(t)
	f := NewFinder(db)
	var fvids []int
	require.NoError(t, db.Select(&fvids, "select distinct feed_version_id from gtfs_stop_times order by 1"))
	// Batches of up to 50 entities, like a loader batch
	entityTypes := []struct {
		name       string
		entityType stopTimeEntityType
		idQuery    string
	}{
		{"stop", stopTimeEntityStop, "select distinct stop_id from gtfs_stop_times where feed_version_id = $1 and stop_id is not null order by 1 limit 50"},
		{"location", stopTimeEntityLocation, "select distinct location_id from gtfs_stop_times where feed_version_id = $1 and location_id is not null order by 1 limit 50"},
		{"location group", stopTimeEntityLocationGroup, "select distinct location_group_id from gtfs_stop_times where feed_version_id = $1 and location_group_id is not null order by 1 limit 50"},
	}
	type departureFilter struct {
		name  string
		where func() *model.StopTimeFilter
	}
	rowCounts := map[string]int{}
	for _, fvid := range fvids {
		// A week of service: the fallback week, or for a feed version without a
		// service window, the week from its earliest calendar
		fvsw, err := f.FindFeedVersionServiceWindow(ctx, fvid)
		require.NoError(t, err)
		week := fvsw.FallbackWeek
		if !fvsw.EndDate.After(fvsw.StartDate) {
			require.NoError(t, db.Get(&week, "select min(start_date) from gtfs_calendars where feed_version_id = $1", fvid))
		}
		var routeOnestopIds []string
		require.NoError(t, db.Select(&routeOnestopIds, "select distinct onestop_id from feed_version_route_onestop_ids where feed_version_id = $1 order by 1 limit 2", fvid))
		for _, et := range entityTypes {
			var entityIDs []int
			require.NoError(t, db.Select(&entityIDs, et.idQuery, fvid))
			if len(entityIDs) == 0 {
				continue
			}
			for day := 0; day < 7; day++ {
				date := tt.NewDate(week.AddDate(0, 0, day))
				// Built fresh for each query, since the builders may rewrite the filter
				filters := []departureFilter{
					{"service date", func() *model.StopTimeFilter {
						return &model.StopTimeFilter{ServiceDate: &date}
					}},
				}
				// The other filters apply the same way on any day
				if day == 0 {
					filters = append(filters,
						departureFilter{"exclude first and last", func() *model.StopTimeFilter {
							return &model.StopTimeFilter{ServiceDate: &date, ExcludeFirst: ptr(true), ExcludeLast: ptr(true)}
						}},
						departureFilter{"time window", func() *model.StopTimeFilter {
							return &model.StopTimeFilter{ServiceDate: &date, StartTime: ptr(7 * 3600), EndTime: ptr(10 * 3600)}
						}},
					)
					if len(routeOnestopIds) > 0 {
						filters = append(filters,
							departureFilter{"routes", func() *model.StopTimeFilter {
								return &model.StopTimeFilter{ServiceDate: &date, RouteOnestopIds: routeOnestopIds}
							}},
							departureFilter{"previous routes", func() *model.StopTimeFilter {
								return &model.StopTimeFilter{ServiceDate: &date, RouteOnestopIds: routeOnestopIds, AllowPreviousRouteOnestopIds: ptr(true)}
							}},
						)
					}
				}
				for _, filter := range filters {
					var want, got []*model.StopTime
					require.NoError(t, dbutil.Select(ctx, db, stopDeparturesSelect(fvid, entityIDs, et.entityType, filter.where()), &want))
					require.NoError(t, dbutil.Select(ctx, db, stopDeparturesMaterializedSelect(fvid, entityIDs, et.entityType, filter.where()), &got))
					sortStopTimes(want)
					sortStopTimes(got)
					// Not assert.Equal: diffing thousands of rows takes minutes
					if !assert.ObjectsAreEqual(want, got) {
						t.Fatalf("feed version %d, %s, %s, %s: got %d rows, want %d; %s", fvid, et.name, date.Val.Format(time.DateOnly), filter.name, len(got), len(want), firstStopTimeDiff(want, got))
					}
					rowCounts[et.name] += len(want)
				}
			}
		}
	}
	// Guard against comparing nothing but empty results
	assert.Greater(t, rowCounts["stop"], 0, "stop departures compared")
	assert.Greater(t, rowCounts["location"], 0, "location departures compared")
	assert.Greater(t, rowCounts["location group"], 0, "location group departures compared")
}

// firstStopTimeDiff describes the first row where two sorted results differ.
func firstStopTimeDiff(want, got []*model.StopTime) string {
	n := min(len(want), len(got))
	for i := 0; i < n; i++ {
		if !assert.ObjectsAreEqual(want[i], got[i]) {
			return fmt.Sprintf("first difference at row %d: got %+v, want %+v", i, *got[i], *want[i])
		}
	}
	return fmt.Sprintf("first difference at row %d, where one result ends", n)
}

func sortStopTimes(sts []*model.StopTime) {
	slices.SortFunc(sts, func(a, b *model.StopTime) int {
		return cmp.Or(
			cmp.Compare(a.TripID.Val, b.TripID.Val),
			cmp.Compare(a.StopSequence.Val, b.StopSequence.Val),
			cmp.Compare(a.DepartureTime.Val, b.DepartureTime.Val),
		)
	})
}
