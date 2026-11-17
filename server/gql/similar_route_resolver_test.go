package gql

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/gql/internal/gqlcase"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// The fixtures have no route published by two feeds, so these tests make some:
// copies of Caltrain's Bullet route in HART's active feed version, inside the
// test's transaction. HART's agency runs on New York time and Caltrain's on Los
// Angeles time, so a copy of the same trips writes them three hours later. One
// copy runs the same trips, one about half of them, one the same trips seven
// minutes later, and one the same local times, which in New York are three
// hours earlier.

const (
	similarRouteSource = "Bu-130"
	// A Tuesday inside the Caltrain feed version's calendars.
	similarRouteDate = "2018-06-05"
	// New York's clock is three hours ahead of Los Angeles' on that date.
	similarRouteTzShift = 3 * 3600
)

// How cloneSimilarRoute copies a route.
type similarRouteCopy struct {
	// Prefixes the copy's route and trip ids.
	prefix string
	// Seconds added to every stop time.
	shift int
	// Copies every other trip only.
	half bool
	// Copies one direction's trips only.
	direction *int
}

// cloneSimilarRoute copies the route into the target feed version under the
// same Onestop ID, with its trips and their stop times as c says, and returns
// the copy's id. The copy's trips keep the source's calendars, shapes and stops,
// and point at the copies of their journey patterns' base trips.
func cloneSimilarRoute(t *testing.T, db sqlx.Ext, srcID, targetFv int, c similarRouteCopy) int {
	t.Helper()
	prefix, shift, half, direction := c.prefix, c.shift, c.half, -1
	if c.direction != nil {
		direction = *c.direction
	}
	var agencyID, newID int
	require.NoError(t, sqlx.Get(db, &agencyID, `select id from gtfs_agencies where feed_version_id = $1 order by id limit 1`, targetFv))
	require.NoError(t, sqlx.Get(db, &newID, `
		insert into gtfs_routes (route_id, route_short_name, route_long_name, route_desc, route_type, route_url, route_color, route_text_color, route_sort_order, feed_version_id, agency_id)
		select $1 || route_id, route_short_name, route_long_name, route_desc, route_type, route_url, route_color, route_text_color, route_sort_order, $2, $3
		from gtfs_routes where id = $4
		returning id`, prefix, targetFv, agencyID, srcID))
	_, err := db.Exec(`
		insert into feed_version_route_onestop_ids (feed_version_id, entity_id, onestop_id)
		select $1, copy.route_id, o.onestop_id
		from gtfs_routes copy, gtfs_routes src
		join feed_version_route_onestop_ids o on o.feed_version_id = src.feed_version_id and o.entity_id = src.route_id
		where copy.id = $2 and src.id = $3`, targetFv, newID, srcID)
	require.NoError(t, err)
	_, err = db.Exec(`
		insert into gtfs_trips (trip_id, trip_headsign, trip_short_name, direction_id, block_id, wheelchair_accessible, bikes_allowed, feed_version_id, route_id, shape_id, stop_pattern_id, service_id, journey_pattern_id, journey_pattern_offset, safe_duration_factor, safe_duration_offset, cars_allowed)
		select $1 || trip_id, trip_headsign, trip_short_name, direction_id, block_id, wheelchair_accessible, bikes_allowed, $2, $3, shape_id, stop_pattern_id, service_id, $1 || journey_pattern_id, journey_pattern_offset, safe_duration_factor, safe_duration_offset, cars_allowed
		from gtfs_trips src
		where route_id = $4 and ($6 < 0 or direction_id = $6) and (not $5 or id % 2 = 0 or exists (
			select 1 from gtfs_trips t where t.route_id = $4 and t.id % 2 = 0 and t.journey_pattern_id = src.trip_id
		))`, prefix, targetFv, newID, srcID, half, direction)
	require.NoError(t, err)
	_, err = db.Exec(`
		insert into gtfs_stop_times (feed_version_id, trip_id, stop_id, arrival_time, departure_time, stop_sequence, shape_dist_traveled, pickup_type, drop_off_type, timepoint, interpolated, stop_headsign)
		select $1, copy.id, st.stop_id, st.arrival_time + $2, st.departure_time + $2, st.stop_sequence, st.shape_dist_traveled, st.pickup_type, st.drop_off_type, st.timepoint, st.interpolated, st.stop_headsign
		from gtfs_stop_times st
		join gtfs_trips src on src.id = st.trip_id and src.feed_version_id = st.feed_version_id
		join gtfs_trips copy on copy.feed_version_id = $1 and copy.trip_id = $3 || src.trip_id
		where src.route_id = $4`, targetFv, shift, prefix, srcID)
	require.NoError(t, err)
	return newID
}

