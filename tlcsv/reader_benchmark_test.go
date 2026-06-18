package tlcsv

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testreader"
)

func BenchmarkReader(b *testing.B) {
	b.SetParallelism(1)
	for k, fe := range testreader.ExternalTestFeeds {
		b.Run(k, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				reader, err := NewReader(fe.URL)
				if err != nil {
					b.Error(err)
				}
				if err := reader.Open(); err != nil {
					b.Error(err)
				}
				testreader.CheckReader(b, fe, reader)
				if err := reader.Close(); err != nil {
					b.Error(err)
				}
			}
		})
	}
}
