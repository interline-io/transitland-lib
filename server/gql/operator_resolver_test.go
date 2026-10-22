package gql

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/gql/internal/gqlcase"
)

func TestOperatorResolver(t *testing.T) {
	testcases := []testcase{
		{
			Name:         "places iso3166 country",
			Query:        `query { operators(where:{adm0_iso: "US"}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"},
		},
		{
			Name:         "places iso3166 state",
			Query:        `query { operators(where:{adm1_iso: "US-CA"}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			Name:         "places iso3166 state not found",
			Query:        `query { operators(where:{adm1_iso: "US-NY"}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:         "places adm0_name",
			Query:        `query { operators(where:{adm0_name: "United States of America"}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"},
		},
		{
			Name:         "places adm1_name",
			Query:        `query { operators(where:{adm1_name: "California"}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			Name:         "places adm1_name not found",
			Query:        `query { operators(where:{adm1_name: "Nowhere"}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:         "places city_name",
			Query:        `query { operators(where:{city_name: "Oakland"}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
	}
	c, _ := newTestClient(t)
	gqlcase.QueryTestcases(t, c, testcases)
}
