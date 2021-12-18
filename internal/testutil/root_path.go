package testutil

import (
	"path/filepath"
	"runtime"
)

var (
	_, b, _, _ = runtime.Caller(0)
	basepath   = filepath.Dir(b)
)

// RootPath returns the project root directory, e.g. two directories up from internal/testutil.
func RootPath() string {
	a, err := filepath.Abs(filepath.Join(basepath, "..", ".."))
	if err != nil {
		return ""
	}
	return a
}

// RelPath returns the absolute path relative to the project root.
func RelPath(p string) string {
	return filepath.Join(RootPath(), p)
}