type similarRouteFixture struct {
	srcID, exactID, halfID, shiftedID, sameLocalID int
	srcFv, haFv, haFeedID                          int
}

// The id of the route in its feed's active feed version.
func activeRouteID(t *testing.T, db sqlx.Ext, routeID string) int {
	t.Helper()
	var id int
	require.NoError(t, sqlx.Get(db, &id, `select gtfs_routes.id from gtfs_routes join feed_states using(feed_version_id) where route_id = $1`, routeID))
	return id
}

// The active feed version of the feed.
func activeFeedVersionID(t *testing.T, db sqlx.Ext, feedOnestopID string) int {
	t.Helper()
	var id int
	require.NoError(t, sqlx.Get(db, &id, `select feed_states.feed_version_id from feed_states join current_feeds on current_feeds.id = feed_states.feed_id where current_feeds.onestop_id = $1`, feedOnestopID))
	return id
}

func makeSimilarRoutes(t *testing.T, cfg model.Config) similarRouteFixture {
	t.Helper()
	db := cfg.Adapter.DBX()
	var fx similarRouteFixture
	fx.srcID = activeRouteID(t, db, similarRouteSource)
	require.NoError(t, sqlx.Get(db, &fx.srcFv, `select feed_version_id from gtfs_routes where id = $1`, fx.srcID))
	fx.haFv = activeFeedVersionID(t, db, "HA")
	require.NoError(t, sqlx.Get(db, &fx.haFeedID, `select feed_id from feed_versions where id = $1`, fx.haFv))
	tz := similarRouteTzShift
	fx.exactID = cloneSimilarRoute(t, db, fx.srcID, fx.haFv, similarRouteCopy{prefix: "similar-exact-", shift: tz})
	fx.halfID = cloneSimilarRoute(t, db, fx.srcID, fx.haFv, similarRouteCopy{prefix: "similar-half-", shift: tz, half: true})
	fx.shiftedID = cloneSimilarRoute(t, db, fx.srcID, fx.haFv, similarRouteCopy{prefix: "similar-shifted-", shift: tz + 7*60})
	fx.sameLocalID = cloneSimilarRoute(t, db, fx.srcID, fx.haFv, similarRouteCopy{prefix: "similar-samelocal-"})
	return fx
}

func similarRouteDates(t *testing.T) []*tt.Date {
	d, err := tt.ParseDate(similarRouteDate)
	require.NoError(t, err)
	return []*tt.Date{&d}
}

// A finder query with stop correspondences, which the resolver asks for when
// the query selects them.
func similarQuery(where *model.SimilarRouteFilter) model.SimilarRouteQuery {
	return model.SimilarRouteQuery{Where: where, StopCorrespondences: true}
}

// Candidates by their route id.
func similarByOther(srs []*model.SimilarRoute) map[int]*model.SimilarRoute {
	ret := map[int]*model.SimilarRoute{}
	for _, sr := range srs {
		ret[sr.OtherRouteID] = sr
	}
	return ret
}

