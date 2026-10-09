package gql

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// Every realtime message describes one run of a trip: the run on its trip
// descriptor's start_date, or where it names none, the trip's current run. See
// BA-runs-*.json in testdata/server/rt.

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

// One departure's realtime data.
type runRT struct {
	serviceDate string
	delay       *int64
	alerts      []string
	vehicle     string
}

// The realtime data on each departure, by trip_id.
func runsByTrip(stopTimes gjson.Result) map[string]runRT {
	ret := map[string]runRT{}
	for _, st := range stopTimes.Array() {
		r := runRT{
			serviceDate: st.Get("service_date").String(),
			delay:       optionalInt(st.Get("departure.estimated_delay")),
			vehicle:     st.Get("trip.vehicle_position.vehicle.label").String(),
		}
		for _, a := range st.Get("trip.alerts.#.header_text.0.text").Array() {
			r.alerts = append(r.alerts, a.String())
		}
		ret[st.Get("trip.trip_id").String()] = r
	}
	return ret
}

func optionalInt(v gjson.Result) *int64 {
	if !v.Exists() || v.Type == gjson.Null {
		return nil
	}
	i := v.Int()
	return &i
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
				"1031527WKDY": {serviceDate: "2018-05-30", delay: ptrInt64(60), alerts: []string{"Run of May 30", "Current run"}, vehicle: "May 30 train"},
				"2211533WKDY": {serviceDate: "2018-05-30", delay: ptrInt64(120), vehicle: "Undated train"},
				"1131530WKDY": {serviceDate: "2018-05-30"},
			},
		},
		{
			// Tomorrow's run gets only what is dated for it.
			name:    "tomorrow's runs",
			whenUtc: rtFixtureWhenUtc,
			vars:    window("2018-05-31"),
			expect: map[string]runRT{
				"1031527WKDY": {serviceDate: "2018-05-31", delay: ptrInt64(300), alerts: []string{"Run of May 31"}},
				"2211533WKDY": {serviceDate: "2018-05-31"},
				"1131530WKDY": {serviceDate: "2018-05-31"},
			},
		},
		{
			name:    "next week's runs",
			whenUtc: rtFixtureWhenUtc,
			vars:    window("2018-06-05"),
			expect: map[string]runRT{
				"1031527WKDY": {serviceDate: "2018-06-05"},
				"2211533WKDY": {serviceDate: "2018-06-05"},
			},
		},
		{
			// At ten past midnight, the previous service date's run of a trip that
			// arrives at 24:31 is still going, so it is the current run.
			name:    "after midnight, late run still going",
			whenUtc: "2018-05-31T07:10:00Z",
			vars:    hw{"where": hw{"date": "2018-05-31", "start_time": 0, "end_time": 300}},
			expect: map[string]runRT{
				"5172328WKDY": {serviceDate: "2018-05-30", delay: ptrInt64(180), alerts: []string{"Late run, current", "Late run of May 30"}},
				"2232328WKDY": {serviceDate: "2018-05-30"},
			},
		},
		{
			// At a quarter to one it has finished, and today's run is current.
			name:    "after midnight, late run finished",
			whenUtc: "2018-05-31T07:45:00Z",
			vars:    hw{"where": hw{"date": "2018-05-31", "start_time": 0, "end_time": 300}},
			expect: map[string]runRT{
				"5172328WKDY": {serviceDate: "2018-05-30", alerts: []string{"Late run of May 30"}},
			},
		},
		{
			// Answered from the fallback week, but for the requested date, which is
			// today: its dated messages and the undated ones both apply.
			name:    "fallback week",
			whenUtc: "2030-05-28T23:00:00Z",
			vars:    hw{"where": hw{"date": "2030-05-28", "start_time": 57600, "end_time": 57900, "use_service_window": true}},
			expect: map[string]runRT{
				"1031527WKDY": {serviceDate: "2030-05-28", delay: ptrInt64(90), alerts: []string{"Run of May 28, 2030", "Current run"}},
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
				got := runsByTrip(gjson.Get(jj, "stops.0.stop_times"))
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
// found without a date, it is its current run.
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
		// At FTVL. A trip found without a date has no date for a delay-only
		// estimate, but its stop times still carry the update.
		delay        *int64
		relationship string
		alerts       []string
		vehicle      string
	}{
		{
			name:         "today's run",
			where:        hw{"trip_id": "1031527WKDY", "service_date": "2018-05-30"},
			delay:        ptrInt64(60),
			relationship: "SCHEDULED",
			alerts:       []string{"Run of May 30", "Current run"},
			vehicle:      "May 30 train",
		},
		{
			name:         "tomorrow's run",
			where:        hw{"trip_id": "1031527WKDY", "service_date": "2018-05-31"},
			delay:        ptrInt64(300),
			relationship: "SCHEDULED",
			alerts:       []string{"Run of May 31"},
		},
		{
			name:         "next week's run",
			where:        hw{"trip_id": "1031527WKDY", "service_date": "2018-06-05"},
			relationship: "STATIC",
		},
		{
			name:         "no date, so the current run",
			where:        hw{"trip_id": "1031527WKDY"},
			relationship: "SCHEDULED",
			alerts:       []string{"Run of May 30", "Current run"},
			vehicle:      "May 30 train",
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
				for _, st := range trip.Get("stop_times").Array() {
					if st.Get("stop_sequence").Int() == 12 {
						assert.Equal(t, tc.delay, optionalInt(st.Get("departure.estimated_delay")), "estimated_delay at FTVL")
						assert.Equal(t, tc.relationship, st.Get("schedule_relationship").String(), "schedule_relationship at FTVL")
					}
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
