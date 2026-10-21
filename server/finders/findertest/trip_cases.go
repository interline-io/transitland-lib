package findertest

// Cases duplicated from server/gql/trip_resolver_test.go.

func tripResolverTestcases() []testcase {
	vars := hw{"trip_id": "3850526WKDY"}
	testcases := []testcase{
		{
			name:   "basic fields",
			query:  `query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {trip_id trip_headsign trip_short_name direction_id block_id wheelchair_accessible bikes_allowed cars_allowed stop_pattern_id }}`,
			vars:   vars,
			expect: `{"trips":[{"bikes_allowed":1,"block_id":null,"cars_allowed":null,"direction_id":1,"stop_pattern_id":21,"trip_headsign":"Antioch","trip_id":"3850526WKDY","trip_short_name":null,"wheelchair_accessible":1}]}`,
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
		// service_dates and dates. Trip 101 runs Monday-Friday, trip 422
		// Saturday and Sunday; 2018-05-28 is a Monday.
		{
			name:   "where service_dates",
			query:  `query{trips(where:{trip_id:"101",service_dates:["2018-05-29","2018-05-30"]}){trip_id service_dates}}`,
			expect: `{"trips":[{"service_dates":["2018-05-29","2018-05-30"],"trip_id":"101"}]}`,
		},
		{
			// service_dates match exactly: the preceding day is not brought in,
			// whether or not the trip runs on it.
			name:   "where service_dates does not reach back",
			query:  `query{trips(where:{trip_id:"101",service_dates:["2018-05-29"]}){service_dates}}`,
			expect: `{"trips":[{"service_dates":["2018-05-29"]}]}`,
		},
		{
			// dates also match the preceding service date, whose after-midnight
			// departures fall on the requested calendar day.
			name:   "where dates reaches back one service date",
			query:  `query{trips(where:{trip_id:"101",dates:["2018-05-29"]}){service_dates}}`,
			expect: `{"trips":[{"service_dates":["2018-05-28","2018-05-29"]}]}`,
		},
		{
			// A weekend trip is selected for Monday purely by that reach-back,
			// and reports only the Sunday it actually runs.
			name:   "where dates selects a trip through the reach-back alone",
			query:  `query{trips(where:{trip_id:"422",dates:["2018-05-28"]}){service_dates}}`,
			expect: `{"trips":[{"service_dates":["2018-05-27"]}]}`,
		},
		{
			name:   "where service_dates does not select that trip",
			query:  `query{trips(where:{trip_id:"422",service_dates:["2018-05-28"]}){service_dates}}`,
			expect: `{"trips":[]}`,
		},
		{
			name:   "where dates takes precedence over service_dates",
			query:  `query{trips(where:{trip_id:"101",dates:["2018-05-29"],service_dates:["2018-05-31"]}){service_dates}}`,
			expect: `{"trips":[{"service_dates":["2018-05-28","2018-05-29"]}]}`,
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
	}
	return testcases
}

func tripFrequenciesTestcases() []testcase {
	// CITY1 runs five headway bands; its first stop departs at 06:00:00.
	const q = `query($trip_id:String!) {trips(where:{feed_onestop_id:"EX", feed_version_sha1:"43e2278aa272879c79460582152b04e7487f0493", trip_id:$trip_id}) {frequencies{start_time end_time headway_secs trip{trip_id stop_times(limit:1){departure_time}}}}}`
	vars := hw{"trip_id": "CITY1"}
	testcases := []testcase{
		{
			name:         "start_time",
			query:        q,
			vars:         vars,
			selector:     "trips.0.frequencies.#.start_time",
			selectExpect: []string{"06:00:00", "08:00:00", "10:00:00", "16:00:00", "19:00:00"},
		},
		{
			name:         "end_time",
			query:        q,
			vars:         vars,
			selector:     "trips.0.frequencies.#.end_time",
			selectExpect: []string{"07:59:59", "09:59:59", "15:59:59", "18:59:59", "22:00:00"},
		},
		{
			name:         "headway_secs",
			query:        q,
			vars:         vars,
			selector:     "trips.0.frequencies.#.headway_secs",
			selectExpect: []string{"1800", "600", "1800", "600", "1800"},
		},
		{
			name:         "trip back reference",
			query:        q,
			vars:         vars,
			selector:     "trips.0.frequencies.#.trip.trip_id",
			selectExpect: []string{"CITY1", "CITY1", "CITY1", "CITY1", "CITY1"},
		},
		{
			// The anchor a caller expands from. It comes from the trip rather
			// than the band, so every band reports the same first departure.
			name:         "first departure through the trip",
			query:        q,
			vars:         vars,
			selector:     "trips.0.frequencies.#.trip.stop_times.0.departure_time",
			selectExpect: []string{"06:00:00", "06:00:00", "06:00:00", "06:00:00", "06:00:00"},
		},
		{
			name:         "trip with one frequency",
			query:        q,
			vars:         hw{"trip_id": "STBA"},
			selector:     "trips.0.frequencies.#.headway_secs",
			selectExpect: []string{"1800"},
		},
		{
			name:         "trip with no frequencies",
			query:        q,
			vars:         hw{"trip_id": "AB1"},
			selector:     "trips.0.frequencies.#.headway_secs",
			selectExpect: []string{},
		},
	}
	return testcases
}
