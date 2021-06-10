package resolvers

import (
	"testing"

	"github.com/99designs/gqlgen/client"
)

func TestTripResolver(t *testing.T) {
	vars := hw{"trip_id": "3850526WKDY"}
	testcases := []testcase{
		{
			"basic fields",
			`query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {trip_id trip_headsign trip_short_name direction_id block_id wheelchair_accessible bikes_allowed stop_pattern_id }}`,
			vars,
			`{"trips":[{"bikes_allowed":1,"block_id":"","direction_id":1,"stop_pattern_id":21,"trip_headsign":"Antioch","trip_id":"3850526WKDY","trip_short_name":"","wheelchair_accessible":1}]}`,
			"",
			nil,
		},
		{
			"calendar",
			`query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {calendar {service_id} }}`,
			vars,
			`{"trips":[{"calendar":{"service_id":"WKDY"}}]}`,
			"",
			nil,
		},
		{
			"route",
			`query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {route {route_id} }}`,
			vars,
			`{"trips":[{"route":{"route_id":"01"}}]}`,
			"",
			nil,
		},
		{
			"shape",
			`query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {shape {shape_id} }}`,
			vars,
			`{"trips":[{"shape":{"shape_id":"02_shp"}}]}`,
			"",
			nil,
		},
		{
			"feed_version",
			`query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {feed_version {sha1} }}`,
			vars,
			`{"trips":[{"feed_version":{"sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0"}}]}`,
			"",
			nil,
		},
		{
			"stop_times",
			`query($trip_id: String!) {  trips(where:{trip_id:$trip_id}) {stop_times {stop_sequence} }}`,
			vars,
			``,
			"trips.0.stop_times.#.stop_sequence",
			[]string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16", "17", "18", "19", "20", "21", "22", "23", "24", "25", "26", "27"},
		},
		{
			"where trip_id",
			`query{  trips(where:{trip_id:"3850526WKDY"}) {trip_id}}`,
			vars,
			``,
			"trips.#.trip_id",
			[]string{"3850526WKDY"},
		},
		{
			"where service_date",
			`query{trips(where:{feed_onestop_id:"CT",service_date:"2018-05-29"}){trip_id}}`,
			hw{},
			``,
			"trips.#.trip_id",
			[]string{"101", "103", "305", "207", "309", "211", "313", "215", "217", "319", "221", "323", "225", "227", "329", "231", "233", "135", "237", "139", "143", "147", "151", "155", "257", "159", "261", "263", "365", "267", "269", "371", "273", "375", "277", "279", "381", "283", "385", "287", "289", "191", "193", "195", "197", "199", "102", "104", "206", "208", "310", "212", "314", "216", "218", "320", "222", "324", "226", "228", "330", "232", "134", "236", "138", "142", "146", "150", "152", "254", "156", "258", "360", "262", "264", "366", "268", "370", "272", "274", "376", "278", "380", "282", "284", "386", "288", "190", "192", "194", "196", "198"},
		},
		// TODO: check where feed_version_sha1, feed_onestop_id but only check count
		// TODO: frequencies
	}
	c := client.New(NewServer())
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testquery(t, c, tc)
		})
	}
}
