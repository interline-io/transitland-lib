// Package tlcsv provides adapters to read and write GTFS from CSV format files.
package tlcsv

import (
	"os"
	"strconv"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/ext"
)

var bufferSize = 1000

// groupBufferSize is the read-ahead for the grouped streams (ShapesByShapeID,
// StopTimesByTripID, TripsWithStopTimes), where each buffered item is a whole shape's
// points or a trip's stop_times. Kept small so the in-flight set isn't ~1000 full
// geometries while the writer drains a batch.
var groupBufferSize = 32

var chunkSize = 250000

func init() {
	// Register readers/writers
	r := func(url string) (adapters.Reader, error) { return NewReader(url) }
	ext.RegisterReader("csv", r)
	ext.RegisterReader("http", r)
	ext.RegisterReader("https", r)
	ext.RegisterReader("s3", r)
	ext.RegisterReader("overlay", r)
	ext.RegisterReader("ftp", r)
	w := func(url string) (adapters.Writer, error) { return NewWriter(url) }
	ext.RegisterWriter("csv", w)
	// Set chunkSize from config.
	if v, e := strconv.Atoi(os.Getenv("TL_GTFS_CHUNKSIZE")); e == nil {
		chunkSize = v
	}
}
