package gqltest

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
)

func stopResolverStopTimesDatesTestcases(t testing.TB, cfg model.Config) []testcase {
	q := `query($stop_id:String!,$sd:Date,$date:Date,$ed:Boolean,$start:Seconds,$end:Seconds){ stops(where:{stop_id:$stop_id}) { stop_times(where:{service_date:$sd, date:$date, start:$start, end:$end, use_service_window:$ed}) {date service_date arrival_time}}}`
	testcases := []testcase{
		// Service date parameter
		{
			Name:         "service_date in range",
			Query:        q,
			Vars:         hw{"stop_id": "MCAR_S", "sd": "2018-05-29", "start": "15:00:00", "end": "16:00:00", "ed": true},
			Selector:     "stops.0.stop_times.0.service_date",
			SelectExpect: []string{"2018-05-29"}, // expect input date
		},
		// Answered from the fallback week, which the exact-dates case below shows
		// has the only service, but reported under the requested date.
		{
			Name:         "service_date after range",
			Query:        q,
			Vars:         hw{"stop_id": "MCAR_S", "sd": "2030-05-28", "start": "15:00:00", "end": "16:00:00", "ed": true},
			Selector:     "stops.0.stop_times.0.service_date",
			SelectExpect: []string{"2030-05-28"},
		},
		{
			Name:         "service_date before range, friday",
			Query:        q,
			Vars:         hw{"stop_id": "MCAR_S", "sd": "2010-05-28", "start": "15:00:00", "end": "16:00:00", "ed": true},
			Selector:     "stops.0.stop_times.0.service_date",
			SelectExpect: []string{"2010-05-28"},
		},
		{
			Name:         "service_date after range, exact dates",
			Query:        q,
			Vars:         hw{"stop_id": "MCAR_S", "sd": "2030-05-28", "start": "15:00:00", "end": "16:00:00", "ed": false},
			Selector:     "stops.0.stop_times.#.service_date",
			SelectExpect: []string{}, // exect no results
		},
		// Date parameter
		{
			Name:         "date 2018-05-29 3pm-4pm",
			Query:        q,
			Vars:         hw{"stop_id": "MCAR_S", "date": "2018-05-29", "start": "15:00:00", "end": "16:00:00"},
			Selector:     "stops.0.stop_times.#.arrival_time",
			SelectExpect: []string{"15:01:00", "15:09:00", "15:09:00", "15:16:00", "15:24:00", "15:24:00", "15:31:00", "15:39:00", "15:39:00", "15:46:00", "15:54:00", "15:54:00"},
		},
		// Previous day
		{
			Name:         "date 2018-05-28 12:15am - 2018-05-19 5am",
			Query:        q,
			Vars:         hw{"stop_id": "MCAR_S", "date": "2018-05-29", "start": "00:15:00", "end": "5:00:00"},
			Selector:     "stops.0.stop_times.#.arrival_time",
			SelectExpect: []string{"24:15:00", "24:15:00", "24:47:00", "24:47:00", "04:31:00", "04:39:00", "04:39:00", "04:46:00", "04:54:00", "04:54:00"},
		},
		// Next day
		{
			Name:         "date 2018-05-28 10:00pm - 2018-05-29 2am",
			Query:        q,
			Vars:         hw{"stop_id": "MCAR_S", "date": "2018-05-28", "start": "22:00:00", "end": "26:00:00"},
			Selector:     "stops.0.stop_times.#.arrival_time",
			SelectExpect: []string{"22:15:00", "22:15:00", "22:35:00", "22:35:00", "22:55:00", "22:55:00", "23:15:00", "23:15:00", "23:35:00", "23:35:00", "23:55:00", "23:55:00", "24:15:00", "24:15:00", "24:47:00", "24:47:00"},
		},
		{
			Name:         "date 2018-05-28 10:00pm - 2018-05-29 5am",
			Query:        q,
			Vars:         hw{"stop_id": "MCAR_S", "date": "2018-05-28", "start": "22:00:00", "end": "29:00:00"},
			Selector:     "stops.0.stop_times.#.arrival_time",
			SelectExpect: []string{"22:15:00", "22:15:00", "22:35:00", "22:35:00", "22:55:00", "22:55:00", "23:15:00", "23:15:00", "23:35:00", "23:35:00", "23:55:00", "23:55:00", "24:15:00", "24:15:00", "24:47:00", "24:47:00", "04:31:00", "04:39:00", "04:39:00", "04:46:00", "04:54:00", "04:54:00"},
		},
		// Check date, service date
		{
			Name:         "date 2018-05-28 10:00pm - 2018-05-29 2am check date",
			Query:        q,
			Vars:         hw{"stop_id": "MCAR_S", "date": "2018-05-28", "start": "22:00:00", "end": "26:00:00"},
			Selector:     "stops.0.stop_times.#.date",
			SelectExpect: []string{"2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-29", "2018-05-29", "2018-05-29", "2018-05-29"},
		},
		{
			Name:         "date 2018-05-28 10:00pm - 2018-05-29 2am check service date",
			Query:        q,
			Vars:         hw{"stop_id": "MCAR_S", "date": "2018-05-28", "start": "22:00:00", "end": "26:00:00"},
			Selector:     "stops.0.stop_times.#.service_date",
			SelectExpect: []string{"2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28"},
		},
	}
	return testcases
}