func TestSimilarRoutesByRouteIDs(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
		ctx := model.WithConfig(context.Background(), cfg)
		fx := makeSimilarRoutes(t, cfg)
		f := cfg.Finder

		t.Run("candidates without dates", func(t *testing.T) {
			got, err := f.SimilarRoutesByRouteIDs(ctx, nil, similarQuery(nil), []int{fx.srcID})
			require.NoError(t, err)
			require.Len(t, got, 1)
			cands := similarByOther(got[0])
			assert.Len(t, cands, 4)
			for _, id := range []int{fx.exactID, fx.halfID, fx.shiftedID, fx.sameLocalID} {
				if assert.Contains(t, cands, id) {
					sr := cands[id]
					assert.True(t, sr.SameOnestopID)
					assert.True(t, sr.RouteTypeBasicMatch)
					assert.Nil(t, sr.Trips)
					assert.Nil(t, sr.Relationship)
					assert.Nil(t, sr.StopCorrespondences)
				}
			}
		})

		t.Run("stop correspondences only when asked", func(t *testing.T) {
			got, err := f.SimilarRoutesByRouteIDs(ctx, nil, model.SimilarRouteQuery{Where: &model.SimilarRouteFilter{ServiceDates: similarRouteDates(t)}}, []int{fx.srcID})
			require.NoError(t, err)
			exact := similarByOther(got[0])[fx.exactID]
			require.NotNil(t, exact)
			assert.Equal(t, model.RouteTripRelationshipSameTrips, *exact.Relationship)
			assert.Nil(t, exact.StopCorrespondences)
		})

		t.Run("trips compared on a date", func(t *testing.T) {
			got, err := f.SimilarRoutesByRouteIDs(ctx, nil, similarQuery(&model.SimilarRouteFilter{ServiceDates: similarRouteDates(t)}), []int{fx.srcID})
			require.NoError(t, err)
			require.Len(t, got[0], 4)
			// Most matched trips first.
			assert.Equal(t, fx.exactID, got[0][0].OtherRouteID)
			cands := similarByOther(got[0])

			exact := cands[fx.exactID]
			require.NotNil(t, exact.Trips)
			assert.Greater(t, *exact.Trips, 0)
			assert.Equal(t, *exact.Trips, *exact.OtherTrips)
			assert.Equal(t, *exact.Trips, *exact.MatchedTrips)
			assert.Equal(t, model.RouteTripRelationshipSameTrips, *exact.Relationship)
			// The copy calls at the same stops, so each pairs with itself.
			require.NotEmpty(t, exact.StopCorrespondences)
			for _, c := range exact.StopCorrespondences {
				assert.Equal(t, c.StopID, c.OtherStopID)
				assert.InDelta(t, 0, c.Distance, 0.001)
				assert.Greater(t, c.MatchedTrips, 0)
			}

			half := cands[fx.halfID]
			require.NotNil(t, half.OtherTrips)
			assert.Greater(t, *half.OtherTrips, 0)
			assert.Less(t, *half.OtherTrips, *half.Trips)
			assert.Equal(t, *half.OtherTrips, *half.MatchedTrips)
			assert.Equal(t, model.RouteTripRelationshipSuperset, *half.Relationship)

			shifted := cands[fx.shiftedID]
			assert.Equal(t, 0, *shifted.MatchedTrips)
			assert.Equal(t, model.RouteTripRelationshipDisjoint, *shifted.Relationship)
			assert.Empty(t, shifted.StopCorrespondences)

			// The same local times in another timezone are other moments.
			sameLocal := cands[fx.sameLocalID]
			assert.Equal(t, 0, *sameLocal.MatchedTrips)
			assert.Equal(t, model.RouteTripRelationshipDisjoint, *sameLocal.Relationship)
		})

		t.Run("journey patterns deduplicated at import", func(t *testing.T) {
			filter := &model.SimilarRouteFilter{ServiceDates: similarRouteDates(t)}
			before, err := f.SimilarRoutesByRouteIDs(ctx, nil, similarQuery(filter), []int{fx.srcID})
			require.NoError(t, err)
			// An import that deduplicates journey patterns keeps stop times only
			// for each pattern's base trip; do the same to both feeds.
			res, err := cfg.Adapter.DBX().Exec(`
				delete from gtfs_stop_times st using gtfs_trips t
				where st.trip_id = t.id and st.feed_version_id = t.feed_version_id
					and t.feed_version_id = any($1) and t.journey_pattern_id <> t.trip_id`, []int{fx.srcFv, fx.haFv})
			require.NoError(t, err)
			n, err := res.RowsAffected()
			require.NoError(t, err)
			require.Greater(t, n, int64(0), "no trips share a journey pattern, so the test checks nothing")
			after, err := f.SimilarRoutesByRouteIDs(ctx, nil, similarQuery(filter), []int{fx.srcID})
			require.NoError(t, err)
			b, a := similarByOther(before[0]), similarByOther(after[0])
			for _, id := range []int{fx.exactID, fx.halfID, fx.shiftedID} {
				assert.Equal(t, *b[id].Trips, *a[id].Trips)
				assert.Equal(t, *b[id].OtherTrips, *a[id].OtherTrips)
				assert.Equal(t, *b[id].MatchedTrips, *a[id].MatchedTrips)
				assert.Equal(t, *b[id].Relationship, *a[id].Relationship)
				assert.Equal(t, len(b[id].StopCorrespondences), len(a[id].StopCorrespondences))
			}
		})

		t.Run("from the copy's side", func(t *testing.T) {
			got, err := f.SimilarRoutesByRouteIDs(ctx, nil, similarQuery(&model.SimilarRouteFilter{ServiceDates: similarRouteDates(t)}), []int{fx.halfID})
			require.NoError(t, err)
			// The other copies are in the same feed, so only the source is a candidate.
			require.Len(t, got[0], 1)
			assert.Equal(t, fx.srcID, got[0][0].OtherRouteID)
			assert.Equal(t, model.RouteTripRelationshipSubset, *got[0][0].Relationship)

			all, err := f.SimilarRoutesByRouteIDs(ctx, nil, similarQuery(&model.SimilarRouteFilter{OtherFeedsOnly: ptr(false)}), []int{fx.halfID})
			require.NoError(t, err)
			assert.Len(t, all[0], 4)
		})

		t.Run("a date with no service", func(t *testing.T) {
			d, err := tt.ParseDate("2030-01-01")
			require.NoError(t, err)
			got, err := f.SimilarRoutesByRouteIDs(ctx, nil, similarQuery(&model.SimilarRouteFilter{ServiceDates: []*tt.Date{&d}}), []int{fx.srcID})
			require.NoError(t, err)
			for _, sr := range got[0] {
				assert.Equal(t, 0, *sr.Trips)
				assert.Equal(t, model.RouteTripRelationshipNoTrips, *sr.Relationship)
			}
		})

		t.Run("limit and batching", func(t *testing.T) {
			one := 1
			got, err := f.SimilarRoutesByRouteIDs(ctx, &one, similarQuery(&model.SimilarRouteFilter{ServiceDates: similarRouteDates(t)}), []int{fx.srcID, fx.exactID})
			require.NoError(t, err)
			require.Len(t, got, 2)
			if assert.Len(t, got[0], 1) {
				assert.Equal(t, fx.exactID, got[0][0].OtherRouteID)
			}
			if assert.Len(t, got[1], 1) {
				assert.Equal(t, fx.srcID, got[1][0].OtherRouteID)
			}
		})

		t.Run("a route with no copies", func(t *testing.T) {
			otherID := activeRouteID(t, cfg.Adapter.DBX(), "Lo-130")
			got, err := f.SimilarRoutesByRouteIDs(ctx, nil, similarQuery(nil), []int{otherID})
			require.NoError(t, err)
			assert.Empty(t, got[0])
		})

		t.Run("private feeds", func(t *testing.T) {
			got, err := f.SimilarRoutesByRouteIDs(ctx, nil, similarQuery(nil), []int{fx.srcID})
			require.NoError(t, err)
			require.NotEmpty(t, got[0], "no candidates while the feeds are public")
			_, err = cfg.Adapter.DBX().Exec(`update feed_states set public = false where feed_id = $1`, fx.haFeedID)
			require.NoError(t, err)
			got, err = f.SimilarRoutesByRouteIDs(ctx, nil, similarQuery(nil), []int{fx.srcID})
			require.NoError(t, err)
			assert.Empty(t, got[0], "expected no candidates from a private feed")
		})
	})
}

