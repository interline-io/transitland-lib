package copier

import (
	"fmt"
	"testing"

	tl "github.com/interline-io/transitland-lib"
)

func Benchmark_stopPatternKey(b *testing.B) {
	stoptimes := []tl.StopTime{}
	for i := 0; i < 50; i++ {
		stoptimes = append(stoptimes, tl.StopTime{StopID: fmt.Sprintf("%d", i*100)})
	}
	m := map[string]int{}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		key := stopPatternKey(stoptimes)
		m[key]++
	}
}