func stopResolverStopTimesDateWindowsTestcases(t testing.TB, cfg model.Config) []testcase {
	bartWeekdayTimes := []string{"15:01:00", "15:09:00", "15:09:00", "15:16:00", "15:24:00", "15:24:00", "15:31:00", "15:39:00", "15:39:00", "15:46:00", "15:54:00", "15:54:00"}
	bartWeekendTimes := []string{"15:15:00", "15:15:00", "15:35:00", "15:35:00", "15:55:00", "15:55:00"}
	q := `query($stop_id:String!,$sd:Date!,$ed:Boolean){ stops(where:{stop_id:$stop_id}) { stop_times(where:{service_date:$sd, start_time:54000, end_time:57600, use_service_window:$ed}) {arrival_time}}}`
	testcases := []testcase{
		{
			Name:         "service date in range",
			Query:        q,
			Vars:         hw{"stop_id": "MCAR_S", "sd": "2018-05-29", "ed": true},
			Selector:     "stops.0.stop_times.#.arrival_time",
			SelectExpect: bartWeekdayTimes,
		},
		{
			Name:         "service date after range",
			Query:        q,
			Vars:         hw{"stop_id": "MCAR_S", "sd": "2030-05-28", "ed": true},
			Selector:     "stops.0.stop_times.#.arrival_time",
			SelectExpect: bartWeekdayTimes,
		},
		{
			Name:         "service date after range, exact dates",
			Query:        q,
			Vars:         hw{"stop_id": "MCAR_S", "sd": "2030-05-28", "ed": false},
			Selector:     "stops.0.stop_times.#.arrival_time",
			SelectExpect: []string{},
		},
		{
			Name:         "service date after range, sunday",
			Query:        q,
			Vars:         hw{"stop_id": "MCAR_S", "sd": "2030-05-26", "ed": true},
			Selector:     "stops.0.stop_times.#.arrival_time",
			SelectExpect: bartWeekendTimes,
		},
		{
			Name:         "service date before range, tuesday",
			Query:        q,
			Vars:         hw{"stop_id": "MCAR_S", "sd": "2010-05-28", "ed": true},
			Selector:     "stops.0.stop_times.#.arrival_time",
			SelectExpect: bartWeekdayTimes,
		},
		{
			Name:         "fv without feed_info, in window, monday",
			Query:        q,
			Vars:         hw{"stop_id": "70011", "sd": "2019-02-11", "ed": true},
			Selector:     "stops.0.stop_times.#.arrival_time",
			SelectExpect: []string{"15:48:00", "15:50:00"},
		},
		{
			Name:         "fv without feed_info, before window, friday",
			Query:        q,
			Vars:         hw{"stop_id": "70011", "sd": "2010-05-28", "ed": true},
			Selector:     "stops.0.stop_times.#.arrival_time",
			SelectExpect: []string{"15:48:00", "15:50:00"},
		},
		{
			Name:         "fv without feed_info, after window, tuesday",
			Query:        q,
			Vars:         hw{"stop_id": "70011", "sd": "2030-05-28", "ed": true},
			Selector:     "stops.0.stop_times.#.arrival_time",
			SelectExpect: []string{"15:48:00", "15:50:00"},
		},
		{
			Name:         "fv without feed_info, after window, tuesday, exact date only",
			Query:        q,
			Vars:         hw{"stop_id": "70011", "sd": "2030-05-28", "ed": false},
			Selector:     "stops.0.stop_times.#.arrival_time",
			SelectExpect: []string{},
		},
	}
	return testcases
}

