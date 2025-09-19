package gql

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/testdata"
)

func TestAgencyResolver(t *testing.T) {
	vars := hw{"agency_id": "caltrain-ca-us"}
	testcases := []testcase{
		{
			name:         "basic",
			query:        `query { agencies {agency_id}}`,
			selector:     "agencies.#.agency_id",
			selectExpect: []string{"caltrain-ca-us", "BART", ""},
		},
		{
			name:   "basic fields",
			query:  `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {onestop_id agency_id agency_name agency_lang agency_phone agency_timezone agency_url agency_email agency_fare_url feed_version_sha1 feed_onestop_id}}`,
			vars:   vars,
			expect: `{"agencies":[{"agency_email":null,"agency_fare_url":null,"agency_id":"caltrain-ca-us","agency_lang":"en","agency_name":"Caltrain","agency_phone":"800-660-4287","agency_timezone":"America/Los_Angeles","agency_url":"http://www.caltrain.com","feed_onestop_id":"CT","feed_version_sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6","onestop_id":"o-9q9-caltrain"}]}`,
		},
		{
			// just ensure this query completes successfully; checking coordinates is a pain and flaky.
			name:         "geometry",
			query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {geometry}}`,
			vars:         vars,
			selector:     "agencies.0.geometry.type",
			selectExpect: []string{"Polygon"},
		},
		{
			name:         "near 100m",
			query:        `query {agencies(where:{near:{lon:-122.407974,lat:37.784471,radius:100.0}}) {agency_id}}`,
			selector:     "agencies.#.agency_id",
			selectExpect: []string{"BART"},
		},
		{
			name:         "near 10000m",
			query:        `query {agencies(where:{near:{lon:-122.407974,lat:37.784471,radius:10000.0}}) {agency_id}}`,
			selector:     "agencies.#.agency_id",
			selectExpect: []string{"caltrain-ca-us", "BART"},
		},
		{
			name:         "within polygon",
			query:        `query{agencies(where:{within:{type:"Polygon",coordinates:[[[-122.39803791046143,37.794626736533836],[-122.40106344223022,37.792303711508595],[-122.3965573310852,37.789641468930114],[-122.3938751220703,37.792354581451946],[-122.39803791046143,37.794626736533836]]]}}){agency_id}}`,
			selector:     "agencies.#.agency_id",
			selectExpect: []string{"BART"},
		},
		{
			name:         "within polygon big",
			query:        `query{agencies(where:{within:{type:"Polygon",coordinates:[[[-122.39481925964355,37.80151060070086],[-122.41653442382812,37.78652126637423],[-122.39662170410156,37.76847577247014],[-122.37301826477051,37.784757615348575],[-122.39481925964355,37.80151060070086]]]}}){id agency_id}}`,
			selector:     "agencies.#.agency_id",
			selectExpect: []string{"caltrain-ca-us", "BART"},
		},
		{
			name:         "where bbox 1",
			query:        `query($bbox:BoundingBox) {agencies(where:{bbox:$bbox}) {agency_id}}`,
			vars:         hw{"bbox": hw{"min_lon": -122.2698781543005, "min_lat": 37.80700393130445, "max_lon": -122.2677640139239, "max_lat": 37.8088734037938}},
			selector:     "agencies.#.agency_id",
			selectExpect: []string{"BART"},
		},
		{
			name:         "where bbox 2",
			query:        `query($bbox:BoundingBox) {agencies(where:{bbox:$bbox}) {agency_id}}`,
			vars:         hw{"bbox": hw{"min_lon": -124.3340029563042, "min_lat": 40.65505368922123, "max_lon": -123.9653594784379, "max_lat": 40.896440342606525}},
			selector:     "agencies.#.agency_id",
			selectExpect: []string{},
		},
		{
			name:        "where bbox too large",
			query:       `query($bbox:BoundingBox) {agencies(where:{bbox:$bbox}) {agency_id}}`,
			vars:        hw{"bbox": hw{"min_lon": -137.88020156441956, "min_lat": 30.072648315782004, "max_lon": -109.00421121090919, "max_lat": 45.02437957865729}},
			expectError: true,
			f: func(t *testing.T, jj string) {
			},
		},
		{
			name:   "feed_version",
			query:  `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {feed_version { sha1 }}}`,
			vars:   vars,
			expect: `{"agencies":[{"feed_version":{"sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6"}}]}`,
		},
		{
			name:         "routes",
			query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {routes { route_id }}}`,
			vars:         vars,
			selector:     "agencies.0.routes.#.route_id",
			selectExpect: []string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
		},
		// places should test filters because it's not a root resolver
		{
			name:         "places",
			query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {places {adm0_name adm0_iso adm1_name adm1_iso city_name}}}`,
			vars:         vars,
			selector:     "agencies.0.places.#.city_name",
			selectExpect: []string{"San Mateo", "San Francisco", "San Jose", ""},
		},
		{
			name:         "places get adm0_iso",
			query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {places {adm0_name adm0_iso adm1_name adm1_iso city_name}}}`,
			vars:         vars,
			selector:     "agencies.0.places.#.adm0_iso",
			selectExpect: []string{"US", "US", "US", "US"},
		},
		{
			name:         "places get adm1_iso",
			query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {places {adm0_name adm0_iso adm1_name adm1_iso city_name}}}`,
			vars:         vars,
			selector:     "agencies.0.places.#.adm1_iso",
			selectExpect: []string{"US-CA", "US-CA", "US-CA", "US-CA"},
		},

		{
			name:         "places rank 0.25",
			query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {places(where:{min_rank:0.25}) {adm0_name adm1_name city_name}}}`,
			vars:         vars,
			selector:     "agencies.0.places.#.city_name",
			selectExpect: []string{"San Mateo", "San Jose", ""},
		},
		{
			name:         "places rank 0.75",
			query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {places(where:{min_rank:0.75}) {adm0_name adm1_name city_name}}}`,
			vars:         vars,
			selector:     "agencies.0.places.#.adm1_name",
			selectExpect: []string{"California"},
		},
		// place iso codes
		{
			name:         "places iso3166 country",
			query:        `query { agencies(where:{adm0_iso: "US"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			vars:         vars,
			selector:     "agencies.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit"},
		},
		{
			name:         "places iso3166 state",
			query:        `query { agencies(where:{adm1_iso: "US-CA"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			vars:         vars,
			selector:     "agencies.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			name:         "places iso3166 state lowercase",
			query:        `query { agencies(where:{adm1_iso: "us-ca"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			vars:         vars,
			selector:     "agencies.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			name:         "places iso3166 state and country",
			query:        `query { agencies(where:{adm0_iso: "us", adm1_iso: "us-ca"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			vars:         vars,
			selector:     "agencies.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			name:         "places iso3166 state and city",
			query:        `query { agencies(where:{city_name: "oakland", adm1_iso: "us-ca"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			vars:         vars,
			selector:     "agencies.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			name:         "places iso3166 state and city no result",
			query:        `query { agencies(where:{city_name: "test", adm1_iso: "us-ca"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			vars:         vars,
			selector:     "agencies.#.onestop_id",
			selectExpect: []string{},
		},
		{
			name:         "places iso3166 state no results",
			query:        `query { agencies(where:{adm1_iso: "US-NY"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			vars:         vars,
			selector:     "agencies.#.onestop_id",
			selectExpect: []string{},
		},
		{
			name:         "places state",
			query:        `query { agencies(where:{adm1_name: "California"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			vars:         vars,
			selector:     "agencies.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			name:         "places state no result",
			query:        `query { agencies(where:{adm1_name: "New York"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			vars:         vars,
			selector:     "agencies.#.onestop_id",
			selectExpect: []string{},
		},
		{
			name:         "places city",
			query:        `query { agencies(where:{city_name: "Berkeley"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			vars:         vars,
			selector:     "agencies.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			name:         "places city 2",
			query:        `query { agencies(where:{city_name: "San Jose"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			vars:         vars,
			selector:     "agencies.#.onestop_id",
			selectExpect: []string{"o-9q9-caltrain"},
		},
		{
			name:         "places city 2 lowercase",
			query:        `query { agencies(where:{city_name: "san jose"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			vars:         vars,
			selector:     "agencies.#.onestop_id",
			selectExpect: []string{"o-9q9-caltrain"},
		},
		// search
		{
			name:         "search",
			query:        `query($search:String!) { agencies(where:{search:$search}) {agency_id}}`,
			vars:         hw{"search": "Bay Area"},
			selector:     "agencies.#.agency_id",
			selectExpect: []string{"BART"},
		},
		{
			name:         "search",
			query:        `query($search:String!) { agencies(where:{search:$search}) {agency_id}}`,
			vars:         hw{"search": "caltrain"},
			selector:     "agencies.#.agency_id",
			selectExpect: []string{"caltrain-ca-us"},
		},
		// TODO
		// {"census_geographies", }
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

func TestAgencyResolver_Cursor(t *testing.T) {
	c, cfg := newTestClient(t)
	allEnts, err := cfg.Finder.FindAgencies(model.WithConfig(context.Background(), cfg), nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	allIds := []string{}
	for _, ent := range allEnts {
		allIds = append(allIds, ent.AgencyID.Val)
	}
	testcases := []testcase{
		{
			name:         "no cursor",
			query:        "query{agencies(limit:10){feed_version{id} id agency_id}}",
			selector:     "agencies.#.agency_id",
			selectExpect: allIds,
		},
		{
			name:         "after 0",
			query:        "query{agencies(after: 0, limit:10){feed_version{id} id agency_id}}",
			selector:     "agencies.#.agency_id",
			selectExpect: allIds,
		},
		{
			name:         "after 1st",
			query:        "query($after: Int!){agencies(after: $after, limit:10){feed_version{id} id agency_id}}",
			vars:         hw{"after": allEnts[1].ID},
			selector:     "agencies.#.agency_id",
			selectExpect: allIds[2:],
		},
	}
	queryTestcases(t, c, testcases)
}

var fgaTestTuples = []authz.TupleKey{
	{
		Subject:  authz.NewEntityKey(authz.UserType, "ian"),
		Object:   authz.NewEntityKey(authz.TenantType, "tl-tenant"),
		Relation: authz.AdminRelation,
	},
	{
		Object:   authz.NewEntityKey(authz.GroupType, "BA-group"),
		Subject:  authz.NewEntityKey(authz.TenantType, "tl-tenant"),
		Relation: authz.ParentRelation,
	},
	// This is a public feed
	{
		Subject:  authz.NewEntityKey(authz.GroupType, "BA-group"),
		Object:   authz.NewEntityKey(authz.FeedType, "BA"),
		Relation: authz.ParentRelation,
	},
	// This is a non-public feed
	{
		Subject:  authz.NewEntityKey(authz.GroupType, "BA-group"),
		Object:   authz.NewEntityKey(authz.FeedType, "EG"),
		Relation: authz.ParentRelation,
	},
}

func TestAgencyResolver_Authz(t *testing.T) {
	ep, a, ok := testutil.CheckEnv("TL_TEST_FGA_ENDPOINT")
	if !ok {
		t.Skip(a)
		return
	}
	cfg := testconfig.Config(t, testconfig.Options{
		FGAEndpoint:    ep,
		FGAModelFile:   testdata.Path("server/authz/tls.json"),
		FGAModelTuples: fgaTestTuples,
	})

	srv, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}

	// Add config and perms middleware
	srv = model.AddConfigAndPerms(cfg, srv)

	testcases := []testcase{
		{
			name:         "basic",
			query:        `query { agencies {agency_id}}`,
			user:         "ian",
			selector:     "agencies.#.agency_id",
			selectExpect: []string{"caltrain-ca-us", "BART", "", "573"},
		},
		{
			name:         "basic",
			query:        `query { agencies {agency_id}}`,
			user:         "public",
			selector:     "agencies.#.agency_id",
			selectExpect: []string{"caltrain-ca-us", "BART", ""},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			c := client.New(usercheck.UserDefaultMiddleware(tc.user)(srv))
			queryTestcase(t, c, tc)
		})
	}
}

func TestAgencyResolver_License(t *testing.T) {
	q := `
	query ($lic: LicenseFilter) {
		agencies(limit: 10000, where: {license: $lic}) {
		  agency_id
		  feed_version {
			feed {
			  onestop_id
			  license {
				share_alike_optional
				create_derived_product
				commercial_use_allowed
				redistribution_allowed
			  }
			}
		  }
		}
	  }	  
	`
	testcases := []testcase{
		// license: share_alike_optional
		{
			name:               "license filter: share_alike_optional = yes",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "YES"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: share_alike_optional = no",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: share_alike_optional = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "EXCLUDE_NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA"},
			selectExpectCount:  2,
		},
		// license: create_derived_product
		{
			name:               "license filter: create_derived_product = yes",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "YES"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: create_derived_product = no",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: create_derived_product = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "EXCLUDE_NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA"},
			selectExpectCount:  2,
		},
		// license: commercial_use_allowed
		{
			name:               "license filter: commercial_use_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "YES"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: commercial_use_allowed = no",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: commercial_use_allowed = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "EXCLUDE_NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA"},
			selectExpectCount:  2,
		},
		// license: redistribution_allowed
		{
			name:               "license filter: redistribution_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "YES"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: redistribution_allowed = no",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: redistribution_allowed = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "EXCLUDE_NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA"},
			selectExpectCount:  2,
		},
		// license: use_without_attribution
		{
			name:               "license filter: use_without_attribution = yes",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "YES"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: use_without_attribution = no",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: use_without_attribution = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "EXCLUDE_NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA"},
			selectExpectCount:  2,
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}