func TestSimilarRoutesSplitRows(t *testing.T) {
	// A feed that splits the route into one row per direction, as feeds split
	// a line by branch: neither row covers the route, but the two together do.
	testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
		ctx := model.WithConfig(context.Background(), cfg)
		db := cfg.Adapter.DBX()
		srcID := activeRouteID(t, db, similarRouteSource)
		baFv := activeFeedVersionID(t, db, "BA")
		// BART's agency, like Caltrain's, runs on Los Angeles time.
		dir0 := cloneSimilarRoute(t, db, srcID, baFv, similarRouteCopy{prefix: "similar-dir0-", direction: ptr(0)})
		dir1 := cloneSimilarRoute(t, db, srcID, baFv, similarRouteCopy{prefix: "similar-dir1-", direction: ptr(1)})
		filter := &model.SimilarRouteFilter{ServiceDates: similarRouteDates(t)}

		got, err := cfg.Finder.SimilarRoutesByRouteIDs(ctx, nil, similarQuery(filter), []int{srcID})
		require.NoError(t, err)
		rows := similarByOther(got[0])
		require.Contains(t, rows, dir0)
		require.Contains(t, rows, dir1)
		a, b := rows[dir0], rows[dir1]
		require.Greater(t, *a.OtherTrips, 0)
		require.Greater(t, *b.OtherTrips, 0)
		// Each row alone: its trips run in the route, which runs more.
		assert.Equal(t, model.RouteTripRelationshipSuperset, *a.Relationship)
		assert.Equal(t, model.RouteTripRelationshipSuperset, *b.Relationship)
		// Both rows together are the route, and each row reports it.
		for _, sr := range []*model.SimilarRoute{a, b} {
			assert.Equal(t, *a.OtherTrips+*b.OtherTrips, *sr.FeedTrips)
			assert.Equal(t, *sr.Trips, *sr.FeedTrips)
			assert.Equal(t, *sr.Trips, *sr.FeedMatchedTrips)
			assert.Equal(t, model.RouteTripRelationshipSameTrips, *sr.FeedRelationship)
		}

		// From a row's side, the other feed has one row, so the feed fields
		// agree with the pair's.
		back, err := cfg.Finder.SimilarRoutesByRouteIDs(ctx, nil, similarQuery(filter), []int{dir0})
		require.NoError(t, err)
		require.Len(t, back[0], 1)
		assert.Equal(t, model.RouteTripRelationshipSubset, *back[0][0].Relationship)
		assert.Equal(t, model.RouteTripRelationshipSubset, *back[0][0].FeedRelationship)
		assert.Equal(t, *back[0][0].MatchedTrips, *back[0][0].FeedMatchedTrips)

		// Without dates the feed fields are null, like the others.
		none, err := cfg.Finder.SimilarRoutesByRouteIDs(ctx, nil, similarQuery(nil), []int{srcID})
		require.NoError(t, err)
		for _, sr := range none[0] {
			assert.Nil(t, sr.FeedTrips)
			assert.Nil(t, sr.FeedRelationship)
		}
	})
}

