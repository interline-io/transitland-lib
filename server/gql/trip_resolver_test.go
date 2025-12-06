package gql

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestTripResolver(t *testing.T) {
	vars := hw{"trip_id": "3850526WKDY"}
	testcases := []testcase{
		{
			name:   "basic fields",
			query:  `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {trip_id trip_headsign trip_short_name direction_id block_id wheelchair_accessible bikes_allowed stop_pattern_id }}`,
			vars:   vars,
			expect: `{"trips":[{"bikes_allowed":1,"block_id":null,"direction_id":1,"stop_pattern_id":21,"trip_headsign":"Antioch","trip_id":"3850526WKDY","trip_short_name":null,"wheelchair_accessible":1}]}`,
		},
		{
			name:   "calendar",
			query:  `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {calendar {service_id} }}`,
			vars:   vars,
			expect: `{"trips":[{"calendar":{"service_id":"WKDY"}}]}`,
		},
		{
			name:   "route",
			query:  `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {route {route_id} }}`,
			vars:   vars,
			expect: `{"trips":[{"route":{"route_id":"01"}}]}`,
		},
		{
			name:   "shape",
			query:  `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {shape {shape_id} }}`,
			vars:   vars,
			expect: `{"trips":[{"shape":{"shape_id":"02_shp"}}]}`,
		},
		{
			name:   "feed_version",
			query:  `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {feed_version {sha1} }}`,
			vars:   vars,
			expect: `{"trips":[{"feed_version":{"sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0"}}]}`,
		},
		{
			name:         "stop_times",
			query:        `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {stop_times {stop_sequence} }}`,
			vars:         vars,
			selector:     "trips.0.stop_times.#.stop_sequence",
			selectExpect: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16", "17", "18", "19", "20", "21", "22", "23", "24", "25", "26", "27"},
		},
		{
			name:         "stop_times end time",
			query:        `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {stop_times(where:{end:"05:45:00"}) {stop_sequence arrival_time departure_time} }}`,
			vars:         vars,
			selector:     "trips.0.stop_times.#.arrival_time",
			selectExpect: []string{"05:26:00", "05:29:00", "05:33:00", "05:36:00", "05:40:00", "05:43:00"},
		},
		{
			name:         "stop_times start time",
			query:        `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {stop_times(where:{start:"06:00:00"}) {stop_sequence arrival_time departure_time} }}`,
			vars:         vars,
			selector:     "trips.0.stop_times.#.arrival_time",
			selectExpect: []string{"06:05:00", "06:08:00", "06:11:00", "06:15:00", "06:17:00", "06:23:00", "06:27:00", "06:32:00", "06:35:00", "06:40:00", "06:43:00", "06:50:00", "07:05:00", "07:13:00"},
		},
		{
			name:         "stop_times start and end time",
			query:        `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {stop_times(where:{start:"06:00:00", end:"06:30:00"}) {stop_sequence arrival_time departure_time} }}`,
			vars:         vars,
			selector:     "trips.0.stop_times.#.arrival_time",
			selectExpect: []string{"06:05:00", "06:08:00", "06:11:00", "06:15:00", "06:17:00", "06:23:00", "06:27:00"},
		},
		{
			name:         "stop_times multiple 1",
			query:        `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {st1: stop_times(limit:5, where:{start:"06:00:00"}) {stop_sequence arrival_time departure_time}, st2: stop_times(limit: 3, where:{start:"06:00:00"}) {stop_sequence arrival_time departure_time}}}`,
			vars:         vars,
			selector:     "trips.0.st1.#.departure_time",
			selectExpect: []string{"06:05:00", "06:08:00", "06:11:00", "06:15:00", "06:17:00"},
		},
		{
			name:         "stop_times multiple 2",
			query:        `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {st1: stop_times(limit:5, where:{start:"06:00:00"}) {stop_sequence arrival_time departure_time}, st2: stop_times(limit: 3, where:{start:"06:00:00"}) {stop_sequence arrival_time departure_time}}}`,
			vars:         vars,
			selector:     "trips.0.st2.#.departure_time",
			selectExpect: []string{"06:05:00", "06:08:00", "06:11:00"},
		},
		{
			name:         "where trip_id",
			query:        `query{  trips(where:{trip_id:"3850526WKDY"}) {trip_id}}`,
			vars:         vars,
			selector:     "trips.#.trip_id",
			selectExpect: []string{"3850526WKDY"},
		},
		{
			name:  "where service_date",
			query: `query{trips(where:{feed_onestop_id:"CT",service_date:"2018-05-29"}){trip_id}}`,

			selector:     "trips.#.trip_id",
			selectExpect: []string{"101", "103", "305", "207", "309", "211", "313", "215", "217", "319", "221", "323", "225", "227", "329", "231", "233", "135", "237", "139", "143", "147", "151", "155", "257", "159", "261", "263", "365", "267", "269", "371", "273", "375", "277", "279", "381", "283", "385", "287", "289", "191", "193", "195", "197", "199", "102", "104", "206", "208", "310", "212", "314", "216", "218", "320", "222", "324", "226", "228", "330", "232", "134", "236", "138", "142", "146", "150", "152", "254", "156", "258", "360", "262", "264", "366", "268", "370", "272", "274", "376", "278", "380", "282", "284", "386", "288", "190", "192", "194", "196", "198"},
		},
		// license
		{
			name:         "license filter: share_alike_optional = yes",
			query:        `query($lic:LicenseFilter) {trips(limit:1,where: {license: $lic}) {trip_id feed_version{feed{license{share_alike_optional}}}}}`,
			vars:         hw{"lic": hw{"share_alike_optional": "YES"}},
			selector:     "trips.0.feed_version.feed.license.share_alike_optional",
			selectExpect: []string{"yes"},
		},
		{
			name:         "license filter: share_alike_optional = no",
			query:        `query($lic:LicenseFilter) {trips(limit:1,where: {license: $lic}) {trip_id feed_version{feed{license{share_alike_optional}}}}}`,
			vars:         hw{"lic": hw{"share_alike_optional": "NO"}},
			selector:     "trips.0.feed_version.feed.license.share_alike_optional",
			selectExpect: []string{"no"},
		},
		{
			name:         "license filter: create_derived_product = yes",
			query:        `query($lic:LicenseFilter) {trips(limit:1,where: {license: $lic}) {trip_id feed_version{feed{license{create_derived_product}}}}}`,
			vars:         hw{"lic": hw{"create_derived_product": "YES"}},
			selector:     "trips.0.feed_version.feed.license.create_derived_product",
			selectExpect: []string{"yes"},
		},
		{
			name:         "license filter: create_derived_product = no",
			query:        `query($lic:LicenseFilter) {trips(limit:1,where: {license: $lic}) {trip_id feed_version{feed{license{create_derived_product}}}}}`,
			vars:         hw{"lic": hw{"create_derived_product": "NO"}},
			selector:     "trips.0.feed_version.feed.license.create_derived_product",
			selectExpect: []string{"no"},
		},
		{
			name:         "license filter: commercial_use_allowed = yes",
			query:        `query($lic:LicenseFilter) {trips(limit:1,where: {license: $lic}) {trip_id feed_version{feed{license{commercial_use_allowed}}}}}`,
			vars:         hw{"lic": hw{"commercial_use_allowed": "YES"}},
			selector:     "trips.0.feed_version.feed.license.commercial_use_allowed",
			selectExpect: []string{"yes"},
		},
		{
			name:         "license filter: commercial_use_allowed = no",
			query:        `query($lic:LicenseFilter) {trips(limit:1,where: {license: $lic}) {trip_id feed_version{feed{license{commercial_use_allowed}}}}}`,
			vars:         hw{"lic": hw{"commercial_use_allowed": "NO"}},
			selector:     "trips.0.feed_version.feed.license.commercial_use_allowed",
			selectExpect: []string{"no"},
		},

		// TODO: check where feed_version_sha1, feed_onestop_id but only check count
		// TODO: frequencies
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

func TestTripResolver_StopPatternID(t *testing.T) {
	query := `query {
		trips(where: {feed_onestop_id: "BA", trip_id:"3230742WKDY"}) {
		  trip_id
		  stop_pattern_id
		}
	}`
	c, _ := newTestClient(t)
	var resp map[string]interface{}
	if err := c.Post(query, &resp); err != nil {
		t.Error(err)
		return
	}
	jj := toJson(resp)
	patId := gjson.Get(jj, "trips.0.stop_pattern_id").Int()
	tc := testcase{
		name: "where trip_id",
		query: `query($patid:Int!) {
			trips(where: {feed_onestop_id: "BA", stop_pattern_id:$patid}) {
			  trip_id
			  stop_pattern_id
			}
		  }
		`,
		vars:         hw{"patid": patId},
		expect:       ``,
		selector:     "trips.#.trip_id",
		selectExpect: []string{"3230742WKDY", "3250757WKDY", "3270812WKDY", "3310827WKDY", "3210842WKDY"},
	}
	t.Run(tc.name, func(t *testing.T) {
		queryTestcase(t, c, tc)
	})
}

func TestTripResolver_License(t *testing.T) {
	q := `
	query ($lic: LicenseFilter) {
		trips(limit: 100000, where: {license: $lic}) {
		  trip_id
		  feed_version {
			feed {
			  onestop_id
			  license {
				share_alike_optional
				create_derived_product
				commercial_use_allowed
				redistribution_allowed
			  }
			}
		  }
		}
	  }
	`
	testcases := []testcase{
		// license: share_alike_optional
		{
			name:               "license filter: share_alike_optional = yes",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "YES"}},
			selector:           "trips.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  14718,
		},
		{
			name:               "license filter: share_alike_optional = no",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "NO"}},
			selector:           "trips.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  2525,
		},
		{
			name:               "license filter: share_alike_optional = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "EXCLUDE_NO"}},
			selector:           "trips.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "ctran-flex"},
			selectExpectCount:  16292,
		},
		// license: create_derived_product
		{
			name:               "license filter: create_derived_product = yes",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "YES"}},
			selector:           "trips.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  14718,
		},
		{
			name:               "license filter: create_derived_product = no",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "NO"}},
			selector:           "trips.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  2525,
		},
		{
			name:               "license filter: create_derived_product = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "EXCLUDE_NO"}},
			selector:           "trips.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "ctran-flex"},
			selectExpectCount:  16292,
		},
		// license: commercial_use_allowed
		{
			name:               "license filter: commercial_use_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "YES"}},
			selector:           "trips.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  14718,
		},
		{
			name:               "license filter: commercial_use_allowed = no",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "NO"}},
			selector:           "trips.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  2525,
		},
		{
			name:               "license filter: commercial_use_allowed = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "EXCLUDE_NO"}},
			selector:           "trips.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "ctran-flex"},
			selectExpectCount:  16292,
		},
		// license: redistribution_allowed
		{
			name:               "license filter: redistribution_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "YES"}},
			selector:           "trips.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  14718,
		},
		{
			name:               "license filter: redistribution_allowed = no",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "NO"}},
			selector:           "trips.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  2525,
		},
		{
			name:               "license filter: redistribution_allowed = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "EXCLUDE_NO"}},
			selector:           "trips.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "ctran-flex"},
			selectExpectCount:  16292,
		},
		// license: use_without_attribution
		{
			name:               "license filter: use_without_attribution = yes",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "YES"}},
			selector:           "trips.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  14718,
		},
		{
			name:               "license filter: use_without_attribution = no",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "NO"}},
			selector:           "trips.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  2525,
		},
		{
			name:               "license filter: use_without_attribution = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "EXCLUDE_NO"}},
			selector:           "trips.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "ctran-flex"},
			selectExpectCount:  16292,
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}
