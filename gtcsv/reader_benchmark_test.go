package gtcsv

import (
	"testing"

	"github.com/interline-io/gotransit/internal/testutil"
)

func BenchmarkReader(b *testing.B) {
	b.SetParallelism(1)
	for k, fe := range testutil.TestFeeds {
		b.Run(k, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				reader, err := NewReader(fe.URL)
				if err != nil {
					b.Error(err)
				}
				if err := reader.Open(); err != nil {
					b.Error(err)
				}
				defer reader.Close()
				testutil.CheckExpectEntities(b, fe, reader)
			}
		})
	}
}
