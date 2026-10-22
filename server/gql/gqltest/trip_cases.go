package gqltest

import (
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/server/gql/internal/gqlcase"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/tidwall/gjson"
)

func tripResolverTestcases(t testing.TB, cfg model.Config) []testcase {
	vars := hw{"trip_id": "3850526WKDY"}
	testcases := []testcase{
		{
			Name:   "basic fields",
			Query:  `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {trip_id trip_headsign trip_short_name direction_id block_id wheelchair_accessible bikes_allowed cars_allowed }}`,
			Vars:   vars,
			Expect: `{"trips":[{"bikes_allowed":1,"block_id":null,"cars_allowed":null,"direction_id":1,"trip_headsign":"Antioch","trip_id":"3850526WKDY","trip_short_name":null,"wheelchair_accessible":1}]}`,
		},
		{
			Name:   "calendar",
			Query:  `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {calendar {service_id} }}`,
			Vars:   vars,
			Expect: `{"trips":[{"calendar":{"service_id":"WKDY"}}]}`,
		},
		{
			Name:   "route",
			Query:  `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {route {route_id} }}`,
			Vars:   vars,
			Expect: `{"trips":[{"route":{"route_id":"01"}}]}`,
		},
		{
			Name:   "shape",
			Query:  `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {shape {shape_id} }}`,
			Vars:   vars,
			Expect: `{"trips":[{"shape":{"shape_id":"02_shp"}}]}`,
		},
		{
			Name:   "feed_version",
			Query:  `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {feed_version {sha1} }}`,
			Vars:   vars,
			Expect: `{"trips":[{"feed_version":{"sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0"}}]}`,
		},
		{
			Name:         "stop_times",
			Query:        `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {stop_times {stop_sequence} }}`,
			Vars:         vars,
			Selector:     "trips.0.stop_times.#.stop_sequence",
			SelectExpect: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16", "17", "18", "19", "20", "21", "22", "23", "24", "25", "26", "27"},
		},
		{
			Name:         "stop_times end time",
			Query:        `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {stop_times(where:{end:"05:45:00"}) {stop_sequence arrival_time departure_time} }}`,
			Vars:         vars,
			Selector:     "trips.0.stop_times.#.arrival_time",
			SelectExpect: []string{"05:26:00", "05:29:00", "05:33:00", "05:36:00", "05:40:00", "05:43:00"},
		},
		{
			Name:         "stop_times start time",
			Query:        `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {stop_times(where:{start:"06:00:00"}) {stop_sequence arrival_time departure_time} }}`,
			Vars:         vars,
			Selector:     "trips.0.stop_times.#.arrival_time",
			SelectExpect: []string{"06:05:00", "06:08:00", "06:11:00", "06:15:00", "06:17:00", "06:23:00", "06:27:00", "06:32:00", "06:35:00", "06:40:00", "06:43:00", "06:50:00", "07:05:00", "07:13:00"},
		},
		{
			Name:         "stop_times start and end time",
			Query:        `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {stop_times(where:{start:"06:00:00", end:"06:30:00"}) {stop_sequence arrival_time departure_time} }}`,
			Vars:         vars,
			Selector:     "trips.0.stop_times.#.arrival_time",
			SelectExpect: []string{"06:05:00", "06:08:00", "06:11:00", "06:15:00", "06:17:00", "06:23:00", "06:27:00"},
		},
		{
			Name:         "stop_times multiple 1",
			Query:        `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {st1: stop_times(limit:5, where:{start:"06:00:00"}) {stop_sequence arrival_time departure_time}, st2: stop_times(limit: 3, where:{start:"06:00:00"}) {stop_sequence arrival_time departure_time}}}`,
			Vars:         vars,
			Selector:     "trips.0.st1.#.departure_time",
			SelectExpect: []string{"06:05:00", "06:08:00", "06:11:00", "06:15:00", "06:17:00"},
		},
		{
			Name:         "stop_times multiple 2",
			Query:        `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {st1: stop_times(limit:5, where:{start:"06:00:00"}) {stop_sequence arrival_time departure_time}, st2: stop_times(limit: 3, where:{start:"06:00:00"}) {stop_sequence arrival_time departure_time}}}`,
			Vars:         vars,
			Selector:     "trips.0.st2.#.departure_time",
			SelectExpect: []string{"06:05:00", "06:08:00", "06:11:00"},
		},
		{
			Name:         "where trip_id",
			Query:        `query{  trips(where:{trip_id:"3850526WKDY"}) {trip_id}}`,
			Vars:         vars,
			Selector:     "trips.#.trip_id",
			SelectExpect: []string{"3850526WKDY"},
		},
		{
			Name:  "where service_date",
			Query: `query{trips(where:{feed_onestop_id:"CT",service_date:"2018-05-29"}){trip_id}}`,

			Selector:     "trips.#.trip_id",
			SelectExpect: []string{"101", "103", "305", "207", "309", "211", "313", "215", "217", "319", "221", "323", "225", "227", "329", "231", "233", "135", "237", "139", "143", "147", "151", "155", "257", "159", "261", "263", "365", "267", "269", "371", "273", "375", "277", "279", "381", "283", "385", "287", "289", "191", "193", "195", "197", "199", "102", "104", "206", "208", "310", "212", "314", "216", "218", "320", "222", "324", "226", "228", "330", "232", "134", "236", "138", "142", "146", "150", "152", "254", "156", "258", "360", "262", "264", "366", "268", "370", "272", "274", "376", "278", "380", "282", "284", "386", "288", "190", "192", "194", "196", "198"},
		},
		// service_dates and dates. Trip 101 runs Monday-Friday, trip 422
		// Saturday and Sunday; 2018-05-28 is a Monday.
		{
			Name:   "where service_dates",
			Query:  `query{trips(where:{trip_id:"101",service_dates:["2018-05-29","2018-05-30"]}){trip_id service_dates}}`,
			Expect: `{"trips":[{"service_dates":["2018-05-29","2018-05-30"],"trip_id":"101"}]}`,
		},
		{
			// service_dates match exactly: the preceding day is not brought in,
			// whether or not the trip runs on it.
			Name:   "where service_dates does not reach back",
			Query:  `query{trips(where:{trip_id:"101",service_dates:["2018-05-29"]}){service_dates}}`,
			Expect: `{"trips":[{"service_dates":["2018-05-29"]}]}`,
		},
		{
			// dates also match the preceding service date, whose after-midnight
			// departures fall on the requested calendar day.
			Name:   "where dates reaches back one service date",
			Query:  `query{trips(where:{trip_id:"101",dates:["2018-05-29"]}){service_dates}}`,
			Expect: `{"trips":[{"service_dates":["2018-05-28","2018-05-29"]}]}`,
		},
		{
			// A weekend trip is selected for Monday purely by that reach-back,
			// and reports only the Sunday it actually runs.
			Name:   "where dates selects a trip through the reach-back alone",
			Query:  `query{trips(where:{trip_id:"422",dates:["2018-05-28"]}){service_dates}}`,
			Expect: `{"trips":[{"service_dates":["2018-05-27"]}]}`,
		},
		{
			Name:   "where service_dates does not select that trip",
			Query:  `query{trips(where:{trip_id:"422",service_dates:["2018-05-28"]}){service_dates}}`,
			Expect: `{"trips":[]}`,
		},
		{
			Name:   "where dates takes precedence over service_dates",
			Query:  `query{trips(where:{trip_id:"101",dates:["2018-05-29"],service_dates:["2018-05-31"]}){service_dates}}`,
			Expect: `{"trips":[{"service_dates":["2018-05-28","2018-05-29"]}]}`,
		},
		// license
		{
			Name:         "license filter: share_alike_optional = yes",
			Query:        `query($lic:LicenseFilter) {trips(limit:1,where: {license: $lic}) {trip_id feed_version{feed{license{share_alike_optional}}}}}`,
			Vars:         hw{"lic": hw{"share_alike_optional": "YES"}},
			Selector:     "trips.0.feed_version.feed.license.share_alike_optional",
			SelectExpect: []string{"yes"},
		},
		{
			Name:         "license filter: share_alike_optional = no",
			Query:        `query($lic:LicenseFilter) {trips(limit:1,where: {license: $lic}) {trip_id feed_version{feed{license{share_alike_optional}}}}}`,
			Vars:         hw{"lic": hw{"share_alike_optional": "NO"}},
			Selector:     "trips.0.feed_version.feed.license.share_alike_optional",
			SelectExpect: []string{"no"},
		},
		{
			Name:         "license filter: create_derived_product = yes",
			Query:        `query($lic:LicenseFilter) {trips(limit:1,where: {license: $lic}) {trip_id feed_version{feed{license{create_derived_product}}}}}`,
			Vars:         hw{"lic": hw{"create_derived_product": "YES"}},
			Selector:     "trips.0.feed_version.feed.license.create_derived_product",
			SelectExpect: []string{"yes"},
		},
		{
			Name:         "license filter: create_derived_product = no",
			Query:        `query($lic:LicenseFilter) {trips(limit:1,where: {license: $lic}) {trip_id feed_version{feed{license{create_derived_product}}}}}`,
			Vars:         hw{"lic": hw{"create_derived_product": "NO"}},
			Selector:     "trips.0.feed_version.feed.license.create_derived_product",
			SelectExpect: []string{"no"},
		},
		{
			Name:         "license filter: commercial_use_allowed = yes",
			Query:        `query($lic:LicenseFilter) {trips(limit:1,where: {license: $lic}) {trip_id feed_version{feed{license{commercial_use_allowed}}}}}`,
			Vars:         hw{"lic": hw{"commercial_use_allowed": "YES"}},
			Selector:     "trips.0.feed_version.feed.license.commercial_use_allowed",
			SelectExpect: []string{"yes"},
		},
		{
			Name:         "license filter: commercial_use_allowed = no",
			Query:        `query($lic:LicenseFilter) {trips(limit:1,where: {license: $lic}) {trip_id feed_version{feed{license{commercial_use_allowed}}}}}`,
			Vars:         hw{"lic": hw{"commercial_use_allowed": "NO"}},
			Selector:     "trips.0.feed_version.feed.license.commercial_use_allowed",
			SelectExpect: []string{"no"},
		},
	}
	return testcases
}

