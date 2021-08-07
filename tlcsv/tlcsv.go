package tlcsv

import (
	"os"
	"strconv"

	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlrow"
)

type WriterAdapter = tlrow.WriterAdapter
type Row = tlrow.Row
type Adapter = tlrow.Adapter
type URLAdapter = tlrow.URLAdapter
type ZipAdapter = tlrow.ZipAdapter

var bufferSize = 1000
var chunkSize = 1000000

func init() {
	// Register readers/writers
	r := func(url string) (tl.Reader, error) { return NewReader(url) }
	ext.RegisterReader("csv", r)
	ext.RegisterReader("http", r)
	ext.RegisterReader("https", r)
	ext.RegisterReader("s3", r)
	ext.RegisterReader("overlay", r)
	ext.RegisterReader("ftp", r)
	w := func(url string) (tl.Writer, error) { return NewWriter(url) }
	ext.RegisterWriter("csv", w)
	// Set chunkSize from config.
	if v, e := strconv.Atoi(os.Getenv("TL_GTFS_CHUNKSIZE")); e == nil {
		chunkSize = v
	}
}
