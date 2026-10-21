package findertest

// Cases duplicated from server/gql/fare_resolver_test.go.

// Fares v1 data comes from the CT and BA feeds; Fares v2 data comes from
// ctran-flex and from synthetic CT rows in testdata/server/test_supplement.pgsql.
func fareResolverTestcases() []testcase {
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
			name:   "fare_attributes transfers and agency are null when not set",
			query:  `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { fare_attributes(limit: 1) { transfers agency { agency_id } } } }`,
			vars:   hw{"sha1": ctSha1},
			expect: `{"feed_versions":[{"fare_attributes":[{"agency":null,"transfers":null}]}]}`,
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
		{
			name:                 "route network_id from routes.txt",
			query:                `query($sha1: String!) { feed_versions(where: {sha1: $sha1}) { routes { network_id } } }`,
			vars:                 hw{"sha1": ctranFlexSha1},
			selector:             "feed_versions.0.routes.#.network_id",
			selectExpectContains: []string{"LOCAL"},
		},
		{
			// Several feed versions load in one batch; the limit applies to each.
			name:  "limit applies per feed version",
			query: `query { feed_versions(where: {feed_onestop_id: "BA"}) { sha1 fare_attributes(limit: 2) { id } } }`,
			sel: []testcaseSelector{
				{selector: `feed_versions.#(sha1=="e535eb2b3b9ac3ef15d82c56575e914575e732e0").fare_attributes.#.id`, expectCount: 2},
				{selector: `feed_versions.#(sha1=="dd7aca4a8e4c90908fd3603c097fabee75fea907").fare_attributes.#.id`, expectCount: 2},
			},
		},
	}
	return testcases
}