func tripResolverFrequenciesTestcases(t testing.TB, cfg model.Config) []testcase {
	// CITY1 runs five headway bands; its first stop departs at 06:00:00.
	const q = `query($trip_id:String!) {trips(where:{feed_onestop_id:"EX", feed_version_sha1:"43e2278aa272879c79460582152b04e7487f0493", trip_id:$trip_id}) {frequencies{start_time end_time headway_secs trip{trip_id stop_times(limit:1){departure_time}}}}}`
	vars := hw{"trip_id": "CITY1"}
	testcases := []testcase{
		{
			Name:         "start_time",
			Query:        q,
			Vars:         vars,
			Selector:     "trips.0.frequencies.#.start_time",
			SelectExpect: []string{"06:00:00", "08:00:00", "10:00:00", "16:00:00", "19:00:00"},
		},
		{
			Name:         "end_time",
			Query:        q,
			Vars:         vars,
			Selector:     "trips.0.frequencies.#.end_time",
			SelectExpect: []string{"07:59:59", "09:59:59", "15:59:59", "18:59:59", "22:00:00"},
		},
		{
			Name:         "headway_secs",
			Query:        q,
			Vars:         vars,
			Selector:     "trips.0.frequencies.#.headway_secs",
			SelectExpect: []string{"1800", "600", "1800", "600", "1800"},
		},
		{
			Name:         "trip back reference",
			Query:        q,
			Vars:         vars,
			Selector:     "trips.0.frequencies.#.trip.trip_id",
			SelectExpect: []string{"CITY1", "CITY1", "CITY1", "CITY1", "CITY1"},
		},
		{
			// The anchor a caller expands from. It comes from the trip rather
			// than the band, so every band reports the same first departure.
			Name:         "first departure through the trip",
			Query:        q,
			Vars:         vars,
			Selector:     "trips.0.frequencies.#.trip.stop_times.0.departure_time",
			SelectExpect: []string{"06:00:00", "06:00:00", "06:00:00", "06:00:00", "06:00:00"},
		},
		{
			Name:         "trip with one frequency",
			Query:        q,
			Vars:         hw{"trip_id": "STBA"},
			Selector:     "trips.0.frequencies.#.headway_secs",
			SelectExpect: []string{"1800"},
		},
		{
			Name:         "trip with no frequencies",
			Query:        q,
			Vars:         hw{"trip_id": "AB1"},
			Selector:     "trips.0.frequencies.#.headway_secs",
			SelectExpect: []string{},
		},
	}
	return testcases
}

