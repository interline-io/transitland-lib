package gql

import (
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/gql/internal/gqlcase"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/testdata"
)

func TestAgencyResolver(t *testing.T) {
	vars := hw{"agency_id": "caltrain-ca-us"}
	testcases := []testcase{
		// places should test filters because it's not a root resolver
		{
			Name:         "places",
			Query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {places {adm0_name adm0_iso adm1_name adm1_iso city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.0.places.#.city_name",
			SelectExpect: []string{"San Mateo", "San Francisco", "San Jose", ""},
		},
		{
			Name:         "places get adm0_iso",
			Query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {places {adm0_name adm0_iso adm1_name adm1_iso city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.0.places.#.adm0_iso",
			SelectExpect: []string{"US", "US", "US", "US"},
		},
		{
			Name:         "places get adm1_iso",
			Query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {places {adm0_name adm0_iso adm1_name adm1_iso city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.0.places.#.adm1_iso",
			SelectExpect: []string{"US-CA", "US-CA", "US-CA", "US-CA"},
		},
		{
			Name:         "places rank 0.25",
			Query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {places(where:{min_rank:0.25}) {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.0.places.#.city_name",
			SelectExpect: []string{"San Mateo", "San Jose", ""},
		},
		{
			Name:         "places rank 0.75",
			Query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {places(where:{min_rank:0.75}) {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.0.places.#.adm1_name",
			SelectExpect: []string{"California"},
		},
		// place iso codes
		{
			Name:         "places iso3166 country",
			Query:        `query { agencies(where:{adm0_iso: "US"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"},
		},
		{
			Name:         "places iso3166 state",
			Query:        `query { agencies(where:{adm1_iso: "US-CA"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			Name:         "places iso3166 state lowercase",
			Query:        `query { agencies(where:{adm1_iso: "us-ca"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			Name:         "places iso3166 state and country",
			Query:        `query { agencies(where:{adm0_iso: "us", adm1_iso: "us-ca"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			Name:         "places iso3166 state and city",
			Query:        `query { agencies(where:{city_name: "oakland", adm1_iso: "us-ca"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			Name:         "places iso3166 state and city no result",
			Query:        `query { agencies(where:{city_name: "test", adm1_iso: "us-ca"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:         "places iso3166 state no results",
			Query:        `query { agencies(where:{adm1_iso: "US-NY"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:         "places state",
			Query:        `query { agencies(where:{adm1_name: "California"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			Name:         "places state no result",
			Query:        `query { agencies(where:{adm1_name: "New York"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			// `%` is an ILIKE wildcard; unescaped it would match every adm0_iso.
			Name:         "places adm0_iso escapes ilike wildcards",
			Query:        `query { agencies(where:{adm0_iso: "%"}) {onestop_id}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:         "places city",
			Query:        `query { agencies(where:{city_name: "Berkeley"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			Name:         "places city 2",
			Query:        `query { agencies(where:{city_name: "San Jose"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{"o-9q9-caltrain"},
		},
		{
			Name:         "places city 2 lowercase",
			Query:        `query { agencies(where:{city_name: "san jose"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{"o-9q9-caltrain"},
		},
	}
	c, _ := newTestClient(t)
	gqlcase.QueryTestcases(t, c, testcases)
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

func TestAgencyResolver_Stops(t *testing.T) {
	testcases := []testcase{
		{
			// Two agencies in one request must not share a limit.
			Name:  "batched agencies keep separate limits",
			Query: `query { agencies(where:{adm0_iso:"US"}) { onestop_id stops(limit:3) { stop_id } } }`,
			Sel: []testcaseSelector{
				{Selector: "agencies.#.onestop_id", Expect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"}},
				{Selector: "agencies.0.stops.#.stop_id", ExpectCount: 3},
				{Selector: "agencies.3.stops.#.stop_id", ExpectCount: 3},
			},
		},
	}
	c, _ := newTestClient(t)
	gqlcase.QueryTestcases(t, c, testcases)
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
			Name:         "basic",
			Query:        `query { agencies {agency_id}}`,
			User:         "ian",
			Selector:     "agencies.#.agency_id",
			SelectExpect: []string{"caltrain-ca-us", "BART", "", "573", "a8b6ef46-7d4d-45f8-8200-cf4f5ce9d5a6", "1"},
		},
		{
			Name:         "basic",
			Query:        `query { agencies {agency_id}}`,
			User:         "public",
			Selector:     "agencies.#.agency_id",
			SelectExpect: []string{"caltrain-ca-us", "BART", "", "a8b6ef46-7d4d-45f8-8200-cf4f5ce9d5a6", "1"},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			c := client.New(usercheck.UserDefaultMiddleware(tc.User)(srv))
			gqlcase.QueryTestcase(t, c, tc)
		})
	}
}
