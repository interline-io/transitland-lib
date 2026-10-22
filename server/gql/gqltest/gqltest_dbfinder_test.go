package gqltest

import (
	"os"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/gql"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
)

func TestMain(m *testing.M) {
	gql.RESOLVER_MAXLIMIT = 100_000
	if a, ok := testutil.CheckTestDB(); !ok {
		log.Print(a)
		return
	}
	os.Exit(m.Run())
}

func TestDBFinder(t *testing.T) {
	Run(t, dbfinderConfig(testconfig.Options{}), Spatial)
}

// UseMaterializedDepartures plans departure queries from a materialized CTE, and
// the results must match those without it.
func TestDBFinder_MaterializedDepartures(t *testing.T) {
	Run(t, dbfinderConfig(testconfig.Options{UseMaterializedDepartures: true}), Spatial)
}

func BenchmarkDBFinder_StopResolver(b *testing.B) {
	c, cfg := newTestClient(b, dbfinderConfig(testconfig.Options{}), "", nil)
	benchmarkTestcases(b, c, stopResolverTestcases(b, cfg))
}

// dbfinderConfig returns the test database's config with these options.
func dbfinderConfig(opts testconfig.Options) NewConfig {
	return func(t testing.TB) model.Config {
		return testconfig.Config(t, opts)
	}
}

func benchmarkTestcases(b *testing.B, c *client.Client, tcs []testcase) {
	for _, tc := range tcs {
		b.Run(tc.name, func(b *testing.B) {
			benchmarkTestcase(b, c, tc)
		})
	}
}

func benchmarkTestcase(b *testing.B, c *client.Client, tc testcase) {
	opts := []client.Option{}
	for k, v := range tc.vars {
		opts = append(opts, client.Var(k, v))
	}
	var resp map[string]any
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		c.MustPost(tc.query, &resp, opts...)
	}
}
