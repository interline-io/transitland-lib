package gqltest

import (
	"context"
	"strconv"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/model"
)

// Fares v1 data comes from the CT and BA feeds; Fares v2 data comes from
// ctran-flex and from the supplemental feed.
func fareResolverTestcases(t testing.TB, cfg model.Config) []testcase {
	supplementalSha1 := testconfig.SupplementalSha1(t, cfg)
	ctSha1 := "d2813c293bcfd7a97dde599527ae6c62c98e66c6"
	baSha1 := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	testcases := []testcase{
		// Fares v1
		{
			Name:         "fare_attributes",
			Query:        `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_attributes { fare_id price currency_type payment_method transfers transfer_duration agency { agency_id } feed_version_sha1 feed_onestop_id feed_version { sha1 } } } }`,
			Vars:         hw{"sha1": ctSha1},
			Selector:     "feed_versions.0.fare_attributes.#.fare_id",
			SelectExpect: []string{"OW_1_20160228", "OW_2_20160228", "OW_3_20160228", "OW_4_20160228", "OW_5_20160228", "OW_6_20160228"},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.fare_attributes.#.price", Expect: []string{"3.75", "6", "8.25", "10.5", "12.75", "15"}},
				{Selector: "feed_versions.0.fare_attributes.#.currency_type", ExpectUnique: []string{"USD"}},
				{Selector: "feed_versions.0.fare_attributes.#.payment_method", ExpectUnique: []string{"1"}},
				{Selector: "feed_versions.0.fare_attributes.#.transfer_duration", ExpectUnique: []string{"14400"}},
				{Selector: "feed_versions.0.fare_attributes.#.feed_version.sha1", ExpectUnique: []string{ctSha1}},
				{Selector: "feed_versions.0.fare_attributes.#.feed_version_sha1", ExpectUnique: []string{ctSha1}},
				{Selector: "feed_versions.0.fare_attributes.#.feed_onestop_id", ExpectUnique: []string{"CT"}},
			},
		},
		{
			Name:   "fare_attributes transfers and agency are null when not set",
			Query:  `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_attributes(limit: 1) { transfers agency { agency_id } } } }`,
			Vars:   hw{"sha1": ctSha1},
			Expect: `{"feed_versions":[{"fare_attributes":[{"agency":null,"transfers":null}]}]}`,
		},
		{
			Name:  "fare_rules",
			Query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_rules(limit: 3) { fare_attribute { fare_id } route { route_id } origin_id destination_id contains_id } } }`,
			Vars:  hw{"sha1": ctSha1},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.fare_rules.#.fare_attribute.fare_id", Expect: []string{"OW_1_20160228", "OW_2_20160228", "OW_3_20160228"}},
				{Selector: "feed_versions.0.fare_rules.#.route.route_id", ExpectUnique: []string{"Bu-130"}},
				{Selector: "feed_versions.0.fare_rules.#.origin_id", ExpectUnique: []string{"1"}},
				{Selector: "feed_versions.0.fare_rules.#.destination_id", Expect: []string{"1", "2", "3"}},
			},
		},
		{
			Name:              "fare_rules default limit",
			Query:             `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_rules { id } } }`,
			Vars:              hw{"sha1": baSha1},
			Selector:          "feed_versions.0.fare_rules.#.id",
			SelectExpectCount: 100,
		},
		{
			Name:              "fare_rules above the default limit",
			Query:             `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_rules(limit: 10000) { id } } }`,
			Vars:              hw{"sha1": baSha1},
			Selector:          "feed_versions.0.fare_rules.#.id",
			SelectExpectCount: 2304,
		},
		// Fares v2
		{
			Name:  "fare_media",
			Query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_media { fare_media_id fare_media_name fare_media_type } } }`,
			Vars:  hw{"sha1": ctranFlexSha1},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.fare_media.#.fare_media_id", Expect: []string{"0", "1", "2", "3", "4"}},
				{Selector: "feed_versions.0.fare_media.#.fare_media_name", Expect: []string{"Cash", "Ticket", "HOP Fastpass", "Open Payment", "Virtual HOP Fastpass"}},
				{Selector: "feed_versions.0.fare_media.#.fare_media_type", Expect: []string{"0", "1", "2", "3", "4"}},
			},
		},
		{
			Name:  "fare_products",
			Query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_products(limit: 2) { fare_product_id fare_product_name rider_category_id fare_media { fare_media_name } amount currency } } }`,
			Vars:  hw{"sha1": ctranFlexSha1},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.fare_products.#.fare_product_id", ExpectUnique: []string{"ADULT_LOCAL_SINGLE_RIDE"}},
				{Selector: "feed_versions.0.fare_products.#.rider_category_id", ExpectUnique: []string{"ADULT"}},
				{Selector: "feed_versions.0.fare_products.#.fare_media.fare_media_name", Expect: []string{"Cash", "Ticket"}},
				{Selector: "feed_versions.0.fare_products.#.amount", ExpectUnique: []string{"1.25"}},
				{Selector: "feed_versions.0.fare_products.#.currency", ExpectUnique: []string{"USD"}},
			},
		},
		{
			Name:              "fare_products count",
			Query:             `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_products { id } } }`,
			Vars:              hw{"sha1": ctranFlexSha1},
			Selector:          "feed_versions.0.fare_products.#.id",
			SelectExpectCount: 14,
		},
		{
			Name:  "fare_products without rider category or fare media",
			Query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_products { fare_product_id rider_category_id fare_media { id } amount duration_start duration_amount duration_unit duration_type } } }`,
			Vars:  hw{"sha1": supplementalSha1},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.fare_products.#.fare_product_id", Expect: []string{"two_zone", "two_zone", "two_zone_peak", "express_upgrade", "day_pass"}},
				{Selector: "feed_versions.0.fare_products.#.rider_category_id", Expect: []string{"adult", "youth", "adult", "", "adult"}},
				{Selector: "feed_versions.0.fare_products.#.fare_media", Expect: []string{"", "", "", "", ""}},
				{Selector: "feed_versions.0.fare_products.#.amount", Expect: []string{"6.4", "3.2", "7.4", "1", "15"}},
				// Experimental duration fields
				{Selector: "feed_versions.0.fare_products.#.duration_start", Expect: []string{"", "", "", "", "0"}},
				{Selector: "feed_versions.0.fare_products.#.duration_amount", Expect: []string{"", "", "", "", "1"}},
				{Selector: "feed_versions.0.fare_products.#.duration_unit", Expect: []string{"", "", "", "", "3"}},
				{Selector: "feed_versions.0.fare_products.#.duration_type", Expect: []string{"", "", "", "", "1"}},
			},
		},
		{
			Name:  "fare_leg_rules",
			Query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_leg_rules { leg_group_id network_id from_area { area_id } to_area { area_id } from_timeframe_group_id to_timeframe_group_id fare_product_id rule_priority transfer_only } } }`,
			Vars:  hw{"sha1": supplementalSha1},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.fare_leg_rules.#.leg_group_id", Expect: []string{"local_legs", "local_legs", "express_legs"}},
				{Selector: "feed_versions.0.fare_leg_rules.#.network_id", Expect: []string{"local", "local", "express"}},
				// The express_legs rule has no areas
				{Selector: "feed_versions.0.fare_leg_rules.#.from_area.area_id", Expect: []string{"north", "north"}},
				{Selector: "feed_versions.0.fare_leg_rules.#.to_area.area_id", Expect: []string{"south", "south"}},
				{Selector: "feed_versions.0.fare_leg_rules.#.from_timeframe_group_id", Expect: []string{"", "weekday_peak", ""}},
				{Selector: "feed_versions.0.fare_leg_rules.#.fare_product_id", Expect: []string{"two_zone", "two_zone_peak", "two_zone"}},
				{Selector: "feed_versions.0.fare_leg_rules.#.rule_priority", Expect: []string{"0", "1", "0"}},
				{Selector: "feed_versions.0.fare_leg_rules.#.transfer_only", Expect: []string{"", "", "1"}},
			},
		},
		{
			// ctran-flex names its network only in routes.network_id, so the
			// importer keeps the GTFS id in fare_leg_rules.network_id.
			Name:         "fare_leg_rules network from routes.network_id",
			Query:        `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_leg_rules { network_id } networks { id } } }`,
			Vars:         hw{"sha1": ctranFlexSha1},
			Selector:     "feed_versions.0.fare_leg_rules.#.network_id",
			SelectExpect: []string{"LOCAL", "LOCAL", "LOCAL"},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.networks.#.id", Expect: []string{}},
			},
		},
		{
			Name:   "fare_leg_join_rules",
			Query:  `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_leg_join_rules { from_network_id to_network_id from_stop { stop_id } to_stop { stop_id } } } }`,
			Vars:   hw{"sha1": supplementalSha1},
			Expect: `{"feed_versions":[{"fare_leg_join_rules":[{"from_network_id":"local","from_stop":{"stop_id":"S1"},"to_network_id":"express","to_stop":{"stop_id":"S2"}}]}]}`,
		},
		{
			Name:  "fare_transfer_rules",
			Query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_transfer_rules { from_leg_group_id to_leg_group_id transfer_count duration_limit duration_limit_type fare_transfer_type fare_product_id filter_fare_product_id } } }`,
			Vars:  hw{"sha1": supplementalSha1},
			Expect: `{"feed_versions":[{"fare_transfer_rules":[` +
				`{"duration_limit":5400,"duration_limit_type":1,"fare_product_id":"express_upgrade","fare_transfer_type":0,"filter_fare_product_id":"two_zone","from_leg_group_id":"local_legs","to_leg_group_id":"express_legs","transfer_count":null},` +
				`{"duration_limit":3600,"duration_limit_type":0,"fare_product_id":null,"fare_transfer_type":0,"filter_fare_product_id":null,"from_leg_group_id":"local_legs","to_leg_group_id":"local_legs","transfer_count":1}` +
				`]}]}`,
		},
		{
			Name:  "rider_categories",
			Query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { rider_categories { rider_category_id rider_category_name is_default_fare_category eligibility_url } } }`,
			Vars:  hw{"sha1": ctranFlexSha1},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.rider_categories.#.rider_category_id", Expect: []string{"ADULT", "HONORED_CITIZEN", "YOUTH"}},
				{Selector: "feed_versions.0.rider_categories.#.is_default_fare_category", Expect: []string{"1", "0", "0"}},
				{Selector: "feed_versions.0.rider_categories.#.eligibility_url", Expect: []string{"", "https://c-tran.com/fares/fares-and-id-cards", "https://c-tran.com/fares/fares-and-id-cards"}},
			},
		},
		{
			// Experimental age fields
			Name:  "rider_categories ages",
			Query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { rider_categories { rider_category_id min_age max_age } } }`,
			Vars:  hw{"sha1": supplementalSha1},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.rider_categories.#.rider_category_id", Expect: []string{"adult", "youth"}},
				{Selector: "feed_versions.0.rider_categories.#.min_age", Expect: []string{"", "5"}},
				{Selector: "feed_versions.0.rider_categories.#.max_age", Expect: []string{"", "18"}},
			},
		},
		{
			Name:   "timeframes",
			Query:  `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { timeframes { timeframe_group_id start_time end_time service { service_id } } } }`,
			Vars:   hw{"sha1": supplementalSha1},
			Expect: `{"feed_versions":[{"timeframes":[{"end_time":"09:00:00","service":{"service_id":"weekday"},"start_time":"06:00:00","timeframe_group_id":"weekday_peak"}]}]}`,
		},
		{
			Name:  "areas and stop_areas",
			Query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { areas { area_id area_name stop_areas { area { area_id } stop { stop_id } } } } }`,
			Vars:  hw{"sha1": supplementalSha1},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.areas.#.area_id", Expect: []string{"north", "south"}},
				{Selector: "feed_versions.0.areas.#.area_name", Expect: []string{"North zone", "South zone"}},
				{Selector: "feed_versions.0.areas.0.stop_areas.#.stop.stop_id", ExpectUnique: []string{"N1", "N2"}},
				{Selector: "feed_versions.0.areas.1.stop_areas.#.stop.stop_id", ExpectUnique: []string{"S1", "S2"}},
				{Selector: "feed_versions.0.areas.1.stop_areas.#.area.area_id", ExpectUnique: []string{"south"}},
			},
		},
		{
			Name:  "networks and route_networks",
			Query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { networks { network_id network_name } route_networks { network { network_id } route { route_id } } } }`,
			Vars:  hw{"sha1": supplementalSha1},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.networks.#.network_id", Expect: []string{"local", "express"}},
				{Selector: "feed_versions.0.networks.#.network_name", Expect: []string{"Local service", "Express service"}},
				{Selector: "feed_versions.0.route_networks.#.route.route_id", ExpectUnique: []string{"L", "X"}},
				{Selector: "feed_versions.0.route_networks.#.network.network_id", ExpectUnique: []string{"local", "express"}},
			},
		},
		{
			// The supplemental feed assigns networks with route_networks.txt, so routes.network_id is empty
			Name:         "route network_id",
			Query:        `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { routes { route_id network_id } } }`,
			Vars:         hw{"sha1": supplementalSha1},
			Selector:     "feed_versions.0.routes.#.network_id",
			SelectExpect: []string{"", "", ""},
		},
		{
			Name:                 "route network_id from routes.txt",
			Query:                `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { routes { network_id } } }`,
			Vars:                 hw{"sha1": ctranFlexSha1},
			Selector:             "feed_versions.0.routes.#.network_id",
			SelectExpectContains: []string{"LOCAL"},
		},
		// Records matched by GTFS id or group id, and reverse links from stops and routes
		{
			Name:  "fare_leg_rules group references",
			Query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_leg_rules { leg_group_id fare_products { fare_product_id rider_category_id } from_timeframes { timeframe_group_id } to_timeframes { timeframe_group_id } } } }`,
			Vars:  hw{"sha1": supplementalSha1},
			Expect: `{"feed_versions":[{"fare_leg_rules":[` +
				`{"leg_group_id":"local_legs","fare_products":[{"fare_product_id":"two_zone","rider_category_id":"adult"},{"fare_product_id":"two_zone","rider_category_id":"youth"}],"from_timeframes":[],"to_timeframes":[]},` +
				`{"leg_group_id":"local_legs","fare_products":[{"fare_product_id":"two_zone_peak","rider_category_id":"adult"}],"from_timeframes":[{"timeframe_group_id":"weekday_peak"}],"to_timeframes":[]},` +
				`{"leg_group_id":"express_legs","fare_products":[{"fare_product_id":"two_zone","rider_category_id":"adult"},{"fare_product_id":"two_zone","rider_category_id":"youth"}],"from_timeframes":[],"to_timeframes":[]}` +
				`]}]}`,
		},
		{
			Name:  "fare_transfer_rules group references",
			Query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_transfer_rules { from_leg_rules { leg_group_id fare_product_id } to_leg_rules { leg_group_id fare_product_id } fare_products { fare_product_id } filter_fare_products { fare_product_id rider_category_id } } } }`,
			Vars:  hw{"sha1": supplementalSha1},
			Expect: `{"feed_versions":[{"fare_transfer_rules":[{` +
				`"from_leg_rules":[{"leg_group_id":"local_legs","fare_product_id":"two_zone"},{"leg_group_id":"local_legs","fare_product_id":"two_zone_peak"}],` +
				`"to_leg_rules":[{"leg_group_id":"express_legs","fare_product_id":"two_zone"}],` +
				`"fare_products":[{"fare_product_id":"express_upgrade"}],` +
				`"filter_fare_products":[{"fare_product_id":"two_zone","rider_category_id":"adult"},{"fare_product_id":"two_zone","rider_category_id":"youth"}]` +
				`},{` +
				`"from_leg_rules":[{"leg_group_id":"local_legs","fare_product_id":"two_zone"},{"leg_group_id":"local_legs","fare_product_id":"two_zone_peak"}],` +
				`"to_leg_rules":[{"leg_group_id":"local_legs","fare_product_id":"two_zone"},{"leg_group_id":"local_legs","fare_product_id":"two_zone_peak"}],` +
				`"fare_products":[],` +
				`"filter_fare_products":[]` +
				`}]}]}`,
		},
		{
			Name:  "fare_products rider_category",
			Query: `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_products { fare_product_id rider_category { rider_category_id rider_category_name } } } }`,
			Vars:  hw{"sha1": supplementalSha1},
			Expect: `{"feed_versions":[{"fare_products":[` +
				`{"fare_product_id":"two_zone","rider_category":{"rider_category_id":"adult","rider_category_name":"Adult"}},` +
				`{"fare_product_id":"two_zone","rider_category":{"rider_category_id":"youth","rider_category_name":"Youth"}},` +
				`{"fare_product_id":"two_zone_peak","rider_category":{"rider_category_id":"adult","rider_category_name":"Adult"}},` +
				`{"fare_product_id":"express_upgrade","rider_category":null},` +
				`{"fare_product_id":"day_pass","rider_category":{"rider_category_id":"adult","rider_category_name":"Adult"}}` +
				`]}]}`,
		},
		{
			Name:   "stop areas",
			Query:  `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { a: stops(where: {stop_id: "N1"}) { areas { area_id } } b: stops(where: {stop_id: "S2"}) { areas { area_id } } c: stops(where: {stop_id: "C"}) { areas { area_id } } } }`,
			Vars:   hw{"sha1": supplementalSha1},
			Expect: `{"feed_versions":[{"a":[{"areas":[{"area_id":"north"}]}],"b":[{"areas":[{"area_id":"south"}]}],"c":[{"areas":[]}]}]}`,
		},
		{
			Name:   "route networks",
			Query:  `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { a: routes(where: {route_id: "L"}) { networks { network_id } } b: routes(where: {route_id: "X"}) { networks { network_id } } c: routes(where: {route_id: "SH"}) { networks { network_id } } } }`,
			Vars:   hw{"sha1": supplementalSha1},
			Expect: `{"feed_versions":[{"a":[{"networks":[{"network_id":"local"}]}],"b":[{"networks":[{"network_id":"express"}]}],"c":[{"networks":[]}]}]}`,
		},
		{
			// ctran-flex names its network only in routes.network_id, so there is no network record
			Name:   "route networks from routes.network_id",
			Query:  `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { routes(where: {route_id: "0553af3e-53b8-4f98-ba47-0fc03d2404de"}) { network_id networks { network_id } } } }`,
			Vars:   hw{"sha1": ctranFlexSha1},
			Expect: `{"feed_versions":[{"routes":[{"network_id":"LOCAL","networks":[]}]}]}`,
		},
		{
			// Several feed versions load in one batch; the limit applies to each.
			Name:  "limit applies per feed version",
			Query: `query { feed_versions(where: {feed_onestop_id: "BA"}) { sha1 fare_attributes(limit: 2) { id } } }`,
			Sel: []testcaseSelector{
				{Selector: `feed_versions.#(sha1=="e535eb2b3b9ac3ef15d82c56575e914575e732e0").fare_attributes.#.id`, ExpectCount: 2},
				{Selector: `feed_versions.#(sha1=="dd7aca4a8e4c90908fd3603c097fabee75fea907").fare_attributes.#.id`, ExpectCount: 2},
			},
		},
		{
			Name:   "feed version without fares",
			Query:  `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_products { id } fare_leg_rules { id } areas { id } networks { id } } }`,
			Vars:   hw{"sha1": baSha1},
			Expect: `{"feed_versions":[{"areas":[],"fare_leg_rules":[],"fare_products":[],"networks":[]}]}`,
		},
	}
	return testcases
}

