package findertest

import (
	"os"
	"testing"

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
	Run(t, func(t testing.TB) model.Config {
		return testconfig.Config(t, testconfig.Options{})
	})
}
