package gql

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestRouteRT_Alerts(t *testing.T) {
	activeVars := rtTestStopQueryVars()
	activeVars["active"] = true
	tcs := []rtTestCase{
		{
			name:  "stop alerts active",
			query: rtTestStopQuery,
			vars:  rtTestStopQueryVars(),
			rtfiles: []testconfig.RTJsonFile{
				{Feed: "BA", Ftype: "realtime_alerts", Fname: "BA-alerts.json"},
			},
			cb: func(t *testing.T, jj string) {
				checkTrip := "1031527WKDY"
				sts := gjson.Get(jj, "stops.0.stop_times").Array()
				found := false
				for _, st := range sts {
					if st.Get("trip.trip_id").String() != checkTrip {
						continue
					}
					found = true
					assert.Equal(t, "05", st.Get("trip.route.route_id").String(), "trip.route.route_id")
					// The route's own two, and the two on one of its trips.
					alerts := st.Get("trip.route.alerts").Array()
					if len(alerts) != 4 {
						t.Errorf("got %d alerts, expected 4", len(alerts))
					}
				}
				if !found {
					t.Errorf("expected to find trip '%s'", checkTrip)
				}
			},
		},
		{
			name:  "stop alerts active",
			query: rtTestStopQuery,
			vars:  activeVars,
			rtfiles: []testconfig.RTJsonFile{
				{Feed: "BA", Ftype: "realtime_alerts", Fname: "BA-alerts.json"},
			},
			cb: func(t *testing.T, jj string) {
				checkTrip := "1031527WKDY"
				sts := gjson.Get(jj, "stops.0.stop_times").Array()
				found := false
				for _, st := range sts {
					if st.Get("trip.trip_id").String() != checkTrip {
						continue
					}
					found = true
					assert.Equal(t, "05", st.Get("trip.route.route_id").String(), "trip.route.route_id")
					alerts := st.Get("trip.route.alerts").Array()
					var headers []string
					for _, a := range alerts {
						headers = append(headers, a.Get("header_text.0.text").String())
					}
					assert.ElementsMatch(t, []string{"Test route header - active", "Test trip header - active"}, headers)
				}
				if !found {
					t.Errorf("expected to find trip '%s'", checkTrip)
				}
			},
		},
	}
	for _, tc := range tcs {
		testRt(t, tc)
	}

}

// Route.alerts returns an alert on the route at one stop wherever the stop is,
// and one on the route's mode; Stop.alerts only the alert at that stop.
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
	testRt(t, rtTestCase{
		name:    "route at stop",
		query:   rtTestStopQuery,
		vars:    rtTestStopQueryVars(),
		rtfiles: rtfiles,
		cb: func(t *testing.T, jj string) {
			stopAlerts := gjson.Get(jj, "stops.0.alerts").Array()
			assert.Equal(t, []string{"Route 05 at Fruitvale"}, headers(stopAlerts))
			if len(stopAlerts) == 1 {
				ie := stopAlerts[0].Get("informed_entity").Array()
				if assert.Len(t, ie, 1) {
					assert.Equal(t, "05", ie[0].Get("route_id").String())
					assert.Equal(t, "FTVL", ie[0].Get("stop_id").String())
				}
			}
			st := gjson.Get(jj, `stops.0.stop_times.#(trip.trip_id=="1031527WKDY")`)
			if !st.Exists() {
				t.Fatal("expected to find trip '1031527WKDY'")
			}
			assert.ElementsMatch(t, []string{"Route 05 at Fruitvale", "Route 05 at 12th St", "Trip 1031527WKDY", "All BART subway service"}, headers(st.Get("trip.route.alerts").Array()))
		},
	})
	// A selector narrowed to a mode is about that mode's routes, not the agency:
	// BART's subway alert reaches route 05, and neither mode alert reaches BART.
	testRt(t, rtTestCase{
		name:    "route type",
		query:   rtTestStopQuery,
		vars:    rtTestStopQueryVars(),
		rtfiles: rtfiles,
		cb: func(t *testing.T, jj string) {
			st := gjson.Get(jj, `stops.0.stop_times.#(trip.trip_id=="1031527WKDY")`)
			if !st.Exists() {
				t.Fatal("expected to find trip '1031527WKDY'")
			}
			assert.Contains(t, headers(st.Get("trip.route.alerts").Array()), "All BART subway service")
			assert.NotContains(t, headers(st.Get("trip.route.alerts").Array()), "All bus service")
			assert.Empty(t, headers(st.Get("trip.route.agency.alerts").Array()))
		},
	})
	testRt(t, rtTestCase{
		name:    "informed entity",
		query:   rtTestStopQuery,
		vars:    rtTestStopQueryVars(),
		rtfiles: rtfiles,
		cb: func(t *testing.T, jj string) {
			st := gjson.Get(jj, `stops.0.stop_times.#(trip.trip_id=="1031527WKDY")`)
			tripAlerts := st.Get("trip.alerts").Array()
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

// Route.alerts returns an alert on one of the route's trips, named by the trip's
// trip_id or route_id; Trip.alerts only those naming the trip itself, and
// Stop.alerts none of them.
func TestRouteRT_TripAlerts(t *testing.T) {
	headers := func(alerts []gjson.Result) []string {
		var ret []string
		for _, a := range alerts {
			ret = append(ret, a.Get("header_text.0.text").String())
		}
		return ret
	}
	testRt(t, rtTestCase{
		name:    "trip alerts",
		query:   rtTestStopQuery,
		vars:    rtTestStopQueryVars(),
		rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_alerts", Fname: "BA-alerts-trips.json"}},
		cb: func(t *testing.T, jj string) {
			route05 := gjson.Get(jj, `stops.0.stop_times.#(trip.trip_id=="1031527WKDY")`)
			route03 := gjson.Get(jj, `stops.0.stop_times.#(trip.trip_id=="2211533WKDY")`)
			if !route05.Exists() || !route03.Exists() {
				t.Fatal("expected to find trips '1031527WKDY' and '2211533WKDY'")
			}
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
}