func tripResolverLicenseTestcases(t testing.TB, cfg model.Config) []testcase {
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
			Name:               "license filter: share_alike_optional = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"share_alike_optional": "YES"}},
			Selector:           "trips.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  31232,
		},
		{
			Name:               "license filter: share_alike_optional = no",
			Query:              q,
			Vars:               hw{"lic": hw{"share_alike_optional": "NO"}},
			Selector:           "trips.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  2525,
		},
		{
			Name:               "license filter: share_alike_optional = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"share_alike_optional": "EXCLUDE_NO"}},
			Selector:           "trips.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  32806,
		},
		// license: create_derived_product
		{
			Name:               "license filter: create_derived_product = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"create_derived_product": "YES"}},
			Selector:           "trips.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  31232,
		},
		{
			Name:               "license filter: create_derived_product = no",
			Query:              q,
			Vars:               hw{"lic": hw{"create_derived_product": "NO"}},
			Selector:           "trips.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  2525,
		},
		{
			Name:               "license filter: create_derived_product = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"create_derived_product": "EXCLUDE_NO"}},
			Selector:           "trips.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  32806,
		},
		// license: commercial_use_allowed
		{
			Name:               "license filter: commercial_use_allowed = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"commercial_use_allowed": "YES"}},
			Selector:           "trips.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  31232,
		},
		{
			Name:               "license filter: commercial_use_allowed = no",
			Query:              q,
			Vars:               hw{"lic": hw{"commercial_use_allowed": "NO"}},
			Selector:           "trips.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  2525,
		},
		{
			Name:               "license filter: commercial_use_allowed = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"commercial_use_allowed": "EXCLUDE_NO"}},
			Selector:           "trips.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  32806,
		},
		// license: redistribution_allowed
		{
			Name:               "license filter: redistribution_allowed = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"redistribution_allowed": "YES"}},
			Selector:           "trips.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  31232,
		},
		{
			Name:               "license filter: redistribution_allowed = no",
			Query:              q,
			Vars:               hw{"lic": hw{"redistribution_allowed": "NO"}},
			Selector:           "trips.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  2525,
		},
		{
			Name:               "license filter: redistribution_allowed = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"redistribution_allowed": "EXCLUDE_NO"}},
			Selector:           "trips.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  32806,
		},
		// license: use_without_attribution
		{
			Name:               "license filter: use_without_attribution = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"use_without_attribution": "YES"}},
			Selector:           "trips.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  31232,
		},
		{
			Name:               "license filter: use_without_attribution = no",
			Query:              q,
			Vars:               hw{"lic": hw{"use_without_attribution": "NO"}},
			Selector:           "trips.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  2525,
		},
		{
			Name:               "license filter: use_without_attribution = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"use_without_attribution": "EXCLUDE_NO"}},
			Selector:           "trips.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  32806,
		},
	}
	return testcases
}

