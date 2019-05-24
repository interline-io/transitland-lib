package copier

import (
	"fmt"
	"testing"

	"github.com/interline-io/gotransit"
)

func Benchmark_stopPatternKey(b *testing.B) {
	stoptimes := []gotransit.StopTime{}
	for i := 0; i < 50; i++ {
		stoptimes = append(stoptimes, gotransit.StopTime{StopID: fmt.Sprintf("%d", i*100)})
	}
	m := map[string]int{}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		key := stopPatternKey(stoptimes)
		m[key]++
	}
}
