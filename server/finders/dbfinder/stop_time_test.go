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
	"github.com/jmoiron/sqlx"
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
						departureFilter{"route limit", func() *model.StopTimeFilter {
							return &model.StopTimeFilter{ServiceDate: &date, RouteLimit: ptr(1)}
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

// With RouteLimit, both departures queries keep each route's first stop times
// at a stop, rather than the stop's first.
func TestStopDeparturesSelect_RouteLimit(t *testing.T) {
	if a, ok := testutil.CheckTestDB(); !ok {
		t.Skip(a)
	}
	ctx := context.Background()
	db := testutil.MustOpenTestDB(t)
	// HART's Marion Transit Center, which many routes serve
	var fvid, stopID int
	require.NoError(t, db.Get(&fvid, "select feed_versions.id from feed_versions join current_feeds on current_feeds.id = feed_versions.feed_id join feed_states on feed_states.feed_version_id = feed_versions.id where current_feeds.onestop_id = $1", "HA"))
	require.NoError(t, db.Get(&stopID, "select id from gtfs_stops where feed_version_id = $1 and stop_id = $2", fvid, "6968"))
	tripRoutes := tripRouteIDs(t, db, fvid)

	date := tt.NewDate(time.Date(2018, 5, 29, 0, 0, 0, 0, time.UTC))
	queries := map[string]func(routeLimit *int) sq.SelectBuilder{
		"departures": func(routeLimit *int) sq.SelectBuilder {
			return stopDeparturesSelect(fvid, []int{stopID}, stopTimeEntityStop, nil, &model.StopTimeFilter{ServiceDate: &date, RouteLimit: routeLimit})
		},
		"materialized departures": func(routeLimit *int) sq.SelectBuilder {
			return stopDeparturesMaterializedSelect(fvid, []int{stopID}, stopTimeEntityStop, nil, &model.StopTimeFilter{ServiceDate: &date, RouteLimit: routeLimit})
		},
	}
	// Each route's departure times, in order
	routeDepartures := func(t *testing.T, q sq.SelectBuilder) map[int][]string {
		var sts []*model.StopTime
		require.NoError(t, dbutil.Select(ctx, db, q, &sts))
		ret := map[int][]string{}
		for _, st := range sts {
			route := tripRoutes[st.TripID.Int()]
			ret[route] = append(ret[route], st.DepartureTime.String())
		}
		return ret
	}
	for name, q := range queries {
		t.Run(name, func(t *testing.T) {
			all := routeDepartures(t, q(nil))
			expect := map[int][]string{}
			trimmed := 0
			for route, departures := range all {
				expect[route] = departures[:min(2, len(departures))]
				trimmed += len(departures) - len(expect[route])
			}
			// Several routes, with departures past the route limit to leave out
			require.Greater(t, len(all), 1)
			require.Greater(t, trimmed, 0)
			assert.Equal(t, expect, routeDepartures(t, q(ptr(2))))
		})
	}
}

// RouteLimit holds across the service days a window spans: a stop's departures
// keep each route's first stop times by when they happen, not each service
// day's first.
func TestStopTimesByStopIDs_RouteLimit(t *testing.T) {
	if a, ok := testutil.CheckTestDB(); !ok {
		t.Skip(a)
	}
	db := testutil.MustOpenTestDB(t)
	f := NewFinder(db)
	// BART's MacArthur, with trips after midnight
	var fvid, stopID int
	require.NoError(t, db.Get(&fvid, "select feed_versions.id from feed_versions join current_feeds on current_feeds.id = feed_versions.feed_id join feed_states on feed_states.feed_version_id = feed_versions.id where current_feeds.onestop_id = $1", "BA"))
	require.NoError(t, db.Get(&stopID, "select id from gtfs_stops where feed_version_id = $1 and stop_id = $2", fvid, "MCAR_S"))
	tripRoutes := tripRouteIDs(t, db, fvid)

	windows := []struct {
		name       string
		date       tt.Date
		start, end int
	}{
		{"into the next service day", tt.NewDate(time.Date(2018, 5, 30, 0, 0, 0, 0, time.UTC)), 23 * 3600, 30 * 3600},
		{"back into the previous service day", tt.NewDate(time.Date(2018, 5, 31, 0, 0, 0, 0, time.UTC)), 15 * 60, 6 * 3600},
	}
	type departure struct {
		serviceDate   string
		departureTime string
		tripID        int
	}
	departureOf := func(st *model.StopTime) departure {
		return departure{st.ServiceDate.Format(time.DateOnly), st.DepartureTime.String(), st.TripID.Int()}
	}
	when := func(st *model.StopTime) int64 {
		return st.ServiceDate.Val.Unix() + int64(st.DepartureTime.Int())
	}
	for _, materialized := range []bool{false, true} {
		ctx := model.WithConfig(context.Background(), model.Config{UseMaterializedDepartures: materialized})
		for _, w := range windows {
			t.Run(fmt.Sprintf("%s, materialized=%t", w.name, materialized), func(t *testing.T) {
				find := func(routeLimit *int) []*model.StopTime {
					date := w.date
					where := &model.StopTimeFilter{Date: &date, StartTime: ptr(w.start), EndTime: ptr(w.end), RouteLimit: routeLimit}
					sts, err := f.StopTimesByStopIDs(ctx, nil, where, []model.FVPair{{FeedVersionID: fvid, EntityID: stopID}})
					require.NoError(t, err)
					return sts[0]
				}
				// Each route's earliest departure, and the service days its departures fall on
				earliest := map[int]*model.StopTime{}
				serviceDates := map[int]map[string]bool{}
				for _, st := range find(nil) {
					route := tripRoutes[st.TripID.Int()]
					if e, ok := earliest[route]; !ok || when(st) < when(e) {
						earliest[route] = st
					}
					if serviceDates[route] == nil {
						serviceDates[route] = map[string]bool{}
					}
					serviceDates[route][st.ServiceDate.Format(time.DateOnly)] = true
				}
				// Some route departs on both service days, so a limit per service day returns it twice
				spanning := 0
				for _, dates := range serviceDates {
					if len(dates) > 1 {
						spanning++
					}
				}
				require.Greater(t, spanning, 0)

				expect := map[int]departure{}
				for route, st := range earliest {
					expect[route] = departureOf(st)
				}
				got := map[int]departure{}
				for _, st := range find(ptr(1)) {
					route := tripRoutes[st.TripID.Int()]
					_, repeated := got[route]
					assert.False(t, repeated, "route %d returned twice", route)
					got[route] = departureOf(st)
				}
				assert.Equal(t, expect, got)
			})
		}
	}
}

// tripRouteIDs maps each trip in a feed version to its route.
func tripRouteIDs(t *testing.T, db *sqlx.DB, fvid int) map[int]int {
	var trips []struct {
		ID      int `db:"id"`
		RouteID int `db:"route_id"`
	}
	require.NoError(t, db.Select(&trips, "select id, route_id from gtfs_trips where feed_version_id = $1", fvid))
	ret := map[int]int{}
	for _, trip := range trips {
		ret[trip.ID] = trip.RouteID
	}
	return ret
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

// A departure of a frequency-based trip carries its instance's start and the
// shift from the trip's own stop times, on both departures queries. The
// "offset" trip's template isn't its pattern's representative, so the shift
// differs from the one the queries apply to the representative's times.
func TestStopDeparturesSelect_FrequencyInstances(t *testing.T) {
	ctx := context.Background()
	tx, err := testutil.MustOpenTestDB(t).Beginx()
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	fvid := testdb.CopyTestFeed(t, tx, testreader.NewFrequencyTestFeed())
	stop1 := 0
	require.NoError(t, tx.Get(&stop1, "select id from gtfs_stops where feed_version_id = $1 and stop_id = $2", fvid, "stop1"))
	// Each trip's own first departure.
	var tripIds []int
	require.NoError(t, tx.Select(&tripIds, "select id from gtfs_trips where feed_version_id = $1", fvid))
	firstDeparture := map[int]int64{}
	for _, id := range tripIds {
		var sts []*model.StopTime
		require.NoError(t, dbutil.Select(ctx, tx, stopTimeSelect([]model.FVPair{{FeedVersionID: fvid, EntityID: id}}, stopTimeEntityTrip, nil), &sts))
		require.NotEmpty(t, sts)
		firstDeparture[id] = sts[0].DepartureTime.Val
	}
	date := tt.NewDate(time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC))
	hm := func(h, m int) int64 { return int64(h*3600 + m*60) }
	for _, q := range []struct {
		name string
		q    sq.SelectBuilder
	}{
		{"live", stopDeparturesSelect(fvid, []int{stop1}, stopTimeEntityStop, nil, &model.StopTimeFilter{ServiceDate: &date})},
		{"materialized", stopDeparturesMaterializedSelect(fvid, []int{stop1}, stopTimeEntityStop, nil, &model.StopTimeFilter{ServiceDate: &date})},
	} {
		t.Run(q.name, func(t *testing.T) {
			var sts []*model.StopTime
			require.NoError(t, dbutil.Select(ctx, tx, q.q, &sts))
			starts := map[string][]int64{}
			for _, st := range sts {
				if !st.RunStart.Valid {
					assert.False(t, st.RunOffset.Valid, "%s has no offset", st.GtfsTripID)
					continue
				}
				starts[st.GtfsTripID] = append(starts[st.GtfsTripID], st.RunStart.Val)
				// The offset moves the trip's own stop times onto the instance,
				// whose first stop departs at its start.
				assert.Equal(t, st.RunStart.Val, firstDeparture[st.TripID.Int()]+st.RunOffset.Val, "%s at %s: first departure plus offset", st.GtfsTripID, st.RunStart)
				assert.Equal(t, st.RunStart.Val, st.DepartureTime.Val, "%s at %s: departure", st.GtfsTripID, st.RunStart)
			}
			for _, v := range starts {
				slices.Sort(v)
			}
			assert.Equal(t, map[string][]int64{
				"offset": {hm(7, 0), hm(7, 30)},
				"meet":   {hm(10, 0), hm(10, 30), hm(11, 0), hm(11, 30)},
			}, starts)
		})
	}
}
