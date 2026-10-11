package gql

import (
	"testing"
)

func TestCalendarResolver(t *testing.T) {
	vars := hw{"trip_id": "3850526WKDY"}
	testcases := []testcase{
		{
			name:   "basic fields",
			query:  `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {calendar{service_id start_date end_date monday tuesday wednesday thursday friday saturday sunday} }}`,
			vars:   vars,
			expect: `{"trips":[{"calendar":{"end_date":"2019-07-01","friday":1,"monday":1,"saturday":0,"service_id":"WKDY","start_date":"2018-05-26","sunday":0,"thursday":1,"tuesday":1,"wednesday":1}}]}`,
		},
		// these will always be ordered by date
		{
			name:   "added_dates",
			query:  `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {calendar{added_dates} }}`,
			vars:   vars,
			expect: `{"trips":[{"calendar":{"added_dates":[]}}]}`,
		},
		{
			name:   "removed_dates",
			query:  `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {calendar{removed_dates} }}`,
			vars:   vars,
			expect: `{"trips":[{"calendar":{"removed_dates":["2018-05-28","2018-07-04","2018-09-03","2018-11-22","2018-12-25","2019-01-01"]}}]}`,
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}
