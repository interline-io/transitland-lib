package tlpb

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
)

func TestReadStopsPB(t *testing.T) {
	ReadStopsPB(testutil.RelPath("test/data/external/bart.zip"))
}

func BenchmarkReadStopsPB(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ReadStopsPB(testutil.RelPath("test/data/external/bart.zip"))
	}
}

func BenchmarkReadStopsTT(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ReadStopsTT(testutil.RelPath("test/data/external/bart.zip"))
	}
}