func TestSimilarRoutesRelativeDates(t *testing.T) {
	t.Run("each route compares on its own feed version's date", func(t *testing.T) {
		// 22:30 Monday in Los Angeles, Caltrain's feed version, is 01:30
		// Tuesday in New York, HART's, so TODAY is a different date for each
		// route in the batch.
		testconfig.ConfigTxRollback(t, testconfig.Options{WhenUtc: "2018-06-05T05:30:00Z"}, func(cfg model.Config) {
			ctx := model.WithConfig(context.Background(), cfg)
			fx := makeSimilarRoutes(t, cfg)
			f := cfg.Finder
			today := model.RelativeDateToday
			got, err := f.SimilarRoutesByRouteIDs(ctx, nil, similarQuery(&model.SimilarRouteFilter{RelativeDate: &today}), []int{fx.srcID, fx.exactID})
			require.NoError(t, err)
			onDate := func(routeID int, date string) *model.SimilarRoute {
				d, err := tt.ParseDate(date)
				require.NoError(t, err)
				one, err := f.SimilarRoutesByRouteIDs(ctx, nil, similarQuery(&model.SimilarRouteFilter{ServiceDates: []*tt.Date{&d}}), []int{routeID})
				require.NoError(t, err)
				for _, sr := range one[0] {
					if sr.OtherRouteID == fx.exactID || sr.OtherRouteID == fx.srcID {
						return sr
					}
				}
				t.Fatalf("no candidate for route %d on %s", routeID, date)
				return nil
			}
			monday := onDate(fx.srcID, "2018-06-04")
			tuesday := onDate(fx.exactID, "2018-06-05")
			fromSrc, fromCopy := similarByOther(got[0])[fx.exactID], similarByOther(got[1])[fx.srcID]
			require.NotNil(t, fromSrc)
			require.NotNil(t, fromCopy)
			assert.Equal(t, *monday.Trips, *fromSrc.Trips)
			assert.Equal(t, *monday.MatchedTrips, *fromSrc.MatchedTrips)
			assert.Equal(t, *tuesday.Trips, *fromCopy.Trips)
			assert.Equal(t, *tuesday.MatchedTrips, *fromCopy.MatchedTrips)
		})
	})

	t.Run("a feed version with no service window", func(t *testing.T) {
		testconfig.ConfigTxRollback(t, testconfig.Options{WhenUtc: "2018-06-04T19:00:00Z"}, func(cfg model.Config) {
			ctx := model.WithConfig(context.Background(), cfg)
			fx := makeSimilarRoutes(t, cfg)
			_, err := cfg.Adapter.DBX().Exec(`delete from feed_version_service_windows where feed_version_id = $1`, fx.srcFv)
			require.NoError(t, err)
			tomorrow := model.RelativeDateTomorrow
			got, err := cfg.Finder.SimilarRoutesByRouteIDs(ctx, nil, similarQuery(&model.SimilarRouteFilter{RelativeDate: &tomorrow}), []int{fx.srcID})
			require.NoError(t, err)
			want, err := cfg.Finder.SimilarRoutesByRouteIDs(ctx, nil, similarQuery(&model.SimilarRouteFilter{ServiceDates: similarRouteDates(t)}), []int{fx.srcID})
			require.NoError(t, err)
			require.Len(t, got[0], len(want[0]))
			for i := range want[0] {
				assert.Equal(t, *want[0][i].Trips, *got[0][i].Trips)
				assert.Equal(t, *want[0][i].MatchedTrips, *got[0][i].MatchedTrips)
			}
		})
	})
}

