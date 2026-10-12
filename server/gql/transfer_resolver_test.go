package gql

import (
	"testing"
)

func TestFeedVersionResolver_Transfers(t *testing.T) {
	baSha1 := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
	ctSha1 := "d2813c293bcfd7a97dde599527ae6c62c98e66c6"
	q := `query($sha1:String!, $where:TransferFilter) {
		feed_versions(where:{sha1:$sha1}) {
			transfers(where:$where) {
				transfer_type min_transfer_time
				from_stop { stop_id } to_stop { stop_id }
				from_route { route_id } to_route { route_id }
				from_trip { trip_id } to_trip { trip_id }
				feed_version_sha1 feed_onestop_id feed_version { sha1 }
			}
		}
	}`
	testcases := []testcase{
		{
			name:  "every transfer in the feed version",
			query: q,
			vars:  hw{"sha1": baSha1},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.transfers.#.from_stop.stop_id", expect: []string{"19TH_N", "MCAR_S", "BALB", "BAYF", "SBRN", "COLS", "FRMT", "WCRK"}},
				{selector: "feed_versions.0.transfers.#.feed_version_sha1", expectUnique: []string{baSha1}},
				{selector: "feed_versions.0.transfers.#.feed_onestop_id", expectUnique: []string{"BA"}},
				{selector: "feed_versions.0.transfers.#.feed_version.sha1", expectUnique: []string{baSha1}},
			},
		},
		{
			name:   "transfer_type",
			query:  q,
			vars:   hw{"sha1": baSha1, "where": hw{"transfer_type": 2}},
			expect: `{"feed_versions":[{"transfers":[{"transfer_type":2,"min_transfer_time":240,"from_stop":{"stop_id":"COLS"},"to_stop":{"stop_id":"COLS"},"from_route":null,"to_route":null,"from_trip":null,"to_trip":null,"feed_version_sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0","feed_onestop_id":"BA","feed_version":{"sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0"}}]}]}`,
		},
		{
			name:         "timed transfers",
			query:        q,
			vars:         hw{"sha1": baSha1, "where": hw{"transfer_type": 1}},
			selector:     "feed_versions.0.transfers.#.to_stop.stop_id",
			selectExpect: []string{"19TH_N", "MCAR_S", "SBRN", "FRMT", "WCRK"},
		},
		{
			// The supplemental CT transfers reference routes and trips.
			name:  "route and trip references",
			query: q,
			vars:  hw{"sha1": ctSha1},
			sel: []testcaseSelector{
				{selector: `feed_versions.0.transfers.#(transfer_type==1)#.from_route.route_id`, expect: []string{"Lo-130"}},
				{selector: `feed_versions.0.transfers.#(transfer_type==1)#.to_route.route_id`, expect: []string{"Bu-130"}},
				{selector: `feed_versions.0.transfers.#(transfer_type==1)#.from_stop.stop_id`, expect: []string{"70011"}},
				{selector: `feed_versions.0.transfers.#(transfer_type==4)#.from_trip.trip_id`, expect: []string{"101"}},
				{selector: `feed_versions.0.transfers.#(transfer_type==4)#.to_trip.trip_id`, expect: []string{"102"}},
				{selector: `feed_versions.0.transfers.#(transfer_type==4)#.from_stop`, expect: []string{""}},
			},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}
