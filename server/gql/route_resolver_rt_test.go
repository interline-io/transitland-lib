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
