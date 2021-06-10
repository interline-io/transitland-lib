package resolvers

import (
	"testing"

	"github.com/99designs/gqlgen/client"
)

func TestStopTimeResolver(t *testing.T) {
	vars := hw{"trip_id": "3850526WKDY"}
	testcases := []testcase{
		{
			"basic",
			`query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {trip_id stop_times { arrival_time }} }`,
			vars,
			``,
			"trips.0.stop_times.#.arrival_time",
			[]string{"19560", "19740", "19980", "20160", "20400", "20580", "20760", "20880", "21000", "21180", "21240", "21360", "21480", "21900", "22080", "22260", "22500", "22620", "22980", "23220", "23520", "23700", "24000", "24180", "24600", "25500", "25980"},
		},
		{
			// these are supposed to always be ordered by stop_sequence, so we can directly check the first one.
			"basic fields",
			`query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {trip_id stop_times(limit:1) { arrival_time departure_time stop_sequence stop_headsign pickup_type drop_off_type timepoint interpolated}} }`,
			vars,
			`{"trips":[{"stop_times":[{"arrival_time":19560,"departure_time":19560,"drop_off_type":null,"interpolated":null,"pickup_type":null,"stop_headsign":"Antioch","stop_sequence":1,"timepoint":1}],"trip_id":"3850526WKDY"}]}`,
			"",
			nil,
		},
		{
			// check stops for a trip
			"stop",
			`query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {trip_id stop_times { stop { stop_id } }} }`,
			vars,
			``,
			"trips.0.stop_times.#.stop.stop_id",
			[]string{"SFIA", "SBRN", "SSAN", "COLM", "DALY", "BALB", "GLEN", "24TH", "16TH", "CIVC", "POWL", "MONT", "EMBR", "WOAK", "12TH", "19TH_N", "MCAR", "ROCK", "ORIN", "LAFY", "WCRK", "PHIL", "CONC", "NCON", "PITT", "PCTR", "ANTC"},
		},
		{
			// go through a stop to get trip_ids
			"trip",
			`query($stop_id: String!) {  stops(where:{stop_id:$stop_id}) {stop_times { trip { trip_id} }} }`,
			hw{"stop_id": "70302"}, // Morgan hill
			``,
			"stops.0.stop_times.#.trip.trip_id",
			[]string{"268", "274", "156"},
		},
		// check StopTimeFilter through a stop
		{
			"where service_date start_time end_time",
			`query{ stops(where:{stop_id:"MCAR_S"}) { stop_times(where:{service_date:"2018-05-30", start_time: 26000, end_time: 30000}) {arrival_time}}}`,
			hw{},
			``,
			"stops.0.stop_times.#.arrival_time",
			[]string{"26280", "26640", "26880", "27180", "27540", "27780", "28080", "28440", "28680", "28980", "29340", "29880", "26640", "27540", "28440", "29340", "26160", "27060", "27960", "28860", "29760"},
		},
		{
			"where service_date end_time",
			`query{ stops(where:{stop_id:"MCAR_S"}) { stop_times(where:{service_date:"2018-05-30", end_time: 20000}) {arrival_time}}}`,
			hw{},
			``,
			"stops.0.stop_times.#.arrival_time",
			[]string{"16740", "17640", "18540", "19440", "16740", "17640", "18540", "19440", "16260", "17160", "18060", "18960", "19860"},
		},
		{
			"where service_date start_time",
			`query{ stops(where:{stop_id:"MCAR_S"}) { stop_times(where:{service_date:"2018-05-30", start_time: 76000}) {arrival_time}}}`,
			hw{},
			``,
			"stops.0.stop_times.#.arrival_time",
			[]string{"76440", "77640", "78840", "80040", "81240", "82440", "83640", "84840", "86040", "87240", "89220", "76440", "77640", "78840", "80040", "81240", "82440", "83640", "84840", "86040", "87240", "89220"},
		},
	}
	c := client.New(NewServer())
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testquery(t, c, tc)
		})
	}
}
