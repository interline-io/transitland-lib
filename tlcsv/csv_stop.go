package tlcsv

import "github.com/interline-io/transitland-lib/tl"

// csvStop helps load/write stops
type csvStop struct {
	tl.Stop
	StopLon float64
	StopLat float64
}
