package tlcsv

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl"
)

func BenchmarkWriter(b *testing.B) {
	b.SetParallelism(1)
	for k, fe := range testutil.ExternalTestFeeds {
		b.Run(k, func(b *testing.B) {
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
				testutil.TestWriter(b, fe, func() tl.Reader {
					a, err := NewReader(fe.URL)
					if err != nil {
						b.Error(err)
					}
					return a
				}, func() tl.Writer {
					return writer
				})
				// Clean up and double check
				if err := os.RemoveAll(tmpdir); err != nil {
					b.Error(err)
				}
				if _, err := os.Stat(tmpdir); !os.IsNotExist(err) {
					b.Error("did not remove temporary directory!", tmpdir)
				}
			}
		})
	}
}
