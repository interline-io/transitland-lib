package gql

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// Realtime data is matched to the runs of a trip it describes: a message naming
// a start_date to that run only, an undated trip update or vehicle position to
// the current run, and an undated alert to the runs operating while it is in
// force. See BA-runs-*.json in testdata/server/rt.

var runsRTFiles = []testconfig.RTJsonFile{
	{Feed: "BA", Ftype: "realtime_trip_updates", Fname: "BA-runs-trip-updates.json"},
	{Feed: "BA", Ftype: "realtime_alerts", Fname: "BA-runs-alerts.json"},
	{Feed: "BA", Ftype: "realtime_vehicle_positions", Fname: "BA-runs-vehicle-positions.json"},
}

const runsDeparturesQuery = `query($where: StopTimeFilter!) {
	stops(where: {stop_id: "FTVL"}) {
		stop_times(where: $where) {
			service_date
			departure { estimated_delay scheduled_utc }
			trip {
				trip_id
				alerts { header_text { text } }
				vehicle_position { vehicle { label } }
			}
		}
	}
}`

// One departure's realtime data, by trip_id.
type runRT struct {
	serviceDate string
	delay       *int64
	alerts      []string
	vehicle     string
}

func runsByTrip(t *testing.T, stopTimes gjson.Result) map[string]runRT {
	ret := map[string]runRT{}
	for _, st := range stopTimes.Array() {
		r := runRT{
			serviceDate: st.Get("service_date").String(),
			vehicle:     st.Get("trip.vehicle_position.vehicle.label").String(),
		}
		if d := st.Get("departure.estimated_delay"); d.Exists() && d.Type != gjson.Null {
			v := d.Int()
			r.delay = &v
		}
		for _, a := range st.Get("trip.alerts.#.header_text.0.text").Array() {
			r.alerts = append(r.alerts, a.String())
		}
		ret[st.Get("trip.trip_id").String()] = r
	}
	return ret
}

func ptrInt64(v int64) *int64 {
	return &v
}

func TestTripRT_Runs_Departures(t *testing.T) {
	// The departures of 4:00 to 4:05 pm at Fruitvale on a service date.
	window := func(serviceDate string) hw {
		return hw{"where": hw{"service_date": serviceDate, "start_time": 57600, "end_time": 57900}}
	}
	tcs := []struct {
		name    string
		whenUtc string
		vars    hw
		expect  map[string]runRT
	}{
		{
			name:    "today's runs",
			whenUtc: rtFixtureWhenUtc,
			vars:    window("2018-05-30"),
			expect: map[string]runRT{
				"1031527WKDY": {serviceDate: "2018-05-30", delay: ptrInt64(60), alerts: []string{"Run of May 30", "During the May 30 run", "Always"}, vehicle: "May 30 train"},
				"2211533WKDY": {serviceDate: "2018-05-30", delay: ptrInt64(120), vehicle: "Undated train"},
				"1131530WKDY": {serviceDate: "2018-05-30"},
			},
		},
		{
			// A dated message describes its run whatever day the query comes; an
			// undated one only the current run, which tomorrow's is not.
			name:    "tomorrow's runs",
			whenUtc: rtFixtureWhenUtc,
			vars:    window("2018-05-31"),
			expect: map[string]runRT{
				"1031527WKDY": {serviceDate: "2018-05-31", delay: ptrInt64(300), alerts: []string{"Run of May 31", "Always"}},
				"2211533WKDY": {serviceDate: "2018-05-31"},
				"1131530WKDY": {serviceDate: "2018-05-31"},
			},
		},
		{
			name:    "next week's runs",
			whenUtc: rtFixtureWhenUtc,
			vars:    window("2018-06-05"),
			expect: map[string]runRT{
				"1031527WKDY": {serviceDate: "2018-06-05", alerts: []string{"Next week", "Always"}},
				"2211533WKDY": {serviceDate: "2018-06-05"},
				"1131530WKDY": {serviceDate: "2018-06-05"},
			},
		},
		{
			// At half past midnight the current run of a trip departing 24:02 is
			// the previous service date's.
			name:    "after midnight",
			whenUtc: "2018-05-31T07:30:00Z",
			vars:    hw{"where": hw{"date": "2018-05-31", "start_time": 0, "end_time": 300}},
			expect: map[string]runRT{
				"5172328WKDY": {serviceDate: "2018-05-30", delay: ptrInt64(180), alerts: []string{"After midnight", "Run of May 30, after midnight"}},
				"2232328WKDY": {serviceDate: "2018-05-30"},
			},
		},
		{
			// Answered from the fallback week, but for the requested date: its
			// dated messages and the current run's undated ones still apply.
			name:    "fallback week",
			whenUtc: "2030-05-28T23:00:00Z",
			vars:    hw{"where": hw{"date": "2030-05-28", "start_time": 57600, "end_time": 57900, "use_service_window": true}},
			expect: map[string]runRT{
				"1031527WKDY": {serviceDate: "2030-05-28", delay: ptrInt64(90), alerts: []string{"Run of May 28, 2030", "Always"}},
				"2211533WKDY": {serviceDate: "2030-05-28", delay: ptrInt64(120), vehicle: "Undated train"},
				"1131530WKDY": {serviceDate: "2030-05-28"},
			},
		},
	}
	for _, tc := range tcs {
		testRt(t, rtTestCase{
			name:    tc.name,
			query:   runsDeparturesQuery,
			vars:    tc.vars,
			rtfiles: runsRTFiles,
			whenUtc: tc.whenUtc,
			cb: func(t *testing.T, jj string) {
				got := runsByTrip(t, gjson.Get(jj, "stops.0.stop_times"))
				for tripId, want := range tc.expect {
					r, ok := got[tripId]
					if !assert.True(t, ok, "expected a departure on trip %s", tripId) {
						continue
					}
					assert.Equal(t, want.serviceDate, r.serviceDate, "%s service_date", tripId)
					assert.Equal(t, want.delay, r.delay, "%s estimated_delay", tripId)
					assert.ElementsMatch(t, want.alerts, r.alerts, "%s alerts", tripId)
					assert.Equal(t, want.vehicle, r.vehicle, "%s vehicle", tripId)
				}
			},
		})
	}
}

