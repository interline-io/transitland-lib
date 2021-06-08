package resolvers

import (
	"testing"

	"github.com/99designs/gqlgen/client"
)

func TestCalendarResolver(t *testing.T) {
	vars := hw{"trip_id": "3850526WKDY"}
	testcases := []testcase{
		{
			"basic fields",
			`query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {calendar{service_id start_date end_date monday tuesday wednesday thursday friday saturday sunday} }}`,
			vars,
			`{"trips":[{"calendar":{"end_date":"2019-07-01T00:00:00Z","friday":1,"monday":1,"saturday":0,"service_id":"WKDY","start_date":"2018-05-26T00:00:00Z","sunday":0,"thursday":1,"tuesday":1,"wednesday":1}}]}`,
			"",
			nil,
		},
		// TODO: added_dates
		// TODO: removed_dates
	}
	c := client.New(newServer())
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testquery(t, c, tc)
		})
	}
}
