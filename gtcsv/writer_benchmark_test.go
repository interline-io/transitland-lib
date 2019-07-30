package gtcsv

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/interline-io/gotransit/internal/testutil"
)

func BenchmarkWriter(b *testing.B) {
	tests := []string{"bart.zip"}
	b.SetParallelism(1)
	for _, k := range tests {
		fe, ok := testutil.ExternalTestFeeds[k]
		if !ok {
			b.Error("no such test feed:", k)
			continue
		}
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
				//
				tmpdir, err := ioutil.TempDir("", "gtfs")
				if err != nil {
					b.Error(err)
					return
				}
				defer os.RemoveAll(tmpdir)
				writer, err := NewWriter(tmpdir)
				if err != nil {
					b.Error(err)
					return
				}
				writer.Open()
				defer writer.Close()
				if err := testutil.DirectCopy(reader, writer); err != nil {
					b.Error(err)
				}
				r2, err := writer.NewReader()
				if err != nil {
					b.Error(err)
				}
				fe.Benchmark(b, r2)
			}
		})
	}
}
