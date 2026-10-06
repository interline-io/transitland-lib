package gql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// Fares v1 data comes from the CT and BA test feeds. Fares v2 data comes from
// the ctran-flex feed (fare_media, fare_products, fare_leg_rules,
// fare_transfer_rules, rider_categories) and from synthetic CT records in
// testdata/server/test_supplement.pgsql (networks, route_networks, areas,
// stop_areas, timeframes, fare_leg_join_rules, and the rules that use them).
func TestFareResolver(t *testing.T) {
	ctSha1 := "d2813c293bcfd7a97dde599527ae6c62c98e66c6"
	baSha1 := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	testcases := []testcase{
		// Fares v1
		{
			name:         "fare_attributes",
			query:        `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_attributes { fare_id price currency_type payment_method transfers transfer_duration agency { agency_id } feed_version_sha1 feed_onestop_id feed_version { sha1 } } } }`,
			vars:         hw{"sha1": ctSha1},
			selector:     "feed_versions.0.fare_attributes.#.fare_id",
			selectExpect: []string{"OW_1_20160228", "OW_2_20160228", "OW_3_20160228", "OW_4_20160228", "OW_5_20160228", "OW_6_20160228"},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.fare_attributes.#.price", expect: []string{"3.75", "6", "8.25", "10.5", "12.75", "15"}},
				{selector: "feed_versions.0.fare_attributes.#.currency_type", expectUnique: []string{"USD"}},
				{selector: "feed_versions.0.fare_attributes.#.payment_method", expectUnique: []string{"1"}},
				{selector: "feed_versions.0.fare_attributes.#.transfer_duration", expectUnique: []string{"14400"}},
				{selector: "feed_versions.0.fare_attributes.#.feed_version.sha1", expectUnique: []string{ctSha1}},
				{selector: "feed_versions.0.fare_attributes.#.feed_version_sha1", expectUnique: []string{ctSha1}},
				{selector: "feed_versions.0.fare_attributes.#.feed_onestop_id", expectUnique: []string{"CT"}},
			},
		},
		{
			name:     "fare_attributes transfers and agency are null when not set",
			query:    `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_attributes(limit: 1) { transfers agency { agency_id } } } }`,
			vars:     hw{"sha1": ctSha1},
			expect:   `{"feed_versions":[{"fare_attributes":[{"agency":null,"transfers":null}]}]}`,
			selector: "feed_versions.0.fare_attributes.#.transfers",
		},
		{
			name:  "fare_rules",
			query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_rules(limit: 3) { fare_attribute { fare_id } route { route_id } origin_id destination_id contains_id } } }`,
			vars:  hw{"sha1": ctSha1},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.fare_rules.#.fare_attribute.fare_id", expect: []string{"OW_1_20160228", "OW_2_20160228", "OW_3_20160228"}},
				{selector: "feed_versions.0.fare_rules.#.route.route_id", expectUnique: []string{"Bu-130"}},
				{selector: "feed_versions.0.fare_rules.#.origin_id", expectUnique: []string{"1"}},
				{selector: "feed_versions.0.fare_rules.#.destination_id", expect: []string{"1", "2", "3"}},
			},
		},
		{
			name:              "fare_rules default limit",
			query:             `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_rules { id } } }`,
			vars:              hw{"sha1": baSha1},
			selector:          "feed_versions.0.fare_rules.#.id",
			selectExpectCount: 100,
		},
		{
			name:              "fare_rules above the default limit",
			query:             `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_rules(limit: 10000) { id } } }`,
			vars:              hw{"sha1": baSha1},
			selector:          "feed_versions.0.fare_rules.#.id",
			selectExpectCount: 2304,
		},
		// Fares v2
		{
			name:  "fare_media",
			query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_media { fare_media_id fare_media_name fare_media_type } } }`,
			vars:  hw{"sha1": ctranFlexSha1},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.fare_media.#.fare_media_id", expect: []string{"0", "1", "2", "3", "4"}},
				{selector: "feed_versions.0.fare_media.#.fare_media_name", expect: []string{"Cash", "Ticket", "HOP Fastpass", "Open Payment", "Virtual HOP Fastpass"}},
				{selector: "feed_versions.0.fare_media.#.fare_media_type", expect: []string{"0", "1", "2", "3", "4"}},
			},
		},
		{
			name:  "fare_products",
			query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_products(limit: 2) { fare_product_id fare_product_name rider_category_id fare_media { fare_media_name } amount currency } } }`,
			vars:  hw{"sha1": ctranFlexSha1},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.fare_products.#.fare_product_id", expectUnique: []string{"ADULT_LOCAL_SINGLE_RIDE"}},
				{selector: "feed_versions.0.fare_products.#.rider_category_id", expectUnique: []string{"ADULT"}},
				{selector: "feed_versions.0.fare_products.#.fare_media.fare_media_name", expect: []string{"Cash", "Ticket"}},
				{selector: "feed_versions.0.fare_products.#.amount", expectUnique: []string{"1.25"}},
				{selector: "feed_versions.0.fare_products.#.currency", expectUnique: []string{"USD"}},
			},
		},
		{
			name:              "fare_products count",
			query:             `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_products { id } } }`,
			vars:              hw{"sha1": ctranFlexSha1},
			selector:          "feed_versions.0.fare_products.#.id",
			selectExpectCount: 14,
		},
		{
			name:  "fare_products without rider category or fare media",
			query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_products { fare_product_id rider_category_id fare_media { id } amount } } }`,
			vars:  hw{"sha1": ctSha1},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.fare_products.#.fare_product_id", expect: []string{"two_zone", "two_zone", "two_zone_peak", "express_upgrade", "day_pass"}},
				{selector: "feed_versions.0.fare_products.#.rider_category_id", expect: []string{"adult", "youth", "adult", "", "adult"}},
				{selector: "feed_versions.0.fare_products.#.fare_media", expect: []string{"", "", "", "", ""}},
				{selector: "feed_versions.0.fare_products.#.amount", expect: []string{"6.4", "3.2", "7.4", "1", "15"}},
			},
		},
		{
			name:  "fare_leg_rules",
			query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_leg_rules { leg_group_id network_id from_area_id to_area_id from_timeframe_group_id to_timeframe_group_id fare_product_id rule_priority } } }`,
			vars:  hw{"sha1": ctSha1},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.fare_leg_rules.#.leg_group_id", expect: []string{"ct_local", "ct_local", "ct_express"}},
				{selector: "feed_versions.0.fare_leg_rules.#.network_id", expect: []string{"local", "local", "express"}},
				{selector: "feed_versions.0.fare_leg_rules.#.from_area_id", expect: []string{"zone1", "zone1", ""}},
				{selector: "feed_versions.0.fare_leg_rules.#.to_area_id", expect: []string{"zone4", "zone4", ""}},
				{selector: "feed_versions.0.fare_leg_rules.#.from_timeframe_group_id", expect: []string{"", "weekday_peak", ""}},
				{selector: "feed_versions.0.fare_leg_rules.#.fare_product_id", expect: []string{"two_zone", "two_zone_peak", "two_zone"}},
				{selector: "feed_versions.0.fare_leg_rules.#.rule_priority", expect: []string{"0", "1", "0"}},
			},
		},
		{
			name:              "fare_leg_rules from a real feed",
			query:             `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_leg_rules { network_id fare_product_id } } }`,
			vars:              hw{"sha1": ctranFlexSha1},
			selector:          "feed_versions.0.fare_leg_rules.#.network_id",
			selectExpectCount: 3,
		},
		// Experimental fields from earlier Fares v2 proposal drafts and Interline extensions
		{
			name:  "fare_products duration",
			query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_products { fare_product_id duration_start duration_amount duration_unit duration_type } } }`,
			vars:  hw{"sha1": ctSha1},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.fare_products.4.fare_product_id", expect: []string{"day_pass"}},
				{selector: "feed_versions.0.fare_products.#.duration_start", expect: []string{"", "", "", "", "0"}},
				{selector: "feed_versions.0.fare_products.#.duration_amount", expect: []string{"", "", "", "", "1"}},
				{selector: "feed_versions.0.fare_products.#.duration_unit", expect: []string{"", "", "", "", "3"}},
				{selector: "feed_versions.0.fare_products.#.duration_type", expect: []string{"", "", "", "", "1"}},
			},
		},
		{
			name:         "fare_leg_rules transfer_only",
			query:        `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_leg_rules { transfer_only } } }`,
			vars:         hw{"sha1": ctSha1},
			selector:     "feed_versions.0.fare_leg_rules.#.transfer_only",
			selectExpect: []string{"", "", "1"},
		},
		{
			name:         "fare_transfer_rules filter_fare_product_id",
			query:        `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_transfer_rules { filter_fare_product_id } } }`,
			vars:         hw{"sha1": ctSha1},
			selector:     "feed_versions.0.fare_transfer_rules.#.filter_fare_product_id",
			selectExpect: []string{"two_zone"},
		},
		{
			name:  "rider_categories ages",
			query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { rider_categories { rider_category_id min_age max_age } } }`,
			vars:  hw{"sha1": ctSha1},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.rider_categories.#.rider_category_id", expect: []string{"adult", "youth"}},
				{selector: "feed_versions.0.rider_categories.#.min_age", expect: []string{"", "5"}},
				{selector: "feed_versions.0.rider_categories.#.max_age", expect: []string{"", "18"}},
			},
		},
		{
			name:   "fare_leg_join_rules",
			query:  `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_leg_join_rules { from_network_id to_network_id from_stop_id to_stop_id } } }`,
			vars:   hw{"sha1": ctSha1},
			expect: `{"feed_versions":[{"fare_leg_join_rules":[{"from_network_id":"local","from_stop_id":"70261","to_network_id":"express","to_stop_id":"70262"}]}]}`,
		},
		{
			name:   "fare_transfer_rules",
			query:  `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_transfer_rules { from_leg_group_id to_leg_group_id transfer_count duration_limit duration_limit_type fare_transfer_type fare_product_id } } }`,
			vars:   hw{"sha1": ctSha1},
			expect: `{"feed_versions":[{"fare_transfer_rules":[{"duration_limit":5400,"duration_limit_type":1,"fare_product_id":"express_upgrade","fare_transfer_type":0,"from_leg_group_id":"ct_local","to_leg_group_id":"ct_express","transfer_count":1}]}]}`,
		},
		{
			name:  "rider_categories",
			query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { rider_categories { rider_category_id rider_category_name is_default_fare_category eligibility_url } } }`,
			vars:  hw{"sha1": ctranFlexSha1},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.rider_categories.#.rider_category_id", expect: []string{"ADULT", "HONORED_CITIZEN", "YOUTH"}},
				{selector: "feed_versions.0.rider_categories.#.is_default_fare_category", expect: []string{"1", "0", "0"}},
				{selector: "feed_versions.0.rider_categories.#.eligibility_url", expect: []string{"", "https://c-tran.com/fares/fares-and-id-cards", "https://c-tran.com/fares/fares-and-id-cards"}},
			},
		},
		{
			name:   "timeframes",
			query:  `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { timeframes { timeframe_group_id start_time end_time service { service_id } } } }`,
			vars:   hw{"sha1": ctSha1},
			expect: `{"feed_versions":[{"timeframes":[{"end_time":"09:00:00","service":{"service_id":"mtwtf"},"start_time":"06:00:00","timeframe_group_id":"weekday_peak"}]}]}`,
		},
		{
			name:  "areas and stop_areas",
			query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { areas { area_id area_name stop_areas { area { area_id } stop { stop_id } } } } }`,
			vars:  hw{"sha1": ctSha1},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.areas.#.area_id", expect: []string{"zone1", "zone4"}},
				{selector: "feed_versions.0.areas.#.area_name", expect: []string{"Zone 1", "Zone 4"}},
				{selector: "feed_versions.0.areas.0.stop_areas.#.stop.stop_id", expectUnique: []string{"70011", "70012"}},
				{selector: "feed_versions.0.areas.1.stop_areas.#.stop.stop_id", expectUnique: []string{"70261", "70262"}},
				{selector: "feed_versions.0.areas.1.stop_areas.#.area.area_id", expectUnique: []string{"zone4"}},
			},
		},
		{
			name:  "networks and route_networks",
			query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { networks { network_id network_name } route_networks { network { network_id } route { route_id } } } }`,
			vars:  hw{"sha1": ctSha1},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.networks.#.network_id", expect: []string{"local", "express"}},
				{selector: "feed_versions.0.networks.#.network_name", expect: []string{"Local service", "Express service"}},
				{selector: "feed_versions.0.route_networks.#.route.route_id", expectUnique: []string{"Lo-130", "Bu-130"}},
				{selector: "feed_versions.0.route_networks.#.network.network_id", expectUnique: []string{"local", "express"}},
			},
		},
		{
			name:         "route network_id",
			query:        `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { routes { route_id network_id } } }`,
			vars:         hw{"sha1": ctSha1},
			selector:     "feed_versions.0.routes.#.network_id",
			selectExpect: []string{"", "", "", "", "", ""},
		},
		{
			// Several feed versions load in one batch; the limit applies to each.
			name:  "limit applies per feed version",
			query: `query { feed_versions(where: {feed_onestop_id: "BA"}) { sha1 fare_attributes(limit: 2) { id } } }`,
			f: func(t *testing.T, jj string) {
				counts := map[string]int{}
				for _, fv := range gjson.Get(jj, "feed_versions").Array() {
					counts[fv.Get("sha1").String()] = len(fv.Get("fare_attributes").Array())
				}
				assert.Equal(t, 2, counts[baSha1])
				assert.Equal(t, 2, counts["dd7aca4a8e4c90908fd3603c097fabee75fea907"])
			},
		},
		{
			name:   "feed version without fares",
			query:  `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_products { id } fare_leg_rules { id } areas { id } networks { id } } }`,
			vars:   hw{"sha1": baSha1},
			expect: `{"feed_versions":[{"areas":[],"fare_leg_rules":[],"fare_products":[],"networks":[]}]}`,
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}
