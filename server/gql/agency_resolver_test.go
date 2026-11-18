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
