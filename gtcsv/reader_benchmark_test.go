package gtcsv

import (
	"testing"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/internal/testutil"
)

func BenchmarkReader(b *testing.B) {
	b.SetParallelism(1)
	for k, fe := range testutil.ExternalTestFeeds {
		b.Run(k, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				testutil.TestReader(b, fe, func() gotransit.Reader {
					reader, err := NewReader(fe.URL)
					if err != nil {
						b.Error(err)
					}
					return reader
				})
			}
		})
	}
}
