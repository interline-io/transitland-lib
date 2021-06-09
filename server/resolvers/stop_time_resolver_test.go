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
	}
	c := client.New(NewServer())
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testquery(t, c, tc)
		})
	}
}
