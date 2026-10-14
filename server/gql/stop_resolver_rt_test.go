package gql

import (
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestStopRT_Basic(t *testing.T) {
	tc := rtTestCase{
		name:    "stop times basic",
		query:   rtTestStopQuery,
		vars:    rtTestStopQueryVars(),
		rtfiles: testconfig.DefaultRTJson(),
		whenUtc: rtFixtureWhenUtc,
		cb: func(t *testing.T, jj string) {
			// A little more explicit version of the string check test
			a := gjson.Get(jj, "stops.0.stop_times").Array()
			delay := 30
			assert.Equal(t, 3, len(a))
			for _, st := range a {
				assert.Equal(t, "America/Los_Angeles", st.Get("arrival.stop_timezone").String(), "arrival.stop_timezone")
				assert.Equal(t, delay, int(st.Get("arrival.delay").Int()), "arrival.delay")
				assert.Equal(t, delay, int(st.Get("arrival.estimated_delay").Int()), "arrival.estimated_delay")
				assert.Equal(t, "America/Los_Angeles", st.Get("departure.stop_timezone").String(), "departure.stop_timezone")
				assert.Equal(t, delay, int(st.Get("departure.delay").Int()), "departure.delay")
				assert.Equal(t, delay, int(st.Get("departure.estimated_delay").Int()), "departure.estimated_delay")
				sched, _ := tt.NewSecondsFromString(st.Get("arrival.scheduled").String())
				est, _ := tt.NewSecondsFromString(st.Get("arrival.estimated").String())
				assert.Equal(t, sched.Int()+delay, est.Int(), "arrival.scheduled + delay = arrival.estimated for this test")
			}
			checkTrip := "1031527WKDY"
			found := false
			for _, st := range a {
				if st.Get("trip.trip_id").String() != checkTrip {
					continue
				}
				found = true
				assert.Equal(t, checkTrip, st.Get("trip.trip_id").String())
				// The trip update has no timestamp of its own, so it takes the
				// header's: when the fixture was fetched.
				assert.Equal(t, rtFixtureWhenUtc, st.Get("trip.timestamp").String())
				// arrival
				assert.Equal(t, "16:02:30", st.Get("arrival.estimated").String())
				assert.Equal(t, "2018-05-30T16:02:30-07:00", st.Get("arrival.estimated_local").String())
				assert.Equal(t, "2018-05-30T23:02:30Z", st.Get("arrival.estimated_utc").String())
				assert.Equal(t, "16:02:00", st.Get("arrival.scheduled").String())
				assert.Equal(t, "2018-05-30T16:02:00-07:00", st.Get("arrival.scheduled_local").String())
				assert.Equal(t, "2018-05-30T23:02:00Z", st.Get("arrival.scheduled_utc").String())
				// departure
				assert.Equal(t, "16:02:30", st.Get("departure.estimated").String())
				assert.Equal(t, "2018-05-30T16:02:30-07:00", st.Get("departure.estimated_local").String())
				assert.Equal(t, "2018-05-30T23:02:30Z", st.Get("departure.estimated_utc").String())
				assert.Equal(t, "16:02:00", st.Get("departure.scheduled").String())
				assert.Equal(t, "2018-05-30T16:02:00-07:00", st.Get("departure.scheduled_local").String())
				assert.Equal(t, "2018-05-30T23:02:00Z", st.Get("departure.scheduled_utc").String())
			}
			if !found {
				t.Errorf("expected to find trip '%s'", checkTrip)
			}
		},
	}
	testRt(t, tc)
}

func TestStopRT_BeforeMidnight(t *testing.T) {
	tc := rtTestCase{
		name:  "just before midnight",
		query: rtTestStopQuery,
		vars: hw{
			"stop_id": "FTVL",
			"stf": hw{
				"date":  "2018-05-30",
				"start": "23:45:00",
				"end":   "24:05:00",
			},
		},
		rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_trip_updates", Fname: "BA-midnight.json"}},
		whenUtc: rtFixtureWhenUtc,
		cb: func(t *testing.T, jj string) {
			a := gjson.Get(jj, "stops.0.stop_times").Array()
			// before midnight trip
			checkTrip := "2292315WKDY"
			found := false
			for _, st := range a {
				if st.Get("trip.trip_id").String() != checkTrip {
					continue
				}
				found = true
				assert.Equal(t, checkTrip, st.Get("trip.trip_id").String())
				assert.Equal(t, "23:55:00", st.Get("arrival.estimated").String())
				assert.Equal(t, "2018-05-30T23:55:00-07:00", st.Get("arrival.estimated_local").String())
				assert.Equal(t, "2018-05-31T06:55:00Z", st.Get("arrival.estimated_utc").String())
				assert.Equal(t, "23:48:00", st.Get("arrival.scheduled").String())
				assert.Equal(t, "2018-05-30T23:48:00-07:00", st.Get("arrival.scheduled_local").String())
				assert.Equal(t, "2018-05-31T06:48:00Z", st.Get("arrival.scheduled_utc").String())
			}
			if !found {
				t.Errorf("expected to find trip '%s'", checkTrip)
			}
		},
	}
	testRt(t, tc)
}

func TestStopRT_AfterMidnight(t *testing.T) {
	tc := rtTestCase{
		name:  "just after midnight",
		query: rtTestStopQuery,
		vars: hw{
			"stop_id": "FTVL",
			"stf": hw{
				"date":  "2018-05-30",
				"start": "23:45:00",
				"end":   "24:05:00",
			},
		},
		rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_trip_updates", Fname: "BA-midnight.json"}},
		whenUtc: rtFixtureWhenUtc,
		cb: func(t *testing.T, jj string) {
			a := gjson.Get(jj, "stops.0.stop_times").Array()
			// after midnight trip
			checkTrip := "5172328WKDY"
			found := false
			for _, st := range a {
				if st.Get("trip.trip_id").String() != checkTrip {
					continue
				}
				found = true
				assert.Equal(t, checkTrip, st.Get("trip.trip_id").String())
				assert.Equal(t, "00:05:00", st.Get("arrival.estimated").String())
				assert.Equal(t, "2018-05-31T00:05:00-07:00", st.Get("arrival.estimated_local").String())
				assert.Equal(t, "2018-05-31T07:05:00Z", st.Get("arrival.estimated_utc").String())
				assert.Equal(t, "24:02:00", st.Get("arrival.scheduled").String())
				assert.Equal(t, "2018-05-31T00:02:00-07:00", st.Get("arrival.scheduled_local").String())
				assert.Equal(t, "2018-05-31T07:02:00Z", st.Get("arrival.scheduled_utc").String())
			}
			if !found {
				t.Errorf("expected to find trip '%s'", checkTrip)
			}
		},
	}
	testRt(t, tc)
}

func TestStopRT_ArrivalFallback(t *testing.T) {
	tc := rtTestCase{
		name:    "arrival will use departure if arrival is not present",
		query:   rtTestStopQuery,
		vars:    rtTestStopQueryVars(),
		rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_trip_updates", Fname: "BA-arrival-fallback.json"}},
		whenUtc: rtFixtureWhenUtc,
		cb: func(t *testing.T, jj string) {
			a := gjson.Get(jj, "stops.0.stop_times").Array()
			checkTrip := "1031527WKDY"
			found := false
			for _, st := range a {
				if st.Get("trip.trip_id").String() != checkTrip {
					continue
				}
				found = true
				assert.Equal(t, "2018-05-30T23:02:30Z", st.Get("arrival.estimated_utc").String())
			}
			if !found {
				t.Errorf("expected to find trip '%s'", checkTrip)
			}
		},
	}
	testRt(t, tc)
}

func TestStopRT_DepartureFallback(t *testing.T) {
	tc := rtTestCase{
		name:    "departure will use arrival if departure is not present",
		query:   rtTestStopQuery,
		vars:    rtTestStopQueryVars(),
		rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_trip_updates", Fname: "BA-departure-fallback.json"}},
		whenUtc: rtFixtureWhenUtc,
		cb: func(t *testing.T, jj string) {
			a := gjson.Get(jj, "stops.0.stop_times").Array()
			checkTrip := "1031527WKDY"
			found := false
			for _, st := range a {
				if st.Get("trip.trip_id").String() != checkTrip {
					continue
				}
				found = true
				assert.Equal(t, "2018-05-30T23:02:30Z", st.Get("departure.estimated_utc").String())
			}
			if !found {
				t.Errorf("expected to find trip '%s'", checkTrip)
			}
		},
	}
	testRt(t, tc)
}

// A stop time the trip update doesn't name takes the delay given at the
// nearest earlier stop, Lake Merritt's 45 seconds, not Coliseum's 90 after it.
func TestStopRT_CarriedDelay(t *testing.T) {
	tc := rtTestCase{
		name:    "the nearest earlier delay carries",
		query:   rtTestStopQuery,
		vars:    rtTestStopQueryVars(),
		rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_trip_updates", Fname: "BA-last-delay.json"}},
		whenUtc: rtFixtureWhenUtc,
		cb: func(t *testing.T, jj string) {
			a := gjson.Get(jj, "stops.0.stop_times").Array()
			checkTrip := "1031527WKDY"
			found := false
			for _, st := range a {
				if st.Get("trip.trip_id").String() != checkTrip {
					continue
				}
				found = true
				assert.Equal(t, "2018-05-30T16:02:00-07:00", st.Get("departure.scheduled_local").String())
				assert.Equal(t, "2018-05-30T23:02:00Z", st.Get("departure.scheduled_utc").String())
				assert.Equal(t, "1527721320", st.Get("departure.scheduled_unix").String())
				assert.Equal(t, "2018-05-30T16:02:45-07:00", st.Get("departure.estimated_local").String())
				assert.Equal(t, "2018-05-30T23:02:45Z", st.Get("departure.estimated_utc").String())
				assert.Equal(t, "1527721365", st.Get("departure.estimated_unix").String())
				assert.Equal(t, int64(45), st.Get("departure.estimated_delay").Int())
				// Check delay is NOT set
				assert.Equal(t, "", st.Get("departure.delay").String())
			}
			if !found {
				t.Errorf("expected to find trip '%s'", checkTrip)
			}
		},
	}
	testRt(t, tc)
}
func TestStopRT_StopIDFallback(t *testing.T) {
	tc := rtTestCase{
		name:    "use stop_id as fallback if no matching stop sequence",
		query:   rtTestStopQuery,
		vars:    rtTestStopQueryVars(),
		rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_trip_updates", Fname: "BA-stop-id-fallback.json"}},
		whenUtc: rtFixtureWhenUtc,
		cb: func(t *testing.T, jj string) {
			a := gjson.Get(jj, "stops.0.stop_times").Array()
			checkTrip := "1031527WKDY"
			found := false
			for _, st := range a {
				if st.Get("trip.trip_id").String() != checkTrip {
					continue
				}
				found = true
				assert.Equal(t, checkTrip, st.Get("trip.trip_id").String())
				assert.Equal(t, "2018-05-30T23:02:30Z", st.Get("departure.estimated_utc").String())
			}
			if !found {
				t.Errorf("expected to find trip '%s'", checkTrip)
			}
		},
	}
	testRt(t, tc)
}

// Stop time updates naming stops by stop_id alone are placed in order along
// the trip, so a stop the update names twice takes the first in order, and the
// second, which no later stop time matches, is left out.
func TestStopRT_StopIDFallback_DoubleVisit(t *testing.T) {
	tc := rtTestCase{
		name:    "a stop named twice takes the update in order",
		query:   rtTestStopQuery,
		vars:    rtTestStopQueryVars(),
		rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_trip_updates", Fname: "BA-stop-double-visit.json"}},
		whenUtc: rtFixtureWhenUtc,
		cb: func(t *testing.T, jj string) {
			a := gjson.Get(jj, "stops.0.stop_times").Array()
			checkTrip := "1031527WKDY"
			found := false
			for _, st := range a {
				if st.Get("trip.trip_id").String() != checkTrip {
					continue
				}
				found = true
				assert.Equal(t, "2018-05-30T23:02:30Z", st.Get("departure.estimated_utc").String())
			}
			if !found {
				t.Errorf("expected to find trip '%s'", checkTrip)
			}
		},
	}
	testRt(t, tc)
}

func TestStopRT_NoRT(t *testing.T) {
	tc := rtTestCase{
		name:    "no rt matches for trip 2211533WKDY",
		query:   rtTestStopQuery,
		vars:    rtTestStopQueryVars(),
		rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_trip_updates", Fname: "BA-departure-fallback.json"}},
		whenUtc: rtFixtureWhenUtc,
		cb: func(t *testing.T, jj string) {
			a := gjson.Get(jj, "stops.0.stop_times").Array()
			checkTrip := "2211533WKDY"
			found := false
			for _, st := range a {
				if st.Get("trip.trip_id").String() != checkTrip {
					continue
				}
				found = true
				assert.Equal(t, checkTrip, st.Get("trip.trip_id").String())
				assert.Equal(t, "STATIC", st.Get("trip.schedule_relationship").String(), "trip.schedule_relationship")
				assert.Equal(t, "", st.Get("trip.timestamp").String())
				assert.Equal(t, "", st.Get("arrival.estimated_utc").String())
				assert.Equal(t, "", st.Get("departure.estimated_utc").String())
			}
			if !found {
				t.Errorf("expected to find trip '%s'", checkTrip)
			}
		},
	}
	testRt(t, tc)
}

func TestStopRT_AddedTrip(t *testing.T) {
	tc := rtTestCase{
		name:    "stop times added trip",
		query:   rtTestStopQuery,
		vars:    rtTestStopQueryVars(),
		rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_trip_updates", Fname: "BA-added.json"}},
		whenUtc: rtFixtureWhenUtc,
		cb: func(t *testing.T, jj string) {
			checkTrip := "-123"
			found := false
			a := gjson.Get(jj, "stops.0.stop_times").Array()
			assert.Equal(t, 4, len(a))
			for _, st := range a {
				if st.Get("trip.trip_id").String() != checkTrip {
					continue
				}
				found = true
				assert.Equal(t, checkTrip, st.Get("trip.trip_id").String(), "trip.trip_id")
				assert.Equal(t, "05", st.Get("trip.route.route_id").String(), "trip.route.route_id")
				assert.Equal(t, "ADDED", st.Get("trip.schedule_relationship").String(), "trip.schedule_relationship")
				assert.Equal(t, "", st.Get("arrival.scheduled").String(), "arrival.scheduled")
				assert.Equal(t, "", st.Get("departure.scheduled").String(), "departure.scheduled")
				assert.Equal(t, "2018-05-30T23:02:32Z", st.Get("arrival.estimated_utc").String(), "arrival.estimated_utc")
				assert.Equal(t, "2018-05-30T23:02:32Z", st.Get("departure.estimated_utc").String(), "departure.estimated_utc")
				assert.Equal(t, 12, int(st.Get("stop_sequence").Int()), "stop_sequence")
			}
			if !found {
				t.Errorf("expected to find trip '%s'", checkTrip)
			}
		},
	}
	testRt(t, tc)
}

func TestStopRT_ScheduleRelationship(t *testing.T) {
	tcs := []rtTestCase{
		{
			name:    "static trip",
			query:   rtTestStopQuery,
			vars:    rtTestStopQueryVars(),
			rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_trip_updates", Fname: "BA-added.json"}},
			whenUtc: rtFixtureWhenUtc,
			cb: func(t *testing.T, jj string) {
				checkTrip := "1031527WKDY"
				found := false
				a := gjson.Get(jj, "stops.0.stop_times").Array()
				for _, st := range a {
					if st.Get("trip.trip_id").String() != checkTrip {
						continue
					}
					found = true
					assert.Equal(t, checkTrip, st.Get("trip.trip_id").String(), "trip.trip_id")
					assert.Equal(t, "2018-05-30T23:02:00Z", st.Get("departure.scheduled_utc").String(), "departure.scheduled_utc")
					assert.Equal(t, "", st.Get("departure.estimated_utc").String(), "departure.estimated_utc")
					assert.Equal(t, "STATIC", st.Get("trip.schedule_relationship").String(), "trip.schedule_relationship")
					assert.Equal(t, "STATIC", st.Get("schedule_relationship").String(), "schedule_relationship")
				}
				if !found {
					t.Errorf("expected to find trip '%s'", checkTrip)
				}
			},
		},

		{
			name:    "scheduled trip",
			query:   rtTestStopQuery,
			vars:    rtTestStopQueryVars(),
			rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_trip_updates", Fname: "BA-added.json"}},
			whenUtc: rtFixtureWhenUtc,
			cb: func(t *testing.T, jj string) {
				checkTrip := "1131530WKDY"
				found := false
				a := gjson.Get(jj, "stops.0.stop_times").Array()
				for _, st := range a {
					if st.Get("trip.trip_id").String() != checkTrip {
						continue
					}
					found = true
					assert.Equal(t, checkTrip, st.Get("trip.trip_id").String(), "trip.trip_id")
					assert.Equal(t, "2018-05-30T23:05:00Z", st.Get("departure.scheduled_utc").String(), "departure.scheduled_utc")
					assert.Equal(t, "2018-05-30T23:05:45Z", st.Get("departure.estimated_utc").String(), "departure.estimated_utc")
					assert.Equal(t, "SCHEDULED", st.Get("trip.schedule_relationship").String(), "trip.schedule_relationship")
					assert.Equal(t, "SCHEDULED", st.Get("schedule_relationship").String(), "schedule_relationship")
				}
				if !found {
					t.Errorf("expected to find trip '%s'", checkTrip)
				}
			},
		},
		{
			name:    "added trip",
			query:   rtTestStopQuery,
			vars:    rtTestStopQueryVars(),
			rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_trip_updates", Fname: "BA-added.json"}},
			whenUtc: rtFixtureWhenUtc,
			cb: func(t *testing.T, jj string) {
				checkTrip := "-123"
				found := false
				a := gjson.Get(jj, "stops.0.stop_times").Array()
				for _, st := range a {
					if st.Get("trip.trip_id").String() != checkTrip {
						continue
					}
					found = true
					assert.Equal(t, checkTrip, st.Get("trip.trip_id").String(), "trip.trip_id")
					assert.Equal(t, "", st.Get("departure.scheduled_utc").String(), "departure.scheduled_utc")
					assert.Equal(t, "2018-05-30T23:02:32Z", st.Get("departure.estimated_utc").String(), "departure.estimated_utc")
					assert.Equal(t, "ADDED", st.Get("trip.schedule_relationship").String(), "trip.schedule_relationship")
					assert.Equal(t, "ADDED", st.Get("schedule_relationship").String(), "schedule_relationship")
				}
				if !found {
					t.Errorf("expected to find trip '%s'", checkTrip)
				}
			},
		},
		{
			name:    "canceled trip",
			query:   rtTestStopQuery,
			vars:    rtTestStopQueryVars(),
			rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_trip_updates", Fname: "BA-added.json"}},
			whenUtc: rtFixtureWhenUtc,
			cb: func(t *testing.T, jj string) {
				checkTrip := "2211533WKDY"
				found := false
				a := gjson.Get(jj, "stops.0.stop_times").Array()
				for _, st := range a {
					if st.Get("trip.trip_id").String() != checkTrip {
						continue
					}
					found = true
					assert.Equal(t, checkTrip, st.Get("trip.trip_id").String(), "trip.trip_id")
					assert.Equal(t, "2018-05-30T23:02:00Z", st.Get("departure.scheduled_utc").String(), "departure.scheduled_utc")
					assert.Equal(t, "", st.Get("departure.estimated_utc").String(), "departure.estimated_utc")
					assert.Equal(t, "CANCELED", st.Get("trip.schedule_relationship").String(), "trip.schedule_relationship")
					assert.Equal(t, "CANCELED", st.Get("schedule_relationship").String(), "chedule_relationship")
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

func TestStopRT_CanceledTrip(t *testing.T) {
	tc := rtTestCase{
		name:    "stop times canceled trip",
		query:   rtTestStopQuery,
		vars:    rtTestStopQueryVars(),
		rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_trip_updates", Fname: "BA-added.json"}},
		whenUtc: rtFixtureWhenUtc,
		cb: func(t *testing.T, jj string) {
			checkTrip := "2211533WKDY"
			found := false
			a := gjson.Get(jj, "stops.0.stop_times").Array()
			assert.Equal(t, 4, len(a))
			for _, st := range a {
				if st.Get("trip.trip_id").String() != checkTrip {
					continue
				}
				found = true
				assert.Equal(t, checkTrip, st.Get("trip.trip_id").String(), "trip.trip_id")
				assert.Equal(t, "03", st.Get("trip.route.route_id").String(), "trip.route.route_id")
				assert.Equal(t, "CANCELED", st.Get("trip.schedule_relationship").String(), "trip.schedule_relationship")
				assert.Equal(t, "16:02:00", st.Get("arrival.scheduled").String(), "arrival.scheduled")
				assert.Equal(t, "16:02:00", st.Get("departure.scheduled").String(), "departure.scheduled")
				assert.Equal(t, "", st.Get("arrival.estimated_utc").String(), "arrival.estimated_utc")
				assert.Equal(t, "", st.Get("departure.estimated_utc").String(), "departure.estimated_utc")
			}
			if !found {
				t.Errorf("expected to find trip '%s'", checkTrip)
			}
		},
	}
	testRt(t, tc)
}

func TestStopRT_Alerts(t *testing.T) {
	activeVars := rtTestStopQueryVars()
	activeVars["active"] = true
	tcs := []rtTestCase{
		{
			name:  "stop alerts",
			query: rtTestStopQuery,
			vars:  rtTestStopQueryVars(),
			rtfiles: []testconfig.RTJsonFile{
				{Feed: "BA", Ftype: "realtime_alerts", Fname: "BA-alerts.json"},
			},
			cb: func(t *testing.T, jj string) {
				alerts := gjson.Get(jj, "stops.0.alerts").Array()
				if len(alerts) != 2 {
					t.Errorf("got %d alerts, expected 2", len(alerts))
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
				alerts := gjson.Get(jj, "stops.0.alerts").Array()
				if len(alerts) == 1 {
					firstAlert := alerts[0]
					assert.Equal(t, "Test stop header - active", firstAlert.Get("header_text.0.text").String(), "header_text.0.text")
					assert.Contains(t, firstAlert.Get("description_text.0.text").String(), "stop_id:FTVL", "description_text.0.text")
				} else {
					t.Errorf("got %d alerts, expected 1", len(alerts))
				}
			},
		},
	}
	for _, tc := range tcs {
		testRt(t, tc)
	}
}

// schedule_relationship must answer the same whether or not arrival and
// departure are also selected. It once did not — population was gated on those
// two, so asking for it alone reported STATIC for rescheduled stop times.
func TestStopRT_ScheduleRelationshipWithoutArrivalDeparture(t *testing.T) {
	const query = `query($stop_id: String!, $stf: StopTimeFilter) {
		stops(where: {stop_id: $stop_id}) {
			stop_times(where: $stf) {
				schedule_relationship
				trip { trip_id }
			}
		}
	}`
	testRt(t, rtTestCase{
		name:    "schedule_relationship alone",
		query:   query,
		vars:    rtTestStopQueryVars(),
		rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_trip_updates", Fname: "BA-added.json"}},
		whenUtc: rtFixtureWhenUtc,
		cb: func(t *testing.T, jj string) {
			checkTrip := "1131530WKDY"
			found := false
			for _, st := range gjson.Get(jj, "stops.0.stop_times").Array() {
				if st.Get("trip.trip_id").String() != checkTrip {
					continue
				}
				found = true
				assert.Equal(t, "SCHEDULED", st.Get("schedule_relationship").String(), "schedule_relationship")
			}
			if !found {
				t.Errorf("expected to find trip '%s'", checkTrip)
			}
		},
	})
}

// Departures carry estimates from a trip updates payload fetched under an hour
// ago, and none from one fetched over an hour ago.
func TestStopRT_StalePayload(t *testing.T) {
	for _, tc := range []struct {
		name      string
		age       time.Duration
		estimated bool
	}{
		{"fetched 59 minutes ago", 59 * time.Minute, true},
		{"fetched 61 minutes ago", 61 * time.Minute, false},
	} {
		testRt(t, rtTestCase{
			name:    tc.name,
			query:   rtTestStopQuery,
			vars:    rtTestStopQueryVars(),
			whenUtc: rtFixtureWhenUtc,
			rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_trip_updates", Fname: "BA.json", Age: tc.age}},
			cb: func(t *testing.T, jj string) {
				sts := gjson.Get(jj, "stops.0.stop_times").Array()
				if !assert.NotEmpty(t, sts) {
					return
				}
				for _, st := range sts {
					assert.Equal(t, tc.estimated, st.Get("arrival.estimated").Value() != nil, "trip %s", st.Get("trip.trip_id").String())
				}
			},
		})
	}
}

// An added trip on a route the feed version doesn't have leaves the rest of
// the stop's departures in place.
func TestStopRT_AddedTripUnknownRoute(t *testing.T) {
	testRt(t, rtTestCase{
		name:    "added trip on an unknown route",
		query:   rtTestStopQuery,
		vars:    rtTestStopQueryVars(),
		rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_trip_updates", Fname: "BA-added-unknown-route.json"}},
		whenUtc: rtFixtureWhenUtc,
		cb: func(t *testing.T, jj string) {
			var trips []string
			for _, st := range gjson.Get(jj, "stops.0.stop_times").Array() {
				trips = append(trips, st.Get("trip.trip_id").String())
			}
			assert.Subset(t, trips, []string{"1031527WKDY", "1131530WKDY", "2211533WKDY"}, "scheduled departures")
		},
	})
}
