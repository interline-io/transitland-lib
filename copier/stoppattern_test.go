package copier

import (
	"fmt"
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

func Benchmark_stopPatternKey(b *testing.B) {
	stoptimes := []gtfs.StopTime{}
	for i := 0; i < 50; i++ {
		stoptimes = append(stoptimes, gtfs.StopTime{StopID: tt.NewString(fmt.Sprintf("%d", i*100))})
	}
	m := map[string]int{}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		key := stopPatternKey(stoptimes)
		m[key]++
	}
}
