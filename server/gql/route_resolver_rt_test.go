package gql

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestRouteRT_Alerts(t *testing.T) {
	tripVars := rtTestStopQueryVars()
	tripVars["include_route_trips"] = true
	activeVars := rtTestStopQueryVars()
	activeVars["active"] = true
	activeVars["include_route_trips"] = true
	routeAlerts := func(t *testing.T, jj string) []gjson.Result {
		st := gjson.Get(jj, `stops.0.stop_times.#(trip.trip_id=="1031527WKDY")`)
		if !st.Exists() {
			t.Fatal("expected to find trip '1031527WKDY'")
		}
		assert.Equal(t, "05", st.Get("trip.route.route_id").String(), "trip.route.route_id")
		return st.Get("trip.route.alerts").Array()
	}
	tcs := []rtTestCase{
		{
			// The route's own two; the two on one of its trips only with include_trips.
			name:    "route alerts",
			query:   rtTestStopQuery,
			vars:    rtTestStopQueryVars(),
			rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_alerts", Fname: "BA-alerts.json"}},
			cb: func(t *testing.T, jj string) {
				assert.Len(t, routeAlerts(t, jj), 2)
			},
		},
		{
			name:    "route alerts with trips",
			query:   rtTestStopQuery,
			vars:    tripVars,
			rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_alerts", Fname: "BA-alerts.json"}},
			cb: func(t *testing.T, jj string) {
				assert.Len(t, routeAlerts(t, jj), 4)
			},
		},
		{
			name:    "route alerts active",
			query:   rtTestStopQuery,
			vars:    activeVars,
			rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_alerts", Fname: "BA-alerts.json"}},
			cb: func(t *testing.T, jj string) {
				var headers []string
				for _, a := range routeAlerts(t, jj) {
					headers = append(headers, a.Get("header_text.0.text").String())
				}
				assert.ElementsMatch(t, []string{"Test route header - active", "Test trip header - active"}, headers)
			},
		},
	}
	for _, tc := range tcs {
		testRt(t, tc)
	}
}

// Route.alerts returns an alert on the route at one stop wherever the stop is,
// and with include_modes and include_trips one on its mode and one on one of its
// trips, but never one on another agency's route with the same route_id.
// Stop.alerts and Trip.alerts return only the alerts on that stop or trip, and
// none whose agency or route is another's.
func TestRouteRT_AlertsAtStop(t *testing.T) {
	rtfiles := []testconfig.RTJsonFile{
		{Feed: "BA", Ftype: "realtime_alerts", Fname: "BA-alerts-informed-entity.json"},
	}
	headers := func(alerts []gjson.Result) []string {
		var ret []string
		for _, a := range alerts {
			ret = append(ret, a.Get("header_text.0.text").String())
		}
		return ret
	}
	tripAt := func(t *testing.T, jj string) gjson.Result {
		st := gjson.Get(jj, `stops.0.stop_times.#(trip.trip_id=="1031527WKDY")`)
		if !st.Exists() {
			t.Fatal("expected to find trip '1031527WKDY'")
		}
		return st
	}
	flagVars := rtTestStopQueryVars()
	flagVars["include_route_modes"] = true
	flagVars["include_route_trips"] = true
	testRt(t, rtTestCase{
		name:    "route at stop",
		query:   rtTestStopQuery,
		vars:    rtTestStopQueryVars(),
		rtfiles: rtfiles,
		cb: func(t *testing.T, jj string) {
			// Not another agency's alert at a stop with this stop_id, one naming a
			// route this feed version doesn't have, one on a mode it doesn't run,
			// nor one giving route 05 another mode.
			stopAlerts := gjson.Get(jj, "stops.0.alerts").Array()
			assert.ElementsMatch(t,
				[]string{"Route 05 at Fruitvale", "BART at Fruitvale", "Fruitvale with an empty trip descriptor", "Subway at Fruitvale"},
				headers(stopAlerts))
			ie := gjson.Get(jj, `stops.0.alerts.#(header_text.0.text=="Route 05 at Fruitvale").informed_entity`).Array()
			if assert.Len(t, ie, 1) {
				assert.Equal(t, "05", ie[0].Get("route_id").String())
				assert.Equal(t, "FTVL", ie[0].Get("stop_id").String())
			}
			assert.ElementsMatch(t, []string{"Route 05 at Fruitvale", "Route 05 at 12th St"}, headers(tripAt(t, jj).Get("trip.route.alerts").Array()))
		},
	})
	testRt(t, rtTestCase{
		name:    "route at stop with modes and trips",
		query:   rtTestStopQuery,
		vars:    flagVars,
		rtfiles: rtfiles,
		cb: func(t *testing.T, jj string) {
			assert.ElementsMatch(t, []string{"Route 05 at Fruitvale", "Route 05 at 12th St", "Trip 1031527WKDY", "All BART subway service"}, headers(tripAt(t, jj).Get("trip.route.alerts").Array()))
		},
	})
	// A selector narrowed to a mode is about that mode's routes, not the agency:
	// BART's subway alert reaches route 05, and neither mode alert reaches BART.
	modeVars := rtTestStopQueryVars()
	modeVars["include_route_modes"] = true
	testRt(t, rtTestCase{
		name:    "route type",
		query:   rtTestStopQuery,
		vars:    modeVars,
		rtfiles: rtfiles,
		cb: func(t *testing.T, jj string) {
			st := tripAt(t, jj)
			assert.Contains(t, headers(st.Get("trip.route.alerts").Array()), "All BART subway service")
			assert.NotContains(t, headers(st.Get("trip.route.alerts").Array()), "All bus service")
			assert.Empty(t, headers(st.Get("trip.route.agency.alerts").Array()))
		},
	})
	// An alert reached through several entities carries the same entity id from
	// each, whether it came through a stop, a route, a trip or an agency.
	idVars := rtTestStopQueryVars()
	idVars["include_modes"] = true
	idVars["include_route_trips"] = true
	testRt(t, rtTestCase{
		name:    "entity id",
		query:   rtTestStopQuery,
		vars:    idVars,
		rtfiles: rtfiles,
		cb: func(t *testing.T, jj string) {
			st := tripAt(t, jj)
			check := func(path string, a gjson.Result, entityId string) {
				assert.Equal(t, entityId, a.Get("entity_id").String(), path)
				assert.Equal(t, "BA", a.Get("rt_feed_onestop_id").String(), path)
			}
			check("stop", gjson.Get(jj, "stops.0.alerts.0"), "route-at-ftvl")
			check("route at stop", st.Get(`trip.route.alerts.#(header_text.0.text=="Route 05 at Fruitvale")`), "route-at-ftvl")
			check("trip", st.Get("trip.alerts.0"), "trip")
			check("route trip", st.Get(`trip.route.alerts.#(header_text.0.text=="Trip 1031527WKDY")`), "trip")
			check("agency mode", st.Get("trip.route.agency.alerts.0"), "mode-subway")
		},
	})
	testRt(t, rtTestCase{
		name:    "informed entity",
		query:   rtTestStopQuery,
		vars:    rtTestStopQueryVars(),
		rtfiles: rtfiles,
		cb: func(t *testing.T, jj string) {
			// Not the alerts naming this trip_id with another agency or route.
			tripAlerts := tripAt(t, jj).Get("trip.alerts").Array()
			if !assert.Len(t, tripAlerts, 1) {
				return
			}
			ie := tripAlerts[0].Get("informed_entity.0")
			assert.Equal(t, "BART", ie.Get("agency_id").String())
			assert.Equal(t, "05", ie.Get("route_id").String())
			assert.Equal(t, int64(1), ie.Get("route_type").Int())
			assert.Equal(t, int64(1), ie.Get("direction_id").Int())
			assert.Equal(t, "1031527WKDY", ie.Get("trip.trip_id").String())
			assert.Equal(t, "2018-05-30", ie.Get("trip.start_date").String())
			assert.Empty(t, ie.Get("stop_id").String(), "stop_id")
		},
	})
}

