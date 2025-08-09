package testdata

import (
	"path/filepath"
	"runtime"
)

// Path returns this directory plus provided path
func Path(p ...string) string {
	_, b, _, _ := runtime.Caller(0)
	dataPath, err := filepath.Abs(filepath.Join(filepath.Dir(b)))
	if err != nil {
		return ""
	}
	a := []string{dataPath}
	a = append(a, p...)
	return filepath.Join(a...)
}
