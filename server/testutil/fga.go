package testutil

import (
	"testing"
)

// FGAServer returns the OpenFGA endpoint in TL_TEST_FGA_ENDPOINT, skipping
// the test when it is unset.
func FGAServer(t testing.TB) string {
	t.Helper()
	endpoint, reason, ok := CheckEnv("TL_TEST_FGA_ENDPOINT")
	if !ok {
		t.Skip(reason)
	}
	return endpoint
}
