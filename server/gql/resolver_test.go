package gql

import (
	"os"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/gql/internal/gqlcase"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/stretchr/testify/assert"
)

const DEFAULT_WHEN = "2022-09-01T00:00:00Z"

type hw = map[string]interface{}

type testcaseSelector = gqlcase.TestcaseSelector

type testcase = gqlcase.Testcase

func TestMain(m *testing.M) {
	// Increase default limit for testing purposes
	RESOLVER_MAXLIMIT = 100_000
	if a, ok := testutil.CheckTestDB(); !ok {
		log.Print(a)
		return
	}
	os.Exit(m.Run())
}

// Test helpers

func newTestClient(t testing.TB) (*client.Client, model.Config) {
	return newTestClientWithOpts(t, testconfig.Options{
		RTJsons: testconfig.DefaultRTJson(),
	})
}

func newTestClientWithOpts(t testing.TB, opts testconfig.Options) (*client.Client, model.Config) {
	if opts.WhenUtc == "" {
		opts.WhenUtc = DEFAULT_WHEN
	}
	cfg := testconfig.Config(t, opts)
	srv := NewDefaultHandler()
	graphqlServer := model.AddConfigAndPerms(cfg, srv)
	srvMiddleware := usercheck.NewUserDefaultMiddleware(func() authn.User {
		return authn.NewCtxUser("testuser", "", "").WithRoles("testrole")
	})
	return client.New(srvMiddleware(graphqlServer)), cfg
}

func Test_checkBbox(t *testing.T) {
	tcs := []struct {
		name   string
		bbox   model.BoundingBox
		area   float64
		expect bool
	}{
		{
			name:   "small 1",
			bbox:   model.BoundingBox{MinLon: -122.27023950636408, MinLat: 37.80659174954722, MaxLon: -122.26747210439623, MaxLat: 37.80928736232528},
			area:   0.08 * 1e6,
			expect: true,
		},
		{
			name:   "small 2",
			bbox:   model.BoundingBox{MinLon: -122.27023950636408, MinLat: 37.80659174954722, MaxLon: -122.26747210439623, MaxLat: 37.80928736232528},
			area:   0.07 * 1e6,
			expect: false,
		},
		{
			name:   "big 1",
			bbox:   model.BoundingBox{MinLon: -123.08182748515924, MinLat: 37.100203650623826, MaxLon: -121.31739022765265, MaxLat: 38.31972646345332},
			area:   22000.0 * 1e6,
			expect: true,
		},
		{
			name:   "big 2",
			bbox:   model.BoundingBox{MinLon: -123.08182748515924, MinLat: 37.100203650623826, MaxLon: -121.31739022765265, MaxLat: 38.31972646345332},
			area:   20000.0 * 1e6,
			expect: false,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			v := checkBbox(&tc.bbox, tc.area)
			assert.Equal(t, tc.expect, v, "expected result")

		})
	}
}
