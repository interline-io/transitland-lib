package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
)

/////////////////////////

// ExpectError is an alias to testutil.ExpectError
type ExpectError = testutil.ExpectError

// PE wraps testutil.PE
func PE(errorStrings ...string) []ExpectError {
	return testutil.ParseExpectErrors(errorStrings...)
}

// CheckErrors wraps testutil.CheckErrors
func CheckErrors(expectedErrors []ExpectError, errs []error, t *testing.T) {
	testutil.CheckErrors(expectedErrors, errs, t)
}
