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

func TestFeedVersionResolver_Calendars(t *testing.T) {
	baSha1 := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
	// CT defines three services in calendar.txt and 24 more only in calendar_dates.txt.
	ctSha1 := "d2813c293bcfd7a97dde599527ae6c62c98e66c6"
	q := `query($sha1:String!, $where:CalendarFilter) {
		feed_versions(where:{sha1:$sha1}) {
			calendars(limit:1000, where:$where) { service_id start_date end_date monday saturday sunday generated }
		}
	}`
	testcases := []testcase{
		{
			name:         "every calendar in the feed version",
			query:        q,
			vars:         hw{"sha1": baSha1},
			selector:     "feed_versions.0.calendars.#.service_id",
			selectExpect: []string{"WKDY", "SAT", "SUN"},
		},
		{
			name:   "service_id",
			query:  q,
			vars:   hw{"sha1": baSha1, "where": hw{"service_id": "SAT"}},
			expect: `{"feed_versions":[{"calendars":[{"service_id":"SAT","start_date":"2018-05-26","end_date":"2019-07-01","monday":0,"saturday":1,"sunday":0,"generated":false}]}]}`,
		},
		{
			name:  "generated calendars",
			query: q,
			vars:  hw{"sha1": ctSha1},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.calendars.#.service_id", expectCount: 27},
				{selector: "feed_versions.0.calendars.#(generated==false)#.service_id", expect: []string{"mtwtf", "sat_sun", "sat_extra"}},
				{selector: "feed_versions.0.calendars.#(generated==true)#.service_id", expectCount: 24},
			},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

func TestFeedVersionResolver_CalendarDates(t *testing.T) {
	// BA removes weekday service and adds Sunday service on six holidays.
	baSha1 := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
	q := `query($sha1:String!, $where:CalendarDateFilter) {
		feed_versions(where:{sha1:$sha1}) {
			calendar_dates(where:$where) { date exception_type service { service_id } feed_version_sha1 feed_onestop_id feed_version { sha1 } }
		}
	}`
	testcases := []testcase{
		{
			name:  "every calendar date in the feed version",
			query: q,
			vars:  hw{"sha1": baSha1},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.calendar_dates.#.date", expectCount: 12},
				{selector: "feed_versions.0.calendar_dates.#.feed_version_sha1", expectUnique: []string{baSha1}},
				{selector: "feed_versions.0.calendar_dates.#.feed_onestop_id", expectUnique: []string{"BA"}},
				{selector: "feed_versions.0.calendar_dates.#.feed_version.sha1", expectUnique: []string{baSha1}},
			},
		},
		{
			name:  "date",
			query: q,
			vars:  hw{"sha1": baSha1, "where": hw{"date": "2018-07-04"}},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.calendar_dates.#.service.service_id", expect: []string{"WKDY", "SUN"}},
				{selector: "feed_versions.0.calendar_dates.#.exception_type", expect: []string{"2", "1"}},
			},
		},
		{
			name:  "exception_type",
			query: q,
			vars:  hw{"sha1": baSha1, "where": hw{"exception_type": 2}},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.calendar_dates.#.date", expect: []string{"2018-05-28", "2018-07-04", "2018-09-03", "2018-11-22", "2018-12-25", "2019-01-01"}},
				{selector: "feed_versions.0.calendar_dates.#.service.service_id", expectUnique: []string{"WKDY"}},
			},
		},
		{
			name:         "date and exception_type",
			query:        q,
			vars:         hw{"sha1": baSha1, "where": hw{"date": "2018-07-04", "exception_type": 1}},
			selector:     "feed_versions.0.calendar_dates.#.service.service_id",
			selectExpect: []string{"SUN"},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}
