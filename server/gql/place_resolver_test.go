package gql

import (
	"testing"
)

func TestPlaceResolver(t *testing.T) {
	q := `query($level: PlaceAggregationLevel,$where: PlaceFilter) {
		places(level: $level, where: $where) {
			adm0_name
			adm1_name
			city_name
			count
			operators {
				onestop_id
			}
		}
	}`
	testcases := []testcase{
		{
			name:         "ADM0",
			query:        q,
			vars:         hw{"level": "ADM0"},
			selector:     "places.#.adm0_name",
			selectExpect: []string{"United States of America"},
		},
		{
			name:         "ADM0 count",
			query:        q,
			vars:         hw{"level": "ADM0"},
			selector:     "places.#.count",
			selectExpect: []string{"4"},
		},
		{
			name:         "ADM0 where",
			query:        q,
			vars:         hw{"level": "ADM0", "where": hw{"adm0_name": "United States of America"}},
			selector:     "places.#.count",
			selectExpect: []string{"4"},
		},
		{
			name:         "ADM0 where 2",
			query:        q,
			vars:         hw{"level": "ADM0", "where": hw{"adm0_name": "Canada"}},
			selector:     "places.#.count",
			selectExpect: []string{},
		},
		{
			name:         "ADM0_ADM1",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1"},
			selector:     "places.#.adm1_name",
			selectExpect: []string{"California", "District of Columbia", "Florida", "Maryland", "Virginia"},
		},
		{
			name:         "ADM0_ADM1 count",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1"},
			selector:     "places.#.count",
			selectExpect: []string{"2", "1", "1", "1", "1"},
		},
		{
			name:         "ADM0_ADM1 where",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1", "where": hw{"adm1_name": "California"}},
			selector:     "places.#.count",
			selectExpect: []string{"2"},
		},
		{
			name:         "ADM0_ADM1_CITY",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1_CITY"},
			selector:     "places.#.city_name",
			selectExpect: []string{"Berkeley", "Oakland", "San Francisco", "San Jose", "San Mateo", "Tampa", "", "", "Washington,  D.C.", "Alexandria", "", "", ""},
		},
		{
			name:         "ADM0_ADM1_CITY where",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"city_name": "Oakland"}},
			selector:     "places.#.city_name",
			selectExpect: []string{"Oakland"},
		},
		// bbox
		{
			name:         "region bbox comes from the admin polygon",
			query:        `query{ places(level: ADM0_ADM1, where: {adm1_name: "California"}) { bbox } }`,
			selector:     "places.0.bbox.coordinates.0.0.0",
			selectExpect: []string{"-124.4092019709999"},
		},
		{
			// Every region, including those with no operators at all.
			name:         "country bbox reaches every region",
			query:        `query{ places(level: ADM0, where: {adm0_name: "United States of America"}) { bbox } }`,
			selector:     "places.0.bbox.coordinates.0.0.0",
			selectExpect: []string{"-179.1435033839999"},
		},
		{
			// A city is a point in Natural Earth, buffered by the radius the place
			// association itself uses.
			name:         "city bbox is buffered around the point",
			query:        `query{ places(level: ADM0_ADM1_CITY, where: {city_name: "Oakland"}) { bbox } }`,
			selector:     "places.0.bbox.coordinates.0.0.0",
			selectExpect: []string{"-122.73241922786842"},
		},
		{
			// A level that identifies a city by fewer names still gets a box.
			name:         "city bbox at a coarser level",
			query:        `query{ places(level: ADM0_CITY, where: {city_name: "Oakland"}) { bbox } }`,
			selector:     "places.0.bbox.coordinates.0.0.0",
			selectExpect: []string{"-122.73241922786842"},
		},
		// operators
		{
			name:         "ADM0 operators",
			query:        q,
			vars:         hw{"level": "ADM0"},
			selector:     "places.0.operators.#.onestop_id",
			selectExpect: []string{"o-dhv-hillsborougharearegionaltransit", "o-9q9-bayarearapidtransit", "o-9q9-caltrain", "o-dqcj-wmata"},
		},
		{
			name:         "ADM0_ADM1 operators",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1", "where": hw{"adm1_name": "California"}},
			selector:     "places.0.operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			name:         "ADM0_ADM1_CITY operators",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"city_name": "Oakland"}},
			selector:     "places.0.operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}
