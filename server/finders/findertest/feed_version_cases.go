package findertest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// Cases duplicated from server/gql/feed_version_resolver_test.go.

func feedVersionResolverLocationsTestcases() []testcase {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	testcases := []testcase{
		{
			name:  "locations",
			query: `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { locations { location_id } }}`,
			vars:  hw{"sha1": ctranFlexSha1},
			f: func(t *testing.T, jj string) {
				locs := gjson.Get(jj, "feed_versions.0.locations").Array()
				assert.Greater(t, len(locs), 0, "expected at least one location")
			},
		},
		{
			name:   "location filter by id",
			query:  `query($sha1: String!, $location_id: String) { feed_versions(where:{sha1:$sha1}) { locations(where:{location_id:$location_id}) { location_id stop_name } }}`,
			vars:   hw{"sha1": ctranFlexSha1, "location_id": "location_id__c7400cc8-959c-42c8-991f-8f601ec9ea59"},
			expect: `{"feed_versions":[{"locations":[{"location_id":"location_id__c7400cc8-959c-42c8-991f-8f601ec9ea59","stop_name":"Rose Village"}]}]}`,
		},
	}
	return testcases
}

func feedVersionResolverBookingRulesTestcases() []testcase {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	testcases := []testcase{
		{
			name:  "booking rules",
			query: `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { booking_rules { booking_rule_id booking_type prior_notice_start_day } }}`,
			vars:  hw{"sha1": ctranFlexSha1},
			f: func(t *testing.T, jj string) {
				brs := gjson.Get(jj, "feed_versions.0.booking_rules").Array()
				assert.Greater(t, len(brs), 0, "expected at least one booking rule")
			},
		},
		{
			name:   "booking rule filter by id",
			query:  `query($sha1: String!, $booking_rule_id: String) { feed_versions(where:{sha1:$sha1}) { booking_rules(where:{booking_rule_id:$booking_rule_id}) { booking_rule_id booking_type } }}`,
			vars:   hw{"sha1": ctranFlexSha1, "booking_rule_id": "booking_rule_id__2bc6804f-9e24-4b91-8947-c73a2363e7b6_MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a"},
			expect: `{"feed_versions":[{"booking_rules":[{"booking_rule_id":"booking_rule_id__2bc6804f-9e24-4b91-8947-c73a2363e7b6_MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a","booking_type":1}]}]}`,
		},
	}
	return testcases
}

func feedVersionResolverLocationGroupsTestcases() []testcase {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	testcases := []testcase{
		{
			name:  "location groups",
			query: `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { location_groups { location_group_id location_group_name } }}`,
			vars:  hw{"sha1": ctranFlexSha1},
			f: func(t *testing.T, jj string) {
				lgs := gjson.Get(jj, "feed_versions.0.location_groups").Array()
				assert.Greater(t, len(lgs), 0, "expected at least one location group")
			},
		},
		{
			name:  "location groups with limit",
			query: `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { location_groups(limit:2) { location_group_id } }}`,
			vars:  hw{"sha1": ctranFlexSha1},
			f: func(t *testing.T, jj string) {
				lgs := gjson.Get(jj, "feed_versions.0.location_groups").Array()
				assert.LessOrEqual(t, len(lgs), 2, "expected at most 2 location groups with limit:2")
			},
		},
		{
			name:   "location group filter by id",
			query:  `query($sha1: String!, $location_group_id: String) { feed_versions(where:{sha1:$sha1}) { location_groups(where:{location_group_id:$location_group_id}) { location_group_id location_group_name } }}`,
			vars:   hw{"sha1": ctranFlexSha1, "location_group_id": "location_group_id__db7489d3-7478-4d3b-a47f-60c58e3fed6e"},
			expect: `{"feed_versions":[{"location_groups":[{"location_group_id":"location_group_id__db7489d3-7478-4d3b-a47f-60c58e3fed6e","location_group_name":"VA Medical Center (fixed route stop)"}]}]}`,
		},
	}
	return testcases
}
