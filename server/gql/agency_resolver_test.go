package gql

import (
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
			selectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"},
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
			// `%` is an ILIKE wildcard; unescaped it would match every adm0_iso.
			name:         "places adm0_iso escapes ilike wildcards",
			query:        `query { agencies(where:{adm0_iso: "%"}) {onestop_id}}`,
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
	}
	c, _ := newTestClient(t)
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

// The WMATA feed is the only fixture with a station hierarchy: 125 served
// platforms hanging off 98 stations, plus entrances and generic nodes that no
// route serves.
func TestAgencyResolver_Stops(t *testing.T) {
	testcases := []testcase{
		{
			// Two agencies in one request must not share a limit.
			name:  "batched agencies keep separate limits",
			query: `query { agencies(where:{adm0_iso:"US"}) { onestop_id stops(limit:3) { stop_id } } }`,
			sel: []testcaseSelector{
				{selector: "agencies.#.onestop_id", expect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"}},
				{selector: "agencies.0.stops.#.stop_id", expectCount: 3},
				{selector: "agencies.3.stops.#.stop_id", expectCount: 3},
			},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

func TestAgencyResolver_Authz(t *testing.T) {
	cfg := testconfig.Config(t, testconfig.Options{
		FGAEndpoint:    testutil.FGAServer(t),
		FGAModelFile:   testdata.Path("server/authz/tls.json"),
		FGAModelTuples: fgaTestTuples,
	})

	srv := NewDefaultHandler()

	// Add config and perms middleware
	srv = model.AddConfigAndPerms(cfg, srv)

	testcases := []testcase{
		{
			name:         "basic",
			query:        `query { agencies {agency_id}}`,
			user:         "ian",
			selector:     "agencies.#.agency_id",
			selectExpect: []string{"caltrain-ca-us", "BART", "", "573", "a8b6ef46-7d4d-45f8-8200-cf4f5ce9d5a6", "1"},
		},
		{
			name:         "basic",
			query:        `query { agencies {agency_id}}`,
			user:         "public",
			selector:     "agencies.#.agency_id",
			selectExpect: []string{"caltrain-ca-us", "BART", "", "a8b6ef46-7d4d-45f8-8200-cf4f5ce9d5a6", "1"},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			c := client.New(usercheck.UserDefaultMiddleware(tc.user)(srv))
			queryTestcase(t, c, tc)
		})
	}
}
