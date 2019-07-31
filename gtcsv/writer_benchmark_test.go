package gtcsv

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/interline-io/gotransit/internal/testutil"
)

func BenchmarkWriter(b *testing.B) {
	b.SetParallelism(1)
	for k, fe := range testutil.ExternalTestFeeds {
		b.Run(k, func(b *testing.B) {
			reader, err := NewReader(fe.URL)
			if err != nil {
				b.Error(err)
			}
			if err := reader.Open(); err != nil {
				b.Error(err)
			}
			defer reader.Close()
			for i := 0; i < b.N; i++ {
				tmpdir, err := ioutil.TempDir("", "gtfs")
				if err != nil {
					b.Error(err)
					return
				}
				writer, err := NewWriter(tmpdir)
				if err != nil {
					b.Error(err)
				}
				if err := writer.Open(); err != nil {
					b.Error(err)
				}
				if err := testutil.DirectCopy(reader, writer); err != nil {
					b.Error(err)
				}
				r2, err := writer.NewReader()
				if err != nil {
					b.Error(err)
				}
				fe.Benchmark(b, r2)
				if err := writer.Close(); err != nil {
					b.Error(err)
				}
				if err := os.RemoveAll(tmpdir); err != nil {
					b.Error(err)
				}
			}
		})
	}
}
