package gqltest

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
)

func bookingRuleResolverTestcases(t testing.TB, cfg model.Config) []testcase {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	brid := "booking_rule_id__2bc6804f-9e24-4b91-8947-c73a2363e7b6_MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a"
	testcases := []testcase{
		{
			Name: "booking rules - returns multiple booking rules",
			Query: `query($sha1: String!) {
				feed_versions(where: {sha1: $sha1}) {
					booking_rules(limit: 1000) {
						booking_rule_id
					}
				}
			}`,
			Vars:              hw{"sha1": ctranFlexSha1},
			Selector:          "feed_versions.0.booking_rules.#.booking_rule_id",
			SelectExpectCount: 128,
		},
		// Test empty result for non-existent booking_rule_id
		{
			Name: "booking rule filter not found",
			Query: `query($sha1: String!) {
				feed_versions(where: {sha1: $sha1}) {
					booking_rules(where:{booking_rule_id: "nonexistent"}) {
						booking_rule_id
					}
				}
			}`,
			Vars:         hw{"sha1": ctranFlexSha1},
			Selector:     "feed_versions.0.booking_rules.#.booking_rule_id",
			SelectExpect: []string{},
		},
		// Test all booking rule fields including nullable ones
		{
			Name: "booking rule all fields",
			Query: `query($sha1: String!, $brid: String!) {
				feed_versions(where: {sha1: $sha1}) {
					booking_rules(where:{booking_rule_id: $brid}) {
						booking_rule_id
						booking_type
						prior_notice_duration_min
						prior_notice_duration_max
						prior_notice_last_day
						prior_notice_last_time
						prior_notice_start_day
						prior_notice_start_time
						message
						pickup_message
						drop_off_message
						phone_number
						info_url
						booking_url
					}
				}
			}`,
			Vars:   hw{"sha1": ctranFlexSha1, "brid": brid},
			Expect: `{"feed_versions":[{"booking_rules":[{"booking_rule_id":"booking_rule_id__2bc6804f-9e24-4b91-8947-c73a2363e7b6_MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a","booking_type":1,"booking_url":"https://book.ridethecurrent.com","drop_off_message":null,"info_url":null,"message":"The Current is an on-demand rideshare service by C-TRAN that provides point-to-point service for just the cost of a local bus ride. Schedule your ride on The Current app, at www.ridethecurrent.com or through our mobile app, or by calling 360-695-0123 then track your driver’s arrival.","phone_number":"360-695-0123","pickup_message":null,"prior_notice_duration_max":null,"prior_notice_duration_min":0,"prior_notice_last_day":null,"prior_notice_last_time":null,"prior_notice_start_day":2,"prior_notice_start_time":"00:00:00"}]}]}`,
		},
	}
	return testcases
}

func bookingRuleResolverFeedVersionTestcases(t testing.TB, cfg model.Config) []testcase {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	brid := "booking_rule_id__2bc6804f-9e24-4b91-8947-c73a2363e7b6_MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a"
	testcases := []testcase{
		{
			Name: "booking rule feed metadata",
			Query: `query($sha1: String!, $brid: String!) {
				feed_versions(where: {sha1: $sha1}) {
					booking_rules(where:{booking_rule_id: $brid}) {
						booking_rule_id
						feed_onestop_id
						feed_version_sha1
						feed_version {
							sha1
						}
					}
				}
			}`,
			Vars: hw{"sha1": ctranFlexSha1, "brid": brid},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.booking_rules.0.booking_rule_id", ExpectUnique: []string{brid}},
				{Selector: "feed_versions.0.booking_rules.0.feed_onestop_id", ExpectUnique: []string{"ctran-flex"}},
				{Selector: "feed_versions.0.booking_rules.0.feed_version_sha1", ExpectUnique: []string{ctranFlexSha1}},
				{Selector: "feed_versions.0.booking_rules.0.feed_version.sha1", ExpectUnique: []string{ctranFlexSha1}},
			},
		},
	}
	return testcases
}

func bookingRuleResolverPriorNoticeServiceTestcases(t testing.TB, cfg model.Config) []testcase {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	brid := "booking_rule_id__2bc6804f-9e24-4b91-8947-c73a2363e7b6_MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a"
	testcases := []testcase{
		// Test prior_notice_service relationship (nullable Calendar)
		// Note: The test data does not have any booking rules with prior_notice_service_id set,
		// so we verify the resolver returns null correctly
		{
			Name: "booking rule prior_notice_service null",
			Query: `query($sha1: String!, $brid: String!) {
				feed_versions(where: {sha1: $sha1}) {
					booking_rules(where:{booking_rule_id: $brid}) {
						booking_rule_id
						prior_notice_service {
							service_id
						}
					}
				}
			}`,
			Vars: hw{"sha1": ctranFlexSha1, "brid": brid},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.booking_rules.0.booking_rule_id", ExpectUnique: []string{brid}},
				// prior_notice_service should be null for this booking rule, so service_id selector returns empty
				{Selector: "feed_versions.0.booking_rules.0.prior_notice_service.service_id", Expect: []string{}},
			},
		},
	}
	return testcases
}