func TestSimilarRoutesResolver(t *testing.T) {
	// Monday the day before similarRouteDate, so TOMORROW resolves to it.
	testconfig.ConfigTxRollback(t, testconfig.Options{WhenUtc: "2018-06-04T19:00:00Z"}, func(cfg model.Config) {
		fx := makeSimilarRoutes(t, cfg)
		srv := model.AddConfigAndPerms(cfg, NewDefaultHandler())
		srv = usercheck.NewUserDefaultMiddleware(func() authn.User {
			return authn.NewCtxUser("testuser", "", "").WithRoles("testrole")
		})(srv)
		c := client.New(srv)
		// The test's transaction is one connection, and gqlgen resolves sibling
		// fields at once, so each query loads through one loader per level:
		// candidate routes in one, correspondence stops in the other.
		qRoutes := `query($ids: [Int!], $where: SimilarRouteFilter, $limit: Int) {
			routes(ids: $ids) {
				similar_routes(where: $where, limit: $limit) {
					route { route_id }
					same_onestop_id
					route_type_basic_match
					trips
					other_trips
					matched_trips
					relationship
				}
			}
		}`
		qStops := `query($ids: [Int!], $where: SimilarRouteFilter) {
			routes(ids: $ids) {
				similar_routes(where: $where) {
					relationship
					stop_correspondences { stop { stop_id } other_stop { stop_id } matched_trips distance }
				}
			}
		}`
		postQuery := func(q string, where any, limit any) (string, error) {
			resp := map[string]any{}
			vars := []client.Option{client.Var("ids", []int{fx.srcID}), client.Var("where", where)}
			if limit != nil {
				vars = append(vars, client.Var("limit", limit))
			}
			err := c.Post(q, &resp, vars...)
			return gqlcase.ToJSON(resp), err
		}

		t.Run("service dates", func(t *testing.T) {
			js, err := postQuery(qRoutes, map[string]any{"service_dates": []string{similarRouteDate}}, nil)
			require.NoError(t, err)
			assert.Equal(t, int64(4), gjson.Get(js, "routes.0.similar_routes.#").Int())
			first := gjson.Get(js, "routes.0.similar_routes.0")
			assert.Equal(t, "similar-exact-"+similarRouteSource, first.Get("route.route_id").String())
			assert.Equal(t, "SAME_TRIPS", first.Get("relationship").String())
			js, err = postQuery(qStops, map[string]any{"service_dates": []string{similarRouteDate}}, nil)
			require.NoError(t, err)
			sc := gjson.Get(js, "routes.0.similar_routes.0.stop_correspondences.0")
			assert.NotEmpty(t, sc.Get("stop.stop_id").String())
			assert.Equal(t, sc.Get("stop.stop_id").String(), sc.Get("other_stop.stop_id").String())
		})

		t.Run("relative date", func(t *testing.T) {
			viaDate, err := postQuery(qRoutes, map[string]any{"service_dates": []string{similarRouteDate}}, nil)
			require.NoError(t, err)
			viaRelative, err := postQuery(qRoutes, map[string]any{"relative_date": "TOMORROW", "days": 1}, nil)
			require.NoError(t, err)
			assert.JSONEq(t, gjson.Get(viaDate, "routes.0.similar_routes").Raw, gjson.Get(viaRelative, "routes.0.similar_routes").Raw)
		})

		t.Run("no dates", func(t *testing.T) {
			js, err := postQuery(qRoutes, nil, 1)
			require.NoError(t, err)
			assert.Equal(t, int64(1), gjson.Get(js, "routes.0.similar_routes.#").Int())
			assert.Equal(t, "null", gjson.Get(js, "routes.0.similar_routes.0.relationship").Raw)
			js, err = postQuery(qStops, nil, nil)
			require.NoError(t, err)
			assert.Equal(t, "null", gjson.Get(js, "routes.0.similar_routes.0.stop_correspondences").Raw)
		})

		t.Run("too many dates", func(t *testing.T) {
			dates := []string{"2018-06-04", "2018-06-05", "2018-06-06", "2018-06-07", "2018-06-08", "2018-06-09", "2018-06-10", "2018-06-11"}
			_, err := postQuery(qRoutes, map[string]any{"service_dates": dates}, nil)
			assert.ErrorContains(t, err, "at most 7 service_dates")
			_, err = postQuery(qRoutes, map[string]any{"relative_date": "TODAY", "days": 0}, nil)
			assert.ErrorContains(t, err, "days must be 1 to 7")
		})
	})
}
