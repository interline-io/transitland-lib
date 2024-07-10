package tl

import (
	"testing"
)

func BenchmarkGetVersion(b *testing.B) {
	for n := 0; n < b.N; n++ {
		vi := getVersion()
		_ = vi
	}
}
