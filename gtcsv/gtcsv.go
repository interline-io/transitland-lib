package gtcsv

import (
	"os"
	"strconv"

	"github.com/interline-io/transitland-lib/tl"
)

var bufferSize = 1000
var chunkSize = 1000000

func init() {
	// Register readers/writers
	r := func(url string) (tl.Reader, error) { return NewReader(url) }
	tl.RegisterReader("csv", r)
	tl.RegisterReader("http", r)
	tl.RegisterReader("https", r)
	tl.RegisterReader("s3", r)
	tl.RegisterReader("overlay", r)
	w := func(url string) (tl.Writer, error) { return NewWriter(url) }
	tl.RegisterWriter("csv", w)
	// Set chunkSize from config.
	if v, e := strconv.Atoi(os.Getenv("GTFS_CHUNKSIZE")); e == nil {
		chunkSize = v
	}
}