// With include_trips, Route.alerts returns an alert on one of the route's trips,
// but not one on another agency's; without it, none of them. Trip.alerts returns
// only those naming the trip itself, and Stop.alerts none of them.
func TestRouteRT_TripAlerts(t *testing.T) {
	rtfiles := []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_alerts", Fname: "BA-alerts-trips.json"}}
	headers := func(alerts []gjson.Result) []string {
		var ret []string
		for _, a := range alerts {
			ret = append(ret, a.Get("header_text.0.text").String())
		}
		return ret
	}
	trips := func(t *testing.T, jj string) (gjson.Result, gjson.Result) {
		route05 := gjson.Get(jj, `stops.0.stop_times.#(trip.trip_id=="1031527WKDY")`)
		route03 := gjson.Get(jj, `stops.0.stop_times.#(trip.trip_id=="2211533WKDY")`)
		if !route05.Exists() || !route03.Exists() {
			t.Fatal("expected to find trips '1031527WKDY' and '2211533WKDY'")
		}
		return route05, route03
	}
	tripVars := rtTestStopQueryVars()
	tripVars["include_route_trips"] = true
	// The departures are the 2018-05-30 run, read that afternoon: the alerts
	// name no start_date, so they are on the trips' current runs.
	testRt(t, rtTestCase{
		name:    "trip alerts",
		query:   rtTestStopQuery,
		vars:    tripVars,
		rtfiles: rtfiles,
		whenUtc: rtFixtureWhenUtc,
		cb: func(t *testing.T, jj string) {
			route05, route03 := trips(t, jj)
			assert.ElementsMatch(t,
				[]string{"Trip 1031527WKDY", "Route 05 by trip descriptor", "Trip 1031527WKDY at Fruitvale"},
				headers(route05.Get("trip.route.alerts").Array()))
			// A trip entity with a route_type is about that trip, not the mode.
			assert.ElementsMatch(t,
				[]string{"Trip 2211533WKDY", "Subway trip 2211533WKDY"},
				headers(route03.Get("trip.route.alerts").Array()))
			assert.ElementsMatch(t,
				[]string{"Trip 1031527WKDY", "Trip 1031527WKDY at Fruitvale"},
				headers(route05.Get("trip.alerts").Array()))
			assert.Empty(t, headers(gjson.Get(jj, "stops.0.alerts").Array()))
			assert.Empty(t, headers(route05.Get("trip.route.agency.alerts").Array()))
		},
	})
	testRt(t, rtTestCase{
		name:    "trip alerts without include_trips",
		query:   rtTestStopQuery,
		vars:    rtTestStopQueryVars(),
		rtfiles: rtfiles,
		whenUtc: rtFixtureWhenUtc,
		cb: func(t *testing.T, jj string) {
			// A trip descriptor naming only the route picks out no trip: it is the route.
			route05, route03 := trips(t, jj)
			assert.Equal(t, []string{"Route 05 by trip descriptor"}, headers(route05.Get("trip.route.alerts").Array()))
			assert.Empty(t, headers(route03.Get("trip.route.alerts").Array()))
			assert.ElementsMatch(t,
				[]string{"Trip 1031527WKDY", "Trip 1031527WKDY at Fruitvale"},
				headers(route05.Get("trip.alerts").Array()))
		},
	})
}