func stopResolverStopTimesTestcases(t testing.TB, cfg model.Config) []testcase {
	vars := hw{"trip_id": "3850526WKDY"}
	testcases := []testcase{
		{
			Name:         "basic",
			Query:        `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {trip_id stop_times { arrival_time }} }`,
			Vars:         vars,
			Selector:     "trips.0.stop_times.#.arrival_time",
			SelectExpect: []string{"05:26:00", "05:29:00", "05:33:00", "05:36:00", "05:40:00", "05:43:00", "05:46:00", "05:48:00", "05:50:00", "05:53:00", "05:54:00", "05:56:00", "05:58:00", "06:05:00", "06:08:00", "06:11:00", "06:15:00", "06:17:00", "06:23:00", "06:27:00", "06:32:00", "06:35:00", "06:40:00", "06:43:00", "06:50:00", "07:05:00", "07:13:00"},
		},
		{
			// these are supposed to always be ordered by stop_sequence, so we can directly check the first one.
			Name:   "basic fields",
			Query:  `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {trip_id stop_times(limit:1) { arrival_time departure_time stop_sequence stop_headsign pickup_type drop_off_type timepoint interpolated}} }`,
			Vars:   vars,
			Expect: `{"trips":[{"stop_times":[{"arrival_time":"05:26:00","departure_time":"05:26:00","drop_off_type":null,"interpolated":null,"pickup_type":null,"stop_headsign":"Antioch","stop_sequence":1,"timepoint":1}],"trip_id":"3850526WKDY"}]}`,
		},
		{
			// check stops for a trip
			Name:         "stop",
			Query:        `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {trip_id stop_times { stop { stop_id } }} }`,
			Vars:         vars,
			Selector:     "trips.0.stop_times.#.stop.stop_id",
			SelectExpect: []string{"SFIA", "SBRN", "SSAN", "COLM", "DALY", "BALB", "GLEN", "24TH", "16TH", "CIVC", "POWL", "MONT", "EMBR", "WOAK", "12TH", "19TH_N", "MCAR", "ROCK", "ORIN", "LAFY", "WCRK", "PHIL", "CONC", "NCON", "PITT", "PCTR", "ANTC"},
		},
		{
			// go through a stop to get trip_ids
			Name:         "trip",
			Query:        `query($stop_id: String!) {  stops(where:{stop_id:$stop_id}) {stop_times { trip { trip_id} }} }`,
			Vars:         hw{"stop_id": "70302"}, // Morgan hill
			Selector:     "stops.0.stop_times.#.trip.trip_id",
			SelectExpect: []string{"268", "274", "156"},
		},
		// check StopTimeFilter through a stop
		{
			Name:         "where service_date start_time end_time",
			Query:        `query{ stops(where:{stop_id:"MCAR_S"}) { stop_times(where:{service_date:"2018-05-30", start_time: 26000, end_time: 30000}) {arrival_time}}}`,
			Selector:     "stops.0.stop_times.#.arrival_time",
			SelectExpect: []string{"07:18:00", "07:24:00", "07:28:00", "07:33:00", "07:39:00", "07:43:00", "07:48:00", "07:54:00", "07:58:00", "08:03:00", "08:09:00", "08:18:00", "07:24:00", "07:39:00", "07:54:00", "08:09:00", "07:16:00", "07:31:00", "07:46:00", "08:01:00", "08:16:00"},
		},
		{
			Name:         "where service_date end_time",
			Query:        `query{ stops(where:{stop_id:"MCAR_S"}) { stop_times(where:{service_date:"2018-05-30", end_time: 20000}) {arrival_time}}}`,
			Selector:     "stops.0.stop_times.#.arrival_time",
			SelectExpect: []string{"04:39:00", "04:54:00", "05:09:00", "05:24:00", "04:39:00", "04:54:00", "05:09:00", "05:24:00", "04:31:00", "04:46:00", "05:01:00", "05:16:00", "05:31:00"},
		},
		{
			Name:         "where service_date start_time",
			Query:        `query{ stops(where:{stop_id:"MCAR_S"}) { stop_times(where:{service_date:"2018-05-30", start_time: 76000}) {arrival_time}}}`,
			Selector:     "stops.0.stop_times.#.arrival_time",
			SelectExpect: []string{"21:14:00", "21:34:00", "21:54:00", "22:14:00", "22:34:00", "22:54:00", "23:14:00", "23:34:00", "23:54:00", "24:14:00", "24:47:00", "21:14:00", "21:34:00", "21:54:00", "22:14:00", "22:34:00", "22:54:00", "23:14:00", "23:34:00", "23:54:00", "24:14:00", "24:47:00"},
		},
		// accept strings for Start / End
		{
			Name:         "start time string",
			Query:        `query{ stops(where:{stop_id:"RICH"}) { stop_times(where:{service_date:"2018-05-30", start: "10:00:00", end: "10:10:00"}) {departure_time}}}`,
			Selector:     "stops.0.stop_times.#.departure_time",
			SelectExpect: []string{"10:02:00", "10:05:00", "10:10:00"},
		},
		// check arrival and departure resolvers
		{
			Name:         "arrival departure base case",
			Query:        `query{ stops(where:{stop_id:"RICH"}) { stop_times(where:{service_date:"2018-05-30", start_time: 76000, end_time: 76900}) {departure_time}}}`,
			Selector:     "stops.0.stop_times.#.departure_time",
			SelectExpect: []string{"21:09:00", "21:14:00", "21:15:00"},
		},
		{
			Name:         "departures",
			Query:        `query{ stops(where:{stop_id:"RICH"}) { departures(where:{service_date:"2018-05-30", start_time: 76000, end_time: 76900}) {departure_time}}}`,
			Selector:     "stops.0.departures.#.departure_time",
			SelectExpect: []string{"21:15:00"},
		},
		{
			Name:         "arrivals",
			Query:        `query{ stops(where:{stop_id:"RICH"}) { arrivals(where:{service_date:"2018-05-30", start_time: 76000, end_time: 76900}) {arrival_time}}}`,
			Selector:     "stops.0.arrivals.#.arrival_time",
			SelectExpect: []string{"21:09:00", "21:14:00"},
		},
		// route_onestop_ids
		{
			Name:         "departure route_onestop_ids",
			Query:        `query{ stops(where:{stop_id:"RICH"}) { departures(where:{service_date:"2018-05-30", start_time: 36000, end_time: 39600}) {departure_time}}}`,
			Selector:     "stops.0.departures.#.departure_time",
			SelectExpect: []string{"10:05:00", "10:12:00", "10:20:00", "10:27:00", "10:35:00", "10:42:00", "10:50:00", "10:57:00"},
		},
		{
			Name:         "departure route_onestop_ids 1",
			Query:        `query{ stops(where:{stop_id:"RICH"}) { departures(where:{route_onestop_ids: ["r-9q8y-richmond~dalycity~millbrae"], service_date:"2018-05-30", start_time: 36000, end_time: 39600}) {departure_time}}}`,
			Selector:     "stops.0.departures.#.departure_time",
			SelectExpect: []string{"10:12:00", "10:27:00", "10:42:00", "10:57:00"},
		},
		{
			Name:         "departure route_onestop_ids 2",
			Query:        `query{ stops(where:{stop_id:"RICH"}) { departures(where:{route_onestop_ids: ["r-9q9n-warmsprings~southfremont~richmond"], service_date:"2018-05-30", start_time: 36000, end_time: 39600}) {departure_time}}}`,
			Selector:     "stops.0.departures.#.departure_time",
			SelectExpect: []string{"10:05:00", "10:20:00", "10:35:00", "10:50:00"},
		},
		// route_limit: the two routes' departures alternate in this window
		{
			Name:         "departure route_limit 1",
			Query:        `query{ stops(where:{stop_id:"RICH"}) { departures(where:{route_limit: 1, service_date:"2018-05-30", start_time: 36000, end_time: 39600}) {departure_time}}}`,
			Selector:     "stops.0.departures.#.departure_time",
			SelectExpect: []string{"10:05:00", "10:12:00"},
		},
		{
			Name:         "departure route_limit 2",
			Query:        `query{ stops(where:{stop_id:"RICH"}) { departures(where:{route_limit: 2, service_date:"2018-05-30", start_time: 36000, end_time: 39600}) {departure_time}}}`,
			Selector:     "stops.0.departures.#.departure_time",
			SelectExpect: []string{"10:05:00", "10:12:00", "10:20:00", "10:27:00"},
		},
		{
			Name:         "departure route_limit, limit still caps the stop",
			Query:        `query{ stops(where:{stop_id:"RICH"}) { departures(limit: 3, where:{route_limit: 2, service_date:"2018-05-30", start_time: 36000, end_time: 39600}) {departure_time}}}`,
			Selector:     "stops.0.departures.#.departure_time",
			SelectExpect: []string{"10:05:00", "10:12:00", "10:20:00"},
		},
	}
	return testcases
}