// A fallback-week departure's scheduled time is on the requested date, not on
// the fallback-week day that supplied its schedule.
func TestTripRT_Runs_FallbackScheduledTime(t *testing.T) {
	testRt(t, rtTestCase{
		name:    "scheduled on the requested date",
		query:   runsDeparturesQuery,
		vars:    hw{"where": hw{"date": "2030-05-28", "start_time": 57600, "end_time": 57900, "use_service_window": true}},
		rtfiles: runsRTFiles,
		whenUtc: "2030-05-28T23:00:00Z",
		cb: func(t *testing.T, jj string) {
			for _, st := range gjson.Get(jj, "stops.0.stop_times").Array() {
				if st.Get("trip.trip_id").String() == "1031527WKDY" {
					assert.Equal(t, "2030-05-28T23:02:00Z", st.Get("departure.scheduled_utc").String())
					return
				}
			}
			t.Error("expected a departure on trip 1031527WKDY")
		},
	})
}

// A trip found by its service date is that run, as the trip viewer finds one;
// found without a date, it is no run in particular, and every message that names
// the trip applies.
func TestTripRT_Runs_Trips(t *testing.T) {
	const query = `query($where: TripFilter!) {
		trips(where: $where) {
			trip_id
			stop_times { stop_sequence schedule_relationship departure { estimated_delay } }
			alerts { header_text { text } }
			vehicle_position { vehicle { label } }
		}
	}`
	tcs := []struct {
		name  string
		where hw
		delay *int64
		// Realtime data from some run, whichever: a trip reached as no run has
		// no one run to choose, and no date for a delay-only estimate.
		anyRun  bool
		alerts  []string
		vehicle string
	}{
		{
			name:    "today's run",
			where:   hw{"trip_id": "1031527WKDY", "service_date": "2018-05-30"},
			delay:   ptrInt64(60),
			alerts:  []string{"Run of May 30", "During the May 30 run", "Always"},
			vehicle: "May 30 train",
		},
		{
			name:   "tomorrow's run",
			where:  hw{"trip_id": "1031527WKDY", "service_date": "2018-05-31"},
			delay:  ptrInt64(300),
			alerts: []string{"Run of May 31", "Always"},
		},
		{
			name:   "next week's run",
			where:  hw{"trip_id": "1031527WKDY", "service_date": "2018-06-05"},
			alerts: []string{"Next week", "Always"},
		},
		{
			name:    "no run in particular",
			where:   hw{"trip_id": "1031527WKDY"},
			anyRun:  true,
			alerts:  []string{"Run of May 30", "Run of May 31", "Run of May 28, 2030", "During the May 30 run", "Next week", "Always"},
			vehicle: "May 30 train",
		},
	}
	for _, tc := range tcs {
		testRt(t, rtTestCase{
			name:    tc.name,
			query:   query,
			vars:    hw{"where": tc.where},
			rtfiles: runsRTFiles,
			whenUtc: rtFixtureWhenUtc,
			cb: func(t *testing.T, jj string) {
				trip := gjson.Get(jj, "trips.0")
				if !assert.Equal(t, "1031527WKDY", trip.Get("trip_id").String()) {
					return
				}
				var delay *int64
				relationship := ""
				for _, st := range trip.Get("stop_times").Array() {
					if st.Get("stop_sequence").Int() == 12 {
						relationship = st.Get("schedule_relationship").String()
						if d := st.Get("departure.estimated_delay"); d.Exists() && d.Type != gjson.Null {
							delay = ptrInt64(d.Int())
						}
					}
				}
				if tc.anyRun {
					assert.Equal(t, "SCHEDULED", relationship, "schedule_relationship at FTVL")
				} else {
					assert.Equal(t, tc.delay, delay, "estimated_delay at FTVL")
				}
				var alerts []string
				for _, a := range trip.Get("alerts.#.header_text.0.text").Array() {
					alerts = append(alerts, a.String())
				}
				assert.ElementsMatch(t, tc.alerts, alerts, "alerts")
				assert.Equal(t, tc.vehicle, trip.Get("vehicle_position.vehicle.label").String(), "vehicle")
			},
		})
	}
}
