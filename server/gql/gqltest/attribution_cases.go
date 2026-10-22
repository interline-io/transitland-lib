package gqltest

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
)

func feedVersionResolverAttributionsTestcases(t testing.TB, cfg model.Config) []testcase {
	supplementalSha1 := findSupplementalSha1(t, cfg)
	// The supplemental feed has attributions: one for the whole feed, and one each
	// for an agency, a route and a trip.
	q := `query($sha1:String!, $where:AttributionFilter) {
		feed_versions(where:{sha1:$sha1}) {
			attributions(where:$where) {
				attribution_id organization_name is_producer is_operator is_authority
				attribution_url attribution_email attribution_phone
				agency { agency_id } route { route_id } trip { trip_id }
			}
		}
	}`
	testcases := []testcase{
		{
			name:  "every attribution in the feed version",
			query: `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { attributions { attribution_id feed_version_sha1 feed_onestop_id feed_version { sha1 } } } }`,
			vars:  hw{"sha1": supplementalSha1},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.attributions.#.attribution_id", expect: []string{"producer", "operator", "authority", "trip_operator"}},
				{selector: "feed_versions.0.attributions.#.feed_version_sha1", expectUnique: []string{supplementalSha1}},
				{selector: "feed_versions.0.attributions.#.feed_onestop_id", expectUnique: []string{"supplemental"}},
				{selector: "feed_versions.0.attributions.#.feed_version.sha1", expectUnique: []string{supplementalSha1}},
			},
		},
		{
			name:   "feed-wide attribution",
			query:  q,
			vars:   hw{"sha1": supplementalSha1, "where": hw{"attribution_id": "producer"}},
			expect: `{"feed_versions":[{"attributions":[{"attribution_id":"producer","organization_name":"Example Data Producer","is_producer":1,"is_operator":0,"is_authority":0,"attribution_url":"https://example.com/gtfs","attribution_email":"gtfs@example.com","attribution_phone":null,"agency":null,"route":null,"trip":null}]}]}`,
		},
		{
			name:   "agency attribution",
			query:  q,
			vars:   hw{"sha1": supplementalSha1, "where": hw{"attribution_id": "operator"}},
			expect: `{"feed_versions":[{"attributions":[{"attribution_id":"operator","organization_name":"Supplemental Transit","is_producer":0,"is_operator":1,"is_authority":0,"attribution_url":null,"attribution_email":null,"attribution_phone":"555-0100","agency":{"agency_id":"supplemental"},"route":null,"trip":null}]}]}`,
		},
		{
			name:         "route attribution",
			query:        q,
			vars:         hw{"sha1": supplementalSha1, "where": hw{"attribution_id": "authority"}},
			selector:     "feed_versions.0.attributions.0.route.route_id",
			selectExpect: []string{"X"},
		},
		{
			name:         "trip attribution",
			query:        q,
			vars:         hw{"sha1": supplementalSha1, "where": hw{"attribution_id": "trip_operator"}},
			selector:     "feed_versions.0.attributions.0.trip.trip_id",
			selectExpect: []string{"T101"},
		},
		{
			name:   "feed version without attributions",
			query:  q,
			vars:   hw{"sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"},
			expect: `{"feed_versions":[{"attributions":[]}]}`,
		},
	}
	return testcases
}