func stopResolverStopTimesRelativeDatesTestcases() []testcaseWithClock {
	q := `query($stop_id:String!,$relative_date:RelativeDate,$next:Int,$start:Seconds,$end:Seconds,$ed:Boolean){ stops(where:{stop_id:$stop_id}) { stop_times(where:{relative_date:$relative_date, start:$start, end:$end, next:$next use_service_window:$ed}) {trip { id trip_id } date service_date arrival_time}}}`
	testcases := []testcaseWithClock{
		{
			whenUtc: "2018-05-30T22:00:00Z",
			testcase: testcase{
				Name:  "today (wednesday)",
				Query: q,
				Vars:  hw{"stop_id": "MCAR_S", "relative_date": "TODAY", "start": "15:00:00", "end": "15:15:00"},
				Sel: []testcaseSelector{{
					Selector: "stops.0.stop_times.#.date",
					Expect:   []string{"2018-05-30", "2018-05-30", "2018-05-30"},
				}, {
					Selector: "stops.0.stop_times.#.arrival_time",
					Expect:   []string{"15:01:00", "15:09:00", "15:09:00"},
				}, {
					Selector: "stops.0.stop_times.#.trip.trip_id",
					Expect:   []string{"4611442WKDY", "3671433WKDY", "2251450WKDY"},
				}},
			},
		},
		{
			whenUtc: "2018-05-27T22:00:00Z",
			testcase: testcase{
				Name:  "today (sunday)",
				Query: q,
				Vars:  hw{"stop_id": "MCAR_S", "relative_date": "TODAY", "start": "15:00:00", "end": "15:15:00"},
				Sel: []testcaseSelector{{
					Selector: "stops.0.stop_times.#.date",
					Expect:   []string{"2018-05-27", "2018-05-27"},
				}, {
					Selector: "stops.0.stop_times.#.arrival_time",
					Expect:   []string{"15:15:00", "15:15:00"},
				}, {
					Selector: "stops.0.stop_times.#.trip.trip_id",
					Expect:   []string{"3671438SUN", "2271456SUN"},
				}},
			},
		},
		{
			whenUtc: "2018-06-04T22:00:00Z", // Monday
			testcase: testcase{
				Name:  "monday",
				Query: q,
				Vars:  hw{"stop_id": "MCAR_S", "relative_date": "MONDAY", "start": "15:00:00", "end": "15:15:00"},
				Sel: []testcaseSelector{{
					Selector: "stops.0.stop_times.#.date",
					Expect:   []string{"2018-06-04", "2018-06-04", "2018-06-04"},
				}, {
					Selector: "stops.0.stop_times.#.arrival_time",
					Expect:   []string{"15:01:00", "15:09:00", "15:09:00"},
				}, {
					Selector: "stops.0.stop_times.#.trip.trip_id",
					Expect:   []string{"4611442WKDY", "3671433WKDY", "2251450WKDY"},
				}},
			},
		},
		{
			whenUtc: "2018-06-04T22:00:00Z", // Monday
			testcase: testcase{
				Name:  "next-monday",
				Query: q,
				Vars:  hw{"stop_id": "MCAR_S", "relative_date": "NEXT_MONDAY", "start": "15:00:00", "end": "15:15:00"},
				Sel: []testcaseSelector{{
					Selector: "stops.0.stop_times.#.date",
					Expect:   []string{"2018-06-11", "2018-06-11", "2018-06-11"},
				}, {
					Selector: "stops.0.stop_times.#.trip.trip_id",
					Expect:   []string{"4611442WKDY", "3671433WKDY", "2251450WKDY"},
				}},
			},
		},
		{
			whenUtc: "2018-05-27T22:00:00Z", // Sunday
			testcase: testcase{
				Name:  "next-sunday",
				Query: q,
				Vars:  hw{"stop_id": "MCAR_S", "relative_date": "NEXT_SUNDAY", "start": "15:00:00", "end": "15:15:00"},
				Sel: []testcaseSelector{{
					Selector: "stops.0.stop_times.#.date",
					Expect:   []string{"2018-06-03", "2018-06-03"},
				}, {
					Selector: "stops.0.stop_times.#.trip.trip_id",
					Expect:   []string{"3671438SUN", "2271456SUN"},
				}},
			},
		},
		{
			whenUtc: "2018-05-25T22:00:00Z",
			testcase: testcase{
				Name:  "sunday",
				Query: q,
				Vars:  hw{"stop_id": "MCAR_S", "relative_date": "SUNDAY", "start": "15:00:00", "end": "15:15:00"},
				Sel: []testcaseSelector{{
					Selector: "stops.0.stop_times.#.trip.trip_id",
					Expect:   []string{"3671438SUN", "2271456SUN"},
				}, {
					Selector: "stops.0.stop_times.#.trip.trip_id",
					Expect:   []string{"3671438SUN", "2271456SUN"},
				}},
			},
		},
		// Combined with Next
		{
			whenUtc: "2018-06-01T19:00:00Z",
			testcase: testcase{
				Name:  "today (next=3600)",
				Query: q,
				Vars:  hw{"stop_id": "MCAR_S", "relative_date": "TODAY", "next": 900},
				Sel: []testcaseSelector{{
					Selector: "stops.0.stop_times.#.date",
					Expect:   []string{"2018-06-01", "2018-06-01", "2018-06-01"},
				}, {
					Selector: "stops.0.stop_times.#.arrival_time",
					Expect:   []string{"12:01:00", "12:09:00", "12:09:00"},
				}, {
					Selector: "stops.0.stop_times.#.trip.trip_id",
					Expect:   []string{"4591142WKDY", "3691133WKDY", "2211150WKDY"},
				}},
			},
		},
		{
			whenUtc: "2018-06-04T19:00:00Z",
			testcase: testcase{
				Name:  "next-monday (next=3600)",
				Query: q,
				Vars:  hw{"stop_id": "MCAR_S", "relative_date": "NEXT_MONDAY", "next": 900},
				Sel: []testcaseSelector{{
					Selector: "stops.0.stop_times.#.date",
					Expect:   []string{"2018-06-11", "2018-06-11", "2018-06-11"},
				}, {
					Selector: "stops.0.stop_times.#.trip.trip_id",
					Expect:   []string{"4591142WKDY", "3691133WKDY", "2211150WKDY"},
				}, {
					Selector: "stops.0.stop_times.#.arrival_time",
					Expect:   []string{"12:01:00", "12:09:00", "12:09:00"},
				}},
			},
		},
		// Combined with fallback
		{
			whenUtc: "2024-07-21T19:00:00Z",
			testcase: testcase{
				Name:  "today (sunday, outside window, next=3600)",
				Query: q,
				Vars:  hw{"stop_id": "MCAR_S", "relative_date": "TODAY", "next": 900},
				Sel: []testcaseSelector{{
					Selector: "stops.0.stop_times.#.date",
					Expect:   []string{},
				}},
			},
		},
		{
			whenUtc: "2024-07-22T19:00:00Z",
			testcase: testcase{
				Name:  "next-monday (outside window, next=3600)",
				Query: q,
				Vars:  hw{"stop_id": "MCAR_S", "relative_date": "NEXT_MONDAY", "next": 900},
				Sel: []testcaseSelector{{
					Selector: "stops.0.stop_times.#.date",
					Expect:   []string{},
				}},
			},
		},
		{
			whenUtc: "2024-07-21T19:00:00Z",
			testcase: testcase{
				Name:  "today (sunday, outside window, use fallback, next=3600)",
				Query: q,
				Vars:  hw{"stop_id": "MCAR_S", "relative_date": "TODAY", "next": 900, "ed": true},
				// The fallback week's Sunday trips, reported on the requested Sunday.
				Sel: []testcaseSelector{{
					Selector: "stops.0.stop_times.#.date",
					Expect:   []string{"2024-07-21", "2024-07-21"},
				}, {
					Selector: "stops.0.stop_times.#.arrival_time",
					Expect:   []string{"12:15:00", "12:15:00"},
				}, {
					Selector: "stops.0.stop_times.#.trip.trip_id",
					Expect:   []string{"3691138SUN", "2251156SUN"},
				}},
			},
		},
		{
			whenUtc: "2024-07-22T19:00:00Z",
			testcase: testcase{
				Name:  "next-monday (outside window, use fallback, next=3600)",
				Query: q,
				Vars:  hw{"stop_id": "MCAR_S", "relative_date": "NEXT_MONDAY", "next": 900, "ed": true},
				Sel: []testcaseSelector{{
					Selector: "stops.0.stop_times.#.date",
					Expect:   []string{"2024-07-29", "2024-07-29", "2024-07-29"},
				}, {
					Selector: "stops.0.stop_times.#.trip.trip_id",
					Expect:   []string{"4591142WKDY", "3691133WKDY", "2211150WKDY"},
				}, {
					Selector: "stops.0.stop_times.#.arrival_time",
					Expect:   []string{"12:01:00", "12:09:00", "12:09:00"},
				}},
			},
		},
	}
	return testcases
}

