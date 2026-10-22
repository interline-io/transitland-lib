package gql

import (
	"testing"
)

func TestOperatorResolver(t *testing.T) {
	testcases := []testcase{
		{
			name:         "places iso3166 country",
			query:        `query { operators(where:{adm0_iso: "US"}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"},
		},
		{
			name:         "places iso3166 state",
			query:        `query { operators(where:{adm1_iso: "US-CA"}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			name:         "places iso3166 state not found",
			query:        `query { operators(where:{adm1_iso: "US-NY"}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{},
		},
		{
			name:         "places adm0_name",
			query:        `query { operators(where:{adm0_name: "United States of America"}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"},
		},
		{
			name:         "places adm1_name",
			query:        `query { operators(where:{adm1_name: "California"}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			name:         "places adm1_name not found",
			query:        `query { operators(where:{adm1_name: "Nowhere"}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{},
		},
		{
			name:         "places city_name",
			query:        `query { operators(where:{city_name: "Oakland"}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}
