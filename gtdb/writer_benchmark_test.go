package gtdb

import (
	"testing"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/gtcsv"
	"github.com/interline-io/gotransit/internal/testutil"
)

func BenchmarkWriter(b *testing.B) {
	wtests := map[string]func() gotransit.Writer{
		// "SpatiaLite": func() gotransit.Writer {
		// 	writer, _ := NewWriter("sqlite3://:memory:")
		// 	return writer
		// },
		"Postgres": func() gotransit.Writer {
			writer, _ := NewWriter("postgres://localhost/tl?sslmode=disable")
			return writer
		},
	}
	for k, wfunc := range wtests {
		b.Run(k, func(b *testing.B) {
			for k, fe := range testutil.ExternalTestFeeds {
				rfunc := func() gotransit.Reader {
					reader, _ := gtcsv.NewReader(fe.URL)
					return reader
				}
				b.Run(k, func(b *testing.B) {
					testutil.BenchmarkWriter(b, fe, rfunc, wfunc)
				})
			}
		})
	}
}