func stopResolverStopTimesNextTestcases() []testcaseWithClock {
	testcases := []testcaseWithClock{
		// Relative times
		{
			whenUtc: "2018-05-30T22:00:00Z",
			testcase: testcase{
				Name:         "where next 3600",
				Query:        `query{ stops(where:{stop_id:"MCAR_S"}) { stop_times(where:{next:3600}) {arrival_time}}}`,
				Selector:     "stops.0.stop_times.#.arrival_time",
				SelectExpect: []string{"15:01:00", "15:09:00", "15:09:00", "15:16:00", "15:24:00", "15:24:00", "15:31:00", "15:39:00", "15:39:00", "15:46:00", "15:54:00", "15:54:00"}, // these should start at 15:00 - 16:00

			},
		},
		{
			whenUtc: "2018-05-30T06:00:00Z", // 23:00 local time
			testcase: testcase{
				Name:         "where next 7200, includes after midnight",
				Query:        `query{ stops(where:{stop_id:"MCAR_S"}) { stop_times(where:{next:7200}) {arrival_time}}}`,
				Selector:     "stops.0.stop_times.#.arrival_time",
				SelectExpect: []string{"23:14:00", "23:14:00", "23:34:00", "23:34:00", "23:54:00", "23:54:00", "24:14:00", "24:14:00", "24:47:00", "24:47:00"},
			},
		},
		{
			whenUtc: "2018-05-30T06:00:00Z", // 23:00 local time
			testcase: testcase{
				Name:         "where next 7200, includes after midnight, check date",
				Query:        `query{ stops(where:{stop_id:"MCAR_S"}) { stop_times(where:{next:7200}) {arrival_time date}}}`,
				Selector:     "stops.0.stop_times.#.date",
				SelectExpect: []string{"2018-05-29", "2018-05-29", "2018-05-29", "2018-05-29", "2018-05-29", "2018-05-29", "2018-05-30", "2018-05-30", "2018-05-30", "2018-05-30"},
			},
		},

		{
			whenUtc: "2018-05-30T22:00:00Z",
			testcase: testcase{
				Name:         "where next 1800",
				Query:        `query{ stops(where:{stop_id:"MCAR_S"}) { stop_times(where:{next:1800}) {arrival_time}}}`,
				Selector:     "stops.0.stop_times.#.arrival_time",
				SelectExpect: []string{"15:01:00", "15:09:00", "15:09:00", "15:16:00", "15:24:00", "15:24:00"}, // these should start at 15:00 - 15:30

			},
		},
		{
			whenUtc: "2018-05-30T22:00:00Z",
			testcase: testcase{
				Name:         "where next 900, east coast",
				Query:        `query{ stops(where:{stop_id:"6497"}) { stop_times(where:{next:900}) {arrival_time}}}`,
				Selector:     "stops.0.stop_times.#.arrival_time",
				SelectExpect: []string{"18:00:00", "18:00:00", "18:00:00", "18:00:00", "18:00:00", "18:03:00", "18:10:00", "18:10:00", "18:13:00", "18:14:00", "18:15:00", "18:15:00"}, // these should start at 18:00 - 18:15

			},
		},
		{
			whenUtc: "2018-05-30T22:00:00Z",
			testcase: testcase{
				Name:  "where next 600, multiple timezones",
				Query: `query{ stops(where:{onestop_ids:["s-dhvrsm227t-universityareatransitcenter", "s-9q9p1wxf72-macarthur"]}) { onestop_id stop_id stop_times(where:{next:600}) {arrival_time}}}`,
				Vars:  hw{},
				// this test checks the json response because it is too complex for the simple element selector approach
				// we should expect east coast times 18:00-18:10, and west coast times 15:00-15:10
				Expect: `{
					"stops": [
					{
						"onestop_id": "s-9q9p1wxf72-macarthur",
						"stop_id": "MCAR",
						"stop_times": [{
							"arrival_time": "15:00:00"
						}, {
							"arrival_time": "15:07:00"
						}]
					}, {
						"onestop_id": "s-9q9p1wxf72-macarthur",
						"stop_id": "MCAR_S",
						"stop_times": [{
							"arrival_time": "15:01:00"
						}, {
							"arrival_time": "15:09:00"
						}, {
							"arrival_time": "15:09:00"
						}]
					},
					{
						"onestop_id": "s-dhvrsm227t-universityareatransitcenter",
						"stop_id": "6497",
						"stop_times": [{
							"arrival_time": "18:00:00"
						}, {
							"arrival_time": "18:00:00"
						}, {
							"arrival_time": "18:00:00"
						}, {
							"arrival_time": "18:00:00"
						}, {
							"arrival_time": "18:00:00"
						}, {
							"arrival_time": "18:03:00"
						}, {
							"arrival_time": "18:10:00"
						}, {
							"arrival_time": "18:10:00"
						}]
					}]
				}`,
			},
		},
	}
	return testcases
}

