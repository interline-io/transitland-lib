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
					alerts := st.Get("trip.route.alerts").Array()
					if len(alerts) != 2 {
						t.Errorf("got %d alerts, expected 2", len(alerts))
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
					if len(alerts) == 1 {
						firstAlert := alerts[0]
						assert.Equal(t, "Test route header - active", firstAlert.Get("header_text.0.text").String(), "header_text.0.text")
						assert.Contains(t, firstAlert.Get("description_text.0.text").String(), "route_id:05", "description_text.0.text")
					} else {
						t.Errorf("got %d alerts, expected 1", len(alerts))
					}
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

// An alert on a route at one stop is an alert about the route: Route.alerts
// returns it wherever the stop is, and Stop.alerts only at that stop.
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
			assert.ElementsMatch(t, []string{"Route 05 at Fruitvale", "Route 05 at 12th St"}, headers(st.Get("trip.route.alerts").Array()))
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
