package testutil

import (
	"github.com/interline-io/transitland-lib/internal/testpath"
)

// RelPath returns the absolute path relative to the project root.
func RelPath(p string) string {
	return testpath.RelPath(p)
}
