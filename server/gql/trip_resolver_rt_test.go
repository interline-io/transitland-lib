package gql

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestTripRT_Alerts(t *testing.T) {
	activeVars := rtTestStopQueryVars()
	activeVars["active"] = true
	tcs := []rtTestCase{
		{
			name:  "trip alerts",
			query: rtTestStopQuery,
			vars:  rtTestStopQueryVars(),
			rtfiles: []testconfig.RTJsonFile{
				{Feed: "BA", Ftype: "realtime_alerts", Fname: "BA-alerts.json"},
			},
			cb: func(t *testing.T, jj string) {
				checkTrip := "1031527WKDY"
				found := false
				a := gjson.Get(jj, "stops.0.stop_times").Array()
				for _, st := range a {
					if st.Get("trip.trip_id").String() != checkTrip {
						continue
					}
					found = true
					alerts := st.Get("trip.alerts").Array()
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
			name:  "trip alerts active",
			query: rtTestStopQuery,
			vars:  activeVars,
			rtfiles: []testconfig.RTJsonFile{
				{Feed: "BA", Ftype: "realtime_alerts", Fname: "BA-alerts.json"},
			},
			cb: func(t *testing.T, jj string) {
				checkTrip := "1031527WKDY"
				found := false
				a := gjson.Get(jj, "stops.0.stop_times").Array()
				for _, st := range a {
					if st.Get("trip.trip_id").String() != checkTrip {
						continue
					}
					found = true
					alerts := st.Get("trip.alerts").Array()
					if len(alerts) == 1 {
						firstAlert := alerts[0]
						assert.Equal(t, "Test trip header - active", firstAlert.Get("header_text.0.text").String(), "header_text.0.text")
						assert.Contains(t, firstAlert.Get("description_text.0.text").String(), "trip_id:1031527WKDY", "description_text.0.text")
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

func TestTripRT_StopTimes(t *testing.T) {
	const tripRtQuery = `query($trip_id:String!) {
	trips(where: { trip_id: $trip_id }) {
	  id
	  trip_id
	  stop_times(limit:100) {
		stop_sequence
		arrival {
			scheduled
			estimated
			estimated_utc
			stop_timezone
			delay
			uncertainty
		}
		departure {
			scheduled
			estimated
			estimated_utc
			stop_timezone
			delay
			uncertainty
		}
	  }
	}
  }`

	tcs := []rtTestCase{
		{
			name:  "trip stop times",
			query: tripRtQuery,
			vars: hw{
				"trip_id": "1031527WKDY",
			},
			rtfiles: testconfig.DefaultRTJson(),
			cb: func(t *testing.T, jj string) {
				checkTrip := "1031527WKDY"
				trip := gjson.Get(jj, "trips.0")
				if trip.Get("trip_id").String() != checkTrip {
					t.Errorf("expected to find trip '%s'", checkTrip)
				}
				a := trip.Get("stop_times").Array()
				assert.Equal(t, 20, len(a))
				delay := 30
				for _, st := range a {
					assert.Equal(t, delay, int(st.Get("arrival.delay").Int()), "arrival.delay")
					assert.Equal(t, delay, int(st.Get("departure.delay").Int()), "departure.delay")
					sched, _ := tt.NewSecondsFromString(st.Get("arrival.scheduled").String())
					est, _ := tt.NewSecondsFromString(st.Get("arrival.estimated").String())
					assert.Equal(t, sched.Int()+delay, est.Int(), "arrival.scheduled + delay = arrival.estimated for this test")
				}
			},
		},
	}
	for _, tc := range tcs {
		testRt(t, tc)
	}
}