func tripResolverFlexStopTimesTestcases(t testing.TB, cfg model.Config) []testcase {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	// A specific trip_id from the C-TRAN flex feed for deterministic testing
	ctranFlexTripID := "trip_id__ri-<2bc6804f-9e24-4b91-8947-c73a2363e7b6>_from-<db7489d3-7478-4d3b-a47f-60c58e3fed6e>_to-<db7489d3-7478-4d3b-a47f-60c58e3fed6e>_si-<MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a>"
	testcases := []testcase{
		{
			Name: "flex stop times via trip",
			Query: `query($sha1: String!, $trip_id: String!) {
				feed_versions(where:{sha1:$sha1}) {
					trips(where:{trip_id:$trip_id}) {
						trip_id
						flex_stop_times {
							stop_sequence
						}
					}
				}
			}`,
			Vars:         hw{"sha1": ctranFlexSha1, "trip_id": ctranFlexTripID},
			Selector:     "feed_versions.0.trips.0.flex_stop_times.#.stop_sequence",
			SelectExpect: []string{"1", "2"},
		},
		{
			Name: "flex stop times fields with pickup window",
			Query: `query($sha1: String!, $trip_id: String!) {
				feed_versions(where:{sha1:$sha1}) {
					trips(where:{trip_id:$trip_id}) {
						flex_stop_times {
							stop_sequence
							start_pickup_drop_off_window
							end_pickup_drop_off_window
							pickup_type
							drop_off_type
						}
					}
				}
			}`,
			Vars:         hw{"sha1": ctranFlexSha1, "trip_id": ctranFlexTripID},
			Selector:     "feed_versions.0.trips.0.flex_stop_times.#.start_pickup_drop_off_window",
			SelectExpect: []string{"05:30:00", "05:30:00"},
		},
		{
			Name: "flex stop times with location group",
			Query: `query($sha1: String!, $trip_id: String!) {
				feed_versions(where:{sha1:$sha1}) {
					trips(where:{trip_id:$trip_id}) {
						flex_stop_times {
							stop_sequence
							location_group {
								location_group_id
							}
						}
					}
				}
			}`,
			Vars:         hw{"sha1": ctranFlexSha1, "trip_id": ctranFlexTripID},
			Selector:     "feed_versions.0.trips.0.flex_stop_times.#.location_group.location_group_id",
			SelectExpect: []string{"location_group_id__db7489d3-7478-4d3b-a47f-60c58e3fed6e", "location_group_id__db7489d3-7478-4d3b-a47f-60c58e3fed6e"},
		},
		{
			Name: "flex stop times with booking rule",
			Query: `query($sha1: String!, $trip_id: String!) {
				feed_versions(where:{sha1:$sha1}) {
					trips(where:{trip_id:$trip_id}) {
						flex_stop_times {
							stop_sequence
							pickup_booking_rule {
								booking_rule_id
								booking_type
							}
						}
					}
				}
			}`,
			Vars:         hw{"sha1": ctranFlexSha1, "trip_id": ctranFlexTripID},
			Selector:     "feed_versions.0.trips.0.flex_stop_times.#.pickup_booking_rule.booking_rule_id",
			SelectExpect: []string{"booking_rule_id__2bc6804f-9e24-4b91-8947-c73a2363e7b6_MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a", "booking_rule_id__2bc6804f-9e24-4b91-8947-c73a2363e7b6_MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a"},
		},
	}
	return testcases
}

