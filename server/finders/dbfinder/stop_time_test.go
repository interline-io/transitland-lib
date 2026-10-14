package dbfinder

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testreader"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
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
					require.NoError(t, dbutil.Select(ctx, db, stopDeparturesSelect(fvid, entityIDs, et.entityType, nil, filter.where()), &want))
					require.NoError(t, dbutil.Select(ctx, db, stopDeparturesMaterializedSelect(fvid, entityIDs, et.entityType, nil, filter.where()), &got))
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

// Both departures queries run a trip in frequencies.txt once per start: every
// headway_secs from start_time, up to but not including end_time. Neither
// expands a frequency under minExpandedHeadwaySecs.
func TestStopDeparturesSelect_Frequencies(t *testing.T) {
	if a, ok := testutil.CheckTestDB(); !ok {
		t.Skip(a)
	}
	ctx := context.Background()
	tx, err := testutil.MustOpenTestDB(t).Beginx()
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	// A trip every 30 seconds from 12:00:00 to 13:00:00, on the same pattern as the others
	feed := testreader.NewFrequencyTestFeed()
	feed.TripList = append(feed.TripList, gtfs.Trip{TripID: tt.NewString("rapid"), RouteID: tt.NewKey("route1"), ServiceID: tt.NewKey("service1")})
	feed.StopTimeList = append(feed.StopTimeList,
		gtfs.StopTime{TripID: tt.NewString("rapid"), StopID: tt.NewKey("stop1"), StopSequence: tt.NewInt(1), ArrivalTime: tt.NewSeconds(12 * 3600), DepartureTime: tt.NewSeconds(12 * 3600)},
		gtfs.StopTime{TripID: tt.NewString("rapid"), StopID: tt.NewKey("stop2"), StopSequence: tt.NewInt(2), ArrivalTime: tt.NewSeconds(12*3600 + 600), DepartureTime: tt.NewSeconds(12*3600 + 600)},
	)
	feed.FrequencyList = append(feed.FrequencyList, gtfs.Frequency{TripID: tt.NewString("rapid"), StartTime: tt.NewSeconds(12 * 3600), EndTime: tt.NewSeconds(13 * 3600), HeadwaySecs: tt.NewInt(30)})
	fvid := testdb.CopyTestFeed(t, tx, feed)
	var subMinute int
	require.NoError(t, tx.Get(&subMinute, "select count(*) from gtfs_frequencies where feed_version_id = $1 and headway_secs < $2", fvid, minExpandedHeadwaySecs))
	require.Equal(t, 1, subMinute, "the import keeps the sub-minute frequency")

	// The trips share one journey pattern, each at its own offset, which frequency runs ignore
	var patterns int
	require.NoError(t, tx.Get(&patterns, "select count(distinct journey_pattern_id) from gtfs_trips where feed_version_id = $1", fvid))
	require.Equal(t, 1, patterns, "the trips share one journey pattern")
	var stopID int
	require.NoError(t, tx.Get(&stopID, "select id from gtfs_stops where feed_version_id = $1 and stop_id = $2", fvid, "stop1"))

	date := tt.NewDate(time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC))
	// No departures for rapid, whose only frequency is under a minute
	expect := map[string][]string{
		"regular": {"05:00:00"},
		"offset":  {"07:00:00", "07:30:00"},
		"meet":    {"10:00:00", "10:30:00", "11:00:00", "11:30:00"},
	}
	queries := map[string]func() sq.SelectBuilder{
		"departures": func() sq.SelectBuilder {
			return stopDeparturesSelect(fvid, []int{stopID}, stopTimeEntityStop, nil, &model.StopTimeFilter{ServiceDate: &date})
		},
		"materialized departures": func() sq.SelectBuilder {
			return stopDeparturesMaterializedSelect(fvid, []int{stopID}, stopTimeEntityStop, nil, &model.StopTimeFilter{ServiceDate: &date})
		},
	}
	for name, q := range queries {
		t.Run(name, func(t *testing.T) {
			var sts []*model.StopTime
			require.NoError(t, dbutil.Select(ctx, tx, q(), &sts))
			got := map[string][]string{}
			for _, st := range sts {
				got[st.GtfsTripID] = append(got[st.GtfsTripID], st.DepartureTime.String())
			}
			assert.Equal(t, expect, got)
		})
	}
}

// Both departures queries keep each entity's first limit stop times, in order,
// when one query asks for several entities.
func TestStopDeparturesSelect_Limit(t *testing.T) {
	if a, ok := testutil.CheckTestDB(); !ok {
		t.Skip(a)
	}
	ctx := context.Background()
	tx, err := testutil.MustOpenTestDB(t).Beginx()
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	fvid := testdb.CopyTestFeed(t, tx, testreader.NewFrequencyTestFeed())
	var stop1, stop2 int
	require.NoError(t, tx.Get(&stop1, "select id from gtfs_stops where feed_version_id = $1 and stop_id = $2", fvid, "stop1"))
	require.NoError(t, tx.Get(&stop2, "select id from gtfs_stops where feed_version_id = $1 and stop_id = $2", fvid, "stop2"))

	date := tt.NewDate(time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC))
	queries := map[string]func(limit *int) sq.SelectBuilder{
		"departures": func(limit *int) sq.SelectBuilder {
			return stopDeparturesSelect(fvid, []int{stop1, stop2}, stopTimeEntityStop, limit, &model.StopTimeFilter{ServiceDate: &date})
		},
		"materialized departures": func(limit *int) sq.SelectBuilder {
			return stopDeparturesMaterializedSelect(fvid, []int{stop1, stop2}, stopTimeEntityStop, limit, &model.StopTimeFilter{ServiceDate: &date})
		},
	}
	departureTimes := func(t *testing.T, q sq.SelectBuilder) map[int][]string {
		var sts []*model.StopTime
		require.NoError(t, dbutil.Select(ctx, tx, q, &sts))
		ret := map[int][]string{}
		for _, st := range sts {
			ret[st.StopID.Int()] = append(ret[st.StopID.Int()], st.DepartureTime.String())
		}
		return ret
	}
	for name, q := range queries {
		t.Run(name, func(t *testing.T) {
			// Without a limit, each stop has seven stop times that day
			all := departureTimes(t, q(nil))
			require.Len(t, all[stop1], 7)
			require.Len(t, all[stop2], 7)
			expect := map[int][]string{
				stop1: {"05:00:00", "07:00:00", "07:30:00"},
				stop2: {"05:10:00", "07:10:00", "07:40:00"},
			}
			assert.Equal(t, expect, departureTimes(t, q(ptr(3))))
		})
	}
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
