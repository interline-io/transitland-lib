package resolvers

import (
	"testing"

	"github.com/99designs/gqlgen/client"
)

func TestOperatorResolver(t *testing.T) {
	testcases := []testcase{
		{
			"basic fields",
			`query{operators(where:{onestop_id:"o-9q9-bayarearapidtransit"}) {onestop_id city_name adm1name adm0name}}`,
			hw{},
			`{"operators":[{"adm0name":"United States of America","adm1name":"California","city_name":null,"onestop_id":"o-9q9-bayarearapidtransit"}]}`,
			"",
			nil,
		},
	}
	c := client.New(NewServer())
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testquery(t, c, tc)
		})
	}
}
