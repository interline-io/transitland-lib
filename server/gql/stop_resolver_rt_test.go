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
				assert.Equal(t, "24:05:00", st.Get("arrival.estimated").String())
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

// A skipped stop, or one the update has no data for, reports that relationship
// and no estimate, while a stop the update doesn't name takes the carried
// delay and reports SCHEDULED.
func TestStopRT_SkippedAndNoData(t *testing.T) {
	testRt(t, rtTestCase{
		name:    "skipped and no data stops",
		query:   rtTestStopQuery,
		vars:    rtTestStopQueryVars(),
		rtfiles: []testconfig.RTJsonFile{{Feed: "BA", Ftype: "realtime_trip_updates", Fname: "BA-skipped.json"}},
		whenUtc: rtFixtureWhenUtc,
		cb: func(t *testing.T, jj string) {
			want := map[string]struct {
				rel   string
				delay *int64
			}{
				"1031527WKDY": {"SKIPPED", nil},
				"1131530WKDY": {"NO_DATA", nil},
				"2211533WKDY": {"SCHEDULED", ptr(int64(32))},
			}
			found := map[string]bool{}
			for _, st := range gjson.Get(jj, "stops.0.stop_times").Array() {
				tid := st.Get("trip.trip_id").String()
				w, ok := want[tid]
				if !ok {
					continue
				}
				found[tid] = true
				assert.Equal(t, w.rel, st.Get("schedule_relationship").String(), "%s schedule_relationship", tid)
				assert.Equal(t, w.delay, optionalInt(st.Get("departure.estimated_delay")), "%s estimated_delay", tid)
			}
			assert.Len(t, found, 3)
		},
	})
}

// An update naming a start_time describes that instance of a frequency-based
// trip alone, and the instance's stop times, at the stop and on its trip, are
// its own times with estimates counted from them.
func TestStopRT_FrequencyInstances(t *testing.T) {
	// The example feed's version isn't its feed's active one, so the stop is
	// found by the version's sha1.
	const query = `query($stf: StopTimeFilter) {
		stops(where: {feed_version_sha1: "43e2278aa272879c79460582152b04e7487f0493", stop_id: "STAGECOACH"}) {
			stop_times(where: $stf) {
				schedule_relationship
				trip {
					trip_id
					schedule_relationship
					stop_times { stop_sequence schedule_relationship departure { scheduled estimated_delay } }
					later: stop_times(where: {start: "08:33:00"}) { stop_sequence departure { scheduled } }
				}
				departure { scheduled estimated_delay }
			}
		}
	}`
	testRt(t, rtTestCase{
		name:    "frequency instances",
		query:   query,
		vars:    hw{"stf": hw{"date": "2007-01-02", "start": "08:00:00", "end": "08:59:00"}},
		rtfiles: []testconfig.RTJsonFile{{Feed: "EX", Ftype: "realtime_trip_updates", Fname: "EX-frequency.json"}},
		whenUtc: "2007-01-02T16:05:00Z",
		cb: func(t *testing.T, jj string) {
			instances := 0
			for _, st := range gjson.Get(jj, "stops.0.stop_times").Array() {
				if st.Get("trip.trip_id").String() != "CITY1" {
					continue
				}
				instances++
				start := st.Get("departure.scheduled").String()
				rel, tripRel := st.Get("schedule_relationship").String(), st.Get("trip.schedule_relationship").String()
				delay := optionalInt(st.Get("departure.estimated_delay"))
				// The trip's own stop times are the instance's, so its first stop
				// is this departure.
				first := st.Get("trip.stop_times.0")
				assert.Equal(t, start, first.Get("departure.scheduled").String(), "%s trip.stop_times.0 scheduled", start)
				assert.Equal(t, rel, first.Get("schedule_relationship").String(), "%s trip.stop_times.0 schedule_relationship", start)
				assert.Equal(t, delay, optionalInt(first.Get("departure.estimated_delay")), "%s trip.stop_times.0 estimated_delay", start)
				switch start {
				case "08:10:00":
					assert.Equal(t, "CANCELED", rel, start)
					assert.Equal(t, "CANCELED", tripRel, start)
					assert.Nil(t, delay, start)
				case "08:30:00":
					assert.Equal(t, "SCHEDULED", rel, start)
					assert.Equal(t, "SCHEDULED", tripRel, start)
					assert.Equal(t, ptr(int64(60)), delay, start)
					// The delay carries along the instance.
					assert.Equal(t, ptr(int64(60)), optionalInt(st.Get("trip.stop_times.4.departure.estimated_delay")), "last stop")
					// A time window applies to the instance's own times: its stops
					// from 08:33 on are the last four, which the template visits
					// from 06:07.
					if later := st.Get("trip.later").Array(); assert.Len(t, later, 4, "stops from 08:33") {
						assert.Equal(t, int64(2), later[0].Get("stop_sequence").Int())
						assert.Equal(t, "08:37:00", later[0].Get("departure.scheduled").String())
					}
				default:
					assert.Equal(t, "STATIC", rel, start)
					assert.Equal(t, "STATIC", tripRel, start)
					assert.Nil(t, delay, start)
				}
			}
			assert.Equal(t, 6, instances, "instances of CITY1 from 08:00 to 08:59")
		},
	})
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
