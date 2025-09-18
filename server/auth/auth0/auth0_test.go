package auth0

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/testutil"
)

func TestAuth0Client(t *testing.T) {
	_, a, ok := testutil.CheckEnv("TL_TEST_AUTH0_DOMAIN")
	if !ok {
		t.Skip(a)
		return
	}
}