func tripResolverSafeDurationTestcases(t testing.TB, cfg model.Config) []testcase {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	ctranFlexTripID := "trip_id__ri-<2bc6804f-9e24-4b91-8947-c73a2363e7b6>_from-<db7489d3-7478-4d3b-a47f-60c58e3fed6e>_to-<db7489d3-7478-4d3b-a47f-60c58e3fed6e>_si-<MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a>"
	testcases := []testcase{
		{
			Name: "trip safe duration fields",
			Query: `query($sha1: String!, $trip_id: String!) {
				feed_versions(where:{sha1:$sha1}) {
					trips(where:{trip_id:$trip_id}) {
						trip_id
						safe_duration_factor
						safe_duration_offset
					}
				}
			}`,
			Vars:   hw{"sha1": ctranFlexSha1, "trip_id": ctranFlexTripID},
			Expect: `{"feed_versions":[{"trips":[{"trip_id":"trip_id__ri-<2bc6804f-9e24-4b91-8947-c73a2363e7b6>_from-<db7489d3-7478-4d3b-a47f-60c58e3fed6e>_to-<db7489d3-7478-4d3b-a47f-60c58e3fed6e>_si-<MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a>","safe_duration_factor":1,"safe_duration_offset":0}]}]}`,
		},
		{
			Name: "safe duration fields null for non-flex trip",
			Query: `query($trip_id: String!) {
				trips(where:{trip_id:$trip_id}) {
					trip_id
					safe_duration_factor
					safe_duration_offset
				}
			}`,
			Vars:   hw{"trip_id": "3850526WKDY"},
			Expect: `{"trips":[{"trip_id":"3850526WKDY","safe_duration_factor":null,"safe_duration_offset":null}]}`,
		},
	}
	return testcases
}

