package gql

import (
	"testing"
)

func TestBookingRuleResolver(t *testing.T) {
	brid := "booking_rule_id__2bc6804f-9e24-4b91-8947-c73a2363e7b6_MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a"
	testcases := []testcase{
		{
			name: "booking rules - returns multiple booking rules",
			query: `query {
				feed_versions(where: {sha1: "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"}) {
					booking_rules {
						booking_rule_id
					}
				}
			}`,
			selector:          "feed_versions.0.booking_rules.#.booking_rule_id",
			selectExpectCount: 100, // Just verify we get many results (exact count may vary)
		},
		{
			name: "filters booking rules by booking_rule_id",
			query: `query($brid: String) {
				feed_versions(where: {sha1: "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"}) {
					booking_rules(where:{booking_rule_id: $brid}) {
						booking_rule_id
						booking_type
						prior_notice_duration_min
						message
						phone_number
						info_url
						feed_version {
							sha1
						}
					}
				}
			}`,
			vars: hw{
				"brid": brid,
			},
			expect: `{"feed_versions":[{"booking_rules":[{"booking_rule_id":"booking_rule_id__2bc6804f-9e24-4b91-8947-c73a2363e7b6_MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a","booking_type":1,"feed_version":{"sha1":"e8bc76c3c8602cad745f41a49ed5c5627ad6904c"},"info_url":null,"message":"The Current is an on-demand rideshare service by C-TRAN that provides point-to-point service for just the cost of a local bus ride. Schedule your ride on The Current app, at www.ridethecurrent.com or through our mobile app, or by calling 360-695-0123 then track your driver’s arrival.","phone_number":"360-695-0123","prior_notice_duration_min":0}]}]}`,
		},
	}
	c, _ := newTestClient(t)
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			queryTestcase(t, c, tc)
		})
	}
}
