package gqltest

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/model"
)

func feedVersionResolverTransfersTestcases(t testing.TB, cfg model.Config) []testcase {
	supplementalSha1 := testconfig.SupplementalSha1(t, cfg)
	baSha1 := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
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
			Name:  "every transfer in the feed version",
			Query: q,
			Vars:  hw{"sha1": baSha1},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.transfers.#.from_stop.stop_id", Expect: []string{"19TH_N", "MCAR_S", "BALB", "BAYF", "SBRN", "COLS", "FRMT", "WCRK"}},
				{Selector: "feed_versions.0.transfers.#.feed_version_sha1", ExpectUnique: []string{baSha1}},
				{Selector: "feed_versions.0.transfers.#.feed_onestop_id", ExpectUnique: []string{"BA"}},
				{Selector: "feed_versions.0.transfers.#.feed_version.sha1", ExpectUnique: []string{baSha1}},
			},
		},
		{
			Name:   "transfer_type",
			Query:  q,
			Vars:   hw{"sha1": baSha1, "where": hw{"transfer_type": 2}},
			Expect: `{"feed_versions":[{"transfers":[{"transfer_type":2,"min_transfer_time":240,"from_stop":{"stop_id":"COLS"},"to_stop":{"stop_id":"COLS"},"from_route":null,"to_route":null,"from_trip":null,"to_trip":null,"feed_version_sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0","feed_onestop_id":"BA","feed_version":{"sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0"}}]}]}`,
		},
		{
			Name:         "timed transfers",
			Query:        q,
			Vars:         hw{"sha1": baSha1, "where": hw{"transfer_type": 1}},
			Selector:     "feed_versions.0.transfers.#.to_stop.stop_id",
			SelectExpect: []string{"19TH_N", "MCAR_S", "SBRN", "FRMT", "WCRK"},
		},
		{
			// The supplemental feed's transfers reference routes and trips.
			Name:  "route and trip references",
			Query: q,
			Vars:  hw{"sha1": supplementalSha1},
			Sel: []testcaseSelector{
				{Selector: `feed_versions.0.transfers.#(transfer_type==1)#.from_route.route_id`, Expect: []string{"L"}},
				{Selector: `feed_versions.0.transfers.#(transfer_type==1)#.to_route.route_id`, Expect: []string{"X"}},
				{Selector: `feed_versions.0.transfers.#(transfer_type==1)#.from_stop.stop_id`, Expect: []string{"N1"}},
				{Selector: `feed_versions.0.transfers.#(transfer_type==4)#.from_trip.trip_id`, Expect: []string{"T101"}},
				{Selector: `feed_versions.0.transfers.#(transfer_type==4)#.to_trip.trip_id`, Expect: []string{"T102"}},
				{Selector: `feed_versions.0.transfers.#(transfer_type==4)#.from_stop`, Expect: []string{""}},
			},
		},
	}
	return testcases
}