func stopResolverStopTimesFrequenciesTestcases() []testcaseWithClock {
	testcases := []testcaseWithClock{
		{
			// 6:00:00 -> 22:00:00, 1800 headway_secs; no trip starts at end_time
			testcase: testcase{
				Name:         "frequencies",
				Query:        `query{ stops(where:{feed_version_sha1: "43e2278aa272879c79460582152b04e7487f0493", stop_id:"STAGECOACH"}) { stop_times(limit:1000, where:{service_date:"2007-01-02", route_onestop_ids: ["r-9qscy-30"]}) {departure_time}}}`,
				Selector:     "stops.0.stop_times.#.departure_time",
				SelectExpect: []string{"06:00:00", "06:30:00", "07:00:00", "07:30:00", "08:00:00", "08:30:00", "09:00:00", "09:30:00", "10:00:00", "10:30:00", "11:00:00", "11:30:00", "12:00:00", "12:30:00", "13:00:00", "13:30:00", "14:00:00", "14:30:00", "15:00:00", "15:30:00", "16:00:00", "16:30:00", "17:00:00", "17:30:00", "18:00:00", "18:30:00", "19:00:00", "19:30:00", "20:00:00", "20:30:00", "21:00:00", "21:30:00"},
			},
		},
		{
			// Verified; multiple frequencies over course of day
			testcase: testcase{
				Name:         "frequencies",
				Query:        `query{ stops(where:{feed_version_sha1: "43e2278aa272879c79460582152b04e7487f0493", stop_id:"NADAV"}) { stop_times(limit:1000, where:{service_date:"2007-01-02"}) {departure_time}}}`,
				Selector:     "stops.0.stop_times.#.departure_time",
				SelectExpect: []string{"06:14:00", "06:14:00", "06:44:00", "06:44:00", "07:14:00", "07:14:00", "07:44:00", "07:44:00", "08:14:00", "08:14:00", "08:24:00", "08:24:00", "08:34:00", "08:34:00", "08:44:00", "08:44:00", "08:54:00", "08:54:00", "09:04:00", "09:04:00", "09:14:00", "09:14:00", "09:24:00", "09:24:00", "09:34:00", "09:34:00", "09:44:00", "09:44:00", "09:54:00", "09:54:00", "10:04:00", "10:04:00", "10:14:00", "10:14:00", "10:44:00", "10:44:00", "11:14:00", "11:14:00", "11:44:00", "11:44:00", "12:14:00", "12:14:00", "12:44:00", "12:44:00", "13:14:00", "13:14:00", "13:44:00", "13:44:00", "14:14:00", "14:14:00", "14:44:00", "14:44:00", "15:14:00", "15:14:00", "15:44:00", "15:44:00", "16:14:00", "16:14:00", "16:24:00", "16:24:00", "16:34:00", "16:34:00", "16:44:00", "16:44:00", "16:54:00", "16:54:00", "17:04:00", "17:04:00", "17:14:00", "17:14:00", "17:24:00", "17:24:00", "17:34:00", "17:34:00", "17:44:00", "17:44:00", "17:54:00", "17:54:00", "18:04:00", "18:04:00", "18:14:00", "18:14:00", "18:24:00", "18:24:00", "18:34:00", "18:34:00", "18:44:00", "18:44:00", "18:54:00", "18:54:00", "19:04:00", "19:04:00", "19:14:00", "19:14:00", "19:44:00", "19:44:00", "20:14:00", "20:14:00", "20:44:00", "20:44:00", "21:14:00", "21:14:00", "21:44:00", "21:44:00"},
			},
		},
	}
	return testcases
}
