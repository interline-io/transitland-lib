package gqltest

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
)

func calendarResolverTestcases(t testing.TB, cfg model.Config) []testcase {
	vars := hw{"trip_id": "3850526WKDY"}
	testcases := []testcase{
		{
			Name:   "basic fields",
			Query:  `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {calendar{service_id start_date end_date monday tuesday wednesday thursday friday saturday sunday} }}`,
			Vars:   vars,
			Expect: `{"trips":[{"calendar":{"end_date":"2019-07-01","friday":1,"monday":1,"saturday":0,"service_id":"WKDY","start_date":"2018-05-26","sunday":0,"thursday":1,"tuesday":1,"wednesday":1}}]}`,
		},
		// these will always be ordered by date
		{
			Name:   "added_dates",
			Query:  `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {calendar{added_dates} }}`,
			Vars:   vars,
			Expect: `{"trips":[{"calendar":{"added_dates":[]}}]}`,
		},
		{
			Name:   "removed_dates",
			Query:  `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {calendar{removed_dates} }}`,
			Vars:   vars,
			Expect: `{"trips":[{"calendar":{"removed_dates":["2018-05-28","2018-07-04","2018-09-03","2018-11-22","2018-12-25","2019-01-01"]}}]}`,
		},
	}
	return testcases
}
