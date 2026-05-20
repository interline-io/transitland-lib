package builders

import (
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/mmcloughlin/geohash"
)

// DefaultGeohashPrecisions are the geohash precisions computed for each feed
// version. Currently only p3 (~156×156 km, square) is computed; it is used for
// the bbox discovery filter. p5 (~4.9×4.9 km, square) is left out for now to
// avoid storing cells nothing reads yet — add 5 back here to re-enable it for
// future fingerprint/comparison use.
var DefaultGeohashPrecisions = []uint{3} // {3, 5}

type FeedVersionGeohash struct {
	Geohash   tt.String
	StopCount int
	tt.MinEntity
	tt.FeedVersionEntity
}

func (ent *FeedVersionGeohash) Filename() string {
	return "tl_feed_version_geohashes.txt"
}

func (ent *FeedVersionGeohash) TableName() string {
	return "tl_feed_version_geohashes"
}

// FeedVersionGeohashBuilder accumulates stop locations and GTFS-Flex location
// polygons during the copier pass and computes a per-(precision, geohash)
// stop-count map in Copy(). Designed to be registered with the stats copier
// (fetch / rebuild-stats path); callers read the result via Cells().
//
// Each cell carries a stop_count: positive when the cell contains one or more
// stops, zero when the cell is reached only via a flex location polygon's
// bounding box. Zero-count cells preserve flex feed visibility in bbox_stops
// queries without inflating density metrics. All stop coordinates are kept,
// including out-of-range and (0,0) values, so the cells can also surface
// bad-coordinate stops.
type FeedVersionGeohashBuilder struct {
	precisions []uint
	stops      map[string]*stopGeom
	locations  []tt.Geometry
	cells      map[string]int
}

func NewFeedVersionGeohashBuilder() *FeedVersionGeohashBuilder {
	return &FeedVersionGeohashBuilder{
		precisions: DefaultGeohashPrecisions,
		stops:      map[string]*stopGeom{},
		cells:      map[string]int{},
	}
}

func (pp *FeedVersionGeohashBuilder) AfterWrite(eid string, ent tt.Entity, emap *tt.EntityMap) error {
	switch v := ent.(type) {
	case *gtfs.Stop:
		pp.stops[eid] = &stopGeom{
			lon: v.Geometry.X(),
			lat: v.Geometry.Y(),
		}
	case *gtfs.Location:
		pp.locations = append(pp.locations, v.Geometry)
	}
	return nil
}

func (pp *FeedVersionGeohashBuilder) Copy(_ adapters.EntityCopier) error {
	// Stops contribute cells with stop_count >= 1.
	for _, s := range pp.stops {
		for _, p := range pp.precisions {
			pp.cells[geohash.EncodeWithPrecision(s.lat, s.lon, p)]++
		}
	}
	// Flex location polygons contribute cells with stop_count=0, only for
	// cells not already populated by a stop. Conservative bbox-cover; finer
	// polygon-cell intersection is a possible future refinement.
	for _, loc := range pp.locations {
		bbox, ok := tlxy.BboxFromFlatCoords(loc.FlatCoords())
		if !ok {
			continue
		}
		for _, p := range pp.precisions {
			for _, cell := range tlxy.CellsCoveringBbox(bbox, p) {
				if _, exists := pp.cells[cell]; !exists {
					pp.cells[cell] = 0
				}
			}
		}
	}
	return nil
}

// Cells returns the accumulated geohash→stop_count map after Copy() has run.
func (pp *FeedVersionGeohashBuilder) Cells() map[string]int {
	return pp.cells
}
