package gql

import (
	"fmt"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
)

func TestStopResolver_StopTimes(t *testing.T) {
	testcases := []testcase{
		// Allow previous route onestop ids
		// OLD: r-9q9n-fremont~richmond
		// NEW: r-9q9n-warmsprings~southfremont~richmond
		{
			name:         "departure route_onestop_ids use previous id current ok",
			query:        `query{ stops(where:{stop_id:"RICH"}) { departures(where:{allow_previous_route_onestop_ids: false, route_onestop_ids: ["r-9q9n-warmsprings~southfremont~richmond"], service_date:"2018-05-30", start_time: 36000, end_time: 39600}) {departure_time}}}`,
			selector:     "stops.0.departures.#.departure_time",
			selectExpect: []string{"10:05:00", "10:20:00", "10:35:00", "10:50:00"},
		},
		{
			name:         "departure route_onestop_ids, use previous id, both at once ok",
			query:        `query{ stops(where:{stop_id:"RICH"}) { departures(where:{allow_previous_route_onestop_ids: false, route_onestop_ids: ["r-9q9n-warmsprings~southfremont~richmond","r-9q9n-fremont~richmond"], service_date:"2018-05-30", start_time: 36000, end_time: 39600}) {departure_time}}}`,
			selector:     "stops.0.departures.#.departure_time",
			selectExpect: []string{"10:05:00", "10:20:00", "10:35:00", "10:50:00"},
		},
		{
			name:         "departure route_onestop_ids, use previous id, both at once, no duplicates",
			query:        `query{ stops(where:{stop_id:"RICH"}) { departures(where:{allow_previous_route_onestop_ids: true, route_onestop_ids: ["r-9q9n-warmsprings~southfremont~richmond","r-9q9n-fremont~richmond"], service_date:"2018-05-30", start_time: 36000, end_time: 39600}) {departure_time}}}`,
			selector:     "stops.0.departures.#.departure_time",
			selectExpect: []string{"10:05:00", "10:20:00", "10:35:00", "10:50:00"},
		},
		{
			name:         "departure route_onestop_ids, use previous id, old, fail",
			query:        `query{ stops(where:{stop_id:"RICH"}) { departures(where:{allow_previous_route_onestop_ids: false, route_onestop_ids: ["r-9q9n-fremont~richmond"], service_date:"2018-05-30", start_time: 36000, end_time: 39600}) {departure_time}}}`,
			selector:     "stops.0.departures.#.departure_time",
			selectExpect: []string{},
		},
		{
			name:         "departure route_onestop_ids, use previous id, old, ok",
			query:        `query{ stops(where:{stop_id:"RICH"}) { departures(where:{allow_previous_route_onestop_ids: true, route_onestop_ids: ["r-9q9n-fremont~richmond"], service_date:"2018-05-30", start_time: 36000, end_time: 39600}) {departure_time}}}`,
			selector:     "stops.0.departures.#.departure_time",
			selectExpect: []string{"10:05:00", "10:20:00", "10:35:00", "10:50:00"},
		},
	}
	for _, materialized := range []bool{false, true} {
		t.Run(fmt.Sprintf("UseMaterializedDepartures=%t", materialized), func(t *testing.T) {
			c, _ := newTestClientWithOpts(t, testconfig.Options{
				RTJsons:                   testconfig.DefaultRTJson(),
				UseMaterializedDepartures: materialized,
			})
			queryTestcases(t, c, testcases)
		})
	}
}