func feedVersionResolverFrequenciesTestcases(t testing.TB, cfg model.Config) []testcase {
	exSha1 := "43e2278aa272879c79460582152b04e7487f0493"
	q := `query($sha1:String!) {
		feed_versions(where:{sha1:$sha1}) {
			frequencies { start_time end_time headway_secs trip { trip_id } }
		}
	}`
	testcases := []testcase{
		{
			Name:  "every frequency in the feed version",
			Query: q,
			Vars:  hw{"sha1": exSha1},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.frequencies.#.headway_secs", ExpectCount: 11},
				{Selector: "feed_versions.0.frequencies.#.trip.trip_id", ExpectUnique: []string{"STBA", "CITY1", "CITY2"}},
				{Selector: `feed_versions.0.frequencies.#(trip.trip_id=="STBA")#.start_time`, Expect: []string{"06:00:00"}},
				{Selector: `feed_versions.0.frequencies.#(trip.trip_id=="STBA")#.end_time`, Expect: []string{"22:00:00"}},
			},
		},
		{
			Name:   "feed version fields",
			Query:  `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { frequencies(limit:1) { feed_version_sha1 feed_onestop_id feed_version { sha1 } } } }`,
			Vars:   hw{"sha1": exSha1},
			Expect: `{"feed_versions":[{"frequencies":[{"feed_version_sha1":"43e2278aa272879c79460582152b04e7487f0493","feed_onestop_id":"EX","feed_version":{"sha1":"43e2278aa272879c79460582152b04e7487f0493"}}]}]}`,
		},
		{
			// Trip.frequencies shares the feed version select.
			Name:  "feed version fields through a trip",
			Query: `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { trips(where:{trip_id:"CITY1"}) { frequencies { feed_version_sha1 feed_onestop_id } } } }`,
			Vars:  hw{"sha1": exSha1},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.trips.0.frequencies.#.feed_version_sha1", ExpectUnique: []string{exSha1}},
				{Selector: "feed_versions.0.trips.0.frequencies.#.feed_onestop_id", ExpectUnique: []string{"EX"}},
			},
		},
		{
			Name:   "feed version without frequencies",
			Query:  q,
			Vars:   hw{"sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"},
			Expect: `{"feed_versions":[{"frequencies":[]}]}`,
		},
	}
	return testcases
}

func tripResolverStopPatternIDTest(t *testing.T, c *client.Client, cfg model.Config) {
	query := `query {
		trips(where: {feed_onestop_id: "BA", trip_id:"3230742WKDY"}) {
		  trip_id
		  stop_pattern_id
		}
	}`
	var resp map[string]interface{}
	if err := c.Post(query, &resp); err != nil {
		t.Error(err)
		return
	}
	jj := gqlcase.ToJSON(resp)
	patId := gjson.Get(jj, "trips.0.stop_pattern_id").Int()
	tc := testcase{
		Name: "where trip_id",
		Query: `query($patid:Int!) {
			trips(where: {feed_onestop_id: "BA", stop_pattern_id:$patid}) {
			  trip_id
			  stop_pattern_id
			}
		  }
		`,
		Vars:         hw{"patid": patId},
		Expect:       ``,
		Selector:     "trips.#.trip_id",
		SelectExpect: []string{"3230742WKDY", "3250757WKDY", "3270812WKDY", "3310827WKDY", "3210842WKDY"},
	}
	t.Run(tc.Name, func(t *testing.T) {
		gqlcase.QueryTestcase(t, c, tc)
	})
}
