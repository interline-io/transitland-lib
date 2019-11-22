package gtcsv

import (
	"os"
	"strconv"

	"github.com/interline-io/gotransit"
)

var bufferSize = 1000
var chunkSize = 5000000

func init() {
	// Register readers/writers
	r := func(url string) (gotransit.Reader, error) { return NewReader(url) }
	gotransit.RegisterReader("csv", r)
	gotransit.RegisterReader("http", r)
	gotransit.RegisterReader("https", r)
	gotransit.RegisterReader("s3", r)
	w := func(url string) (gotransit.Writer, error) { return NewWriter(url) }
	gotransit.RegisterWriter("csv", w)
	// Set chunkSize from config.
	if v, e := strconv.Atoi(os.Getenv("GTFS_CHUNKSIZE")); e == nil {
		chunkSize = v
	}
}
