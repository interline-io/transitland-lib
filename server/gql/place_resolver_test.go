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
			selectExpect: []string{"3"},
		},
		{
			name:         "ADM0 where",
			query:        q,
			vars:         hw{"level": "ADM0", "where": hw{"adm0_name": "United States of America"}},
			selector:     "places.#.count",
			selectExpect: []string{"3"},
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
			selectExpect: []string{"California", "Florida"},
		},
		{
			name:         "ADM0_ADM1 count",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1"},
			selector:     "places.#.count",
			selectExpect: []string{"1", "2"},
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
			selectExpect: []string{"Berkeley", "Oakland", "San Francisco", "San Jose", "San Mateo", "Tampa", "", ""},
		},
		{
			name:         "ADM0_ADM1_CITY where",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"city_name": "Oakland"}},
			selector:     "places.#.city_name",
			selectExpect: []string{"Oakland"},
		},
		// operators
		{
			name:         "ADM0 operators",
			query:        q,
			vars:         hw{"level": "ADM0"},
			selector:     "places.0.operators.#.onestop_id",
			selectExpect: []string{"o-dhv-hillsborougharearegionaltransit", "o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
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