func feedVersionResolverStopAreasTestcases(t testing.TB, cfg model.Config) []testcase {
	supplementalSha1 := testconfig.SupplementalSha1(t, cfg)
	testcases := []testcase{
		{
			Name:  "every stop area in the feed version",
			Query: `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { stop_areas { area { area_id } stop { stop_id } feed_version_sha1 } } }`,
			Vars:  hw{"sha1": supplementalSha1},
			Sel: []testcaseSelector{
				{Selector: `feed_versions.0.stop_areas.#(area.area_id=="north")#.stop.stop_id`, Expect: []string{"N1", "N2"}},
				{Selector: `feed_versions.0.stop_areas.#(area.area_id=="south")#.stop.stop_id`, Expect: []string{"S1", "S2"}},
				{Selector: "feed_versions.0.stop_areas.#.feed_version_sha1", ExpectUnique: []string{supplementalSha1}},
			},
		},
	}
	return testcases
}

// fareResolverCursorTestcases pages through fare rules and stop areas with
// after.
func fareResolverCursorTestcases(t testing.TB, cfg model.Config) []testcase {
	supplementalSha1 := testconfig.SupplementalSha1(t, cfg)
	ctx := model.WithConfig(context.Background(), cfg)

	baSha1 := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
	fvids := map[string]int{}
	for _, sha1 := range []string{baSha1, supplementalSha1} {
		fvs, err := cfg.Finder.FindFeedVersions(ctx, nil, nil, nil, &model.FeedVersionFilter{Sha1: &sha1})
		if err != nil {
			t.Fatal(err)
		}
		if len(fvs) == 0 {
			t.Fatalf("feed version %s not found", sha1)
		}
		fvids[sha1] = fvs[0].ID
	}
	fareRuleGroups, err := cfg.Finder.FareRulesByFeedVersionIDs(ctx, nil, nil, []int{fvids[baSha1]})
	if err != nil {
		t.Fatal(err)
	}
	needAtLeast(t, 1, "fare rule groups", fareRuleGroups)
	fareRules := fareRuleGroups[0]
	needAtLeast(t, 1, "fare rules", fareRules)
	var fareRuleIDs []string
	for _, ent := range fareRules {
		fareRuleIDs = append(fareRuleIDs, strconv.Itoa(ent.ID))
	}
	areaGroups, err := cfg.Finder.AreasByFeedVersionIDs(ctx, nil, nil, []int{fvids[supplementalSha1]})
	if err != nil {
		t.Fatal(err)
	}
	needAtLeast(t, 1, "area groups", areaGroups)
	needAtLeast(t, 1, "areas", areaGroups[0])
	stopAreaGroups, err := cfg.Finder.StopAreasByAreaIDs(ctx, nil, nil, []int{areaGroups[0][0].ID})
	if err != nil {
		t.Fatal(err)
	}
	needAtLeast(t, 1, "stop area groups", stopAreaGroups)
	stopAreas := stopAreaGroups[0]
	needAtLeast(t, 1, "stop areas", stopAreas)
	var stopAreaIDs []string
	for _, ent := range stopAreas {
		stopAreaIDs = append(stopAreaIDs, strconv.Itoa(ent.ID))
	}

	fareRulesQuery := `query($sha1: String!, $after: Int) { feed_versions(where: {sha1: $sha1}) { fare_rules(limit: 10000, after: $after) { id } } }`
	stopAreasQuery := `query($sha1: String!, $after: Int) { feed_versions(where: {sha1: $sha1}) { areas(limit: 1) { stop_areas(after: $after) { id } } } }`
	testcases := []testcase{
		{
			Name:         "fare_rules no cursor",
			Query:        fareRulesQuery,
			Vars:         hw{"sha1": baSha1},
			Selector:     "feed_versions.0.fare_rules.#.id",
			SelectExpect: fareRuleIDs,
		},
		{
			Name:         "fare_rules after first",
			Query:        fareRulesQuery,
			Vars:         hw{"sha1": baSha1, "after": fareRules[0].ID},
			Selector:     "feed_versions.0.fare_rules.#.id",
			SelectExpect: fareRuleIDs[1:],
		},
		{
			Name:         "stop_areas after first",
			Query:        stopAreasQuery,
			Vars:         hw{"sha1": supplementalSha1, "after": stopAreas[0].ID},
			Selector:     "feed_versions.0.areas.0.stop_areas.#.id",
			SelectExpect: stopAreaIDs[1:],
		},
	}
	return testcases
}
