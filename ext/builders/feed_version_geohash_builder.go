package builders

import (
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/mmcloughlin/geohash"
)

// DefaultGeohashPrecisions are the geohash precisions computed for each feed
// version. p3 (tlxy.GeohashBboxFilterPrecision, ~156×156 km) backs the bbox
// discovery filter and must stay in this set — both sides reference the shared
// constant. p5 (~4.9×4.9 km, square) is the finer per-FV fingerprint precision
// for the FV-vs-FV comparison use case.
var DefaultGeohashPrecisions = []uint{tlxy.GeohashBboxFilterPrecision, 5}

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
// bounding box. Zero-count cells preserve flex feed visibility in
// feed/feed_version spatial queries without inflating density metrics. Stops
// with no coordinate at all are skipped, but explicit out-of-range and (0,0)
// coordinate values are kept, so the cells can also surface bad-coordinate
// stops.
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
		// Skip stops with no coordinate at all (Geometry.Valid is false); an
		// explicit (0,0) or out-of-range value is Valid and still recorded, so
		// genuinely bad coordinates remain discoverable.
		if !v.Geometry.Valid {
			return nil
		}
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
	// Stops contribute cells at every precision with stop_count >= 1 (p3 backs
	// discovery, p5 is the fingerprint).
	for _, s := range pp.stops {
		for _, p := range pp.precisions {
			pp.cells[geohash.EncodeWithPrecision(s.lat, s.lon, p)]++
		}
	}
	// Flex location polygons contribute stop_count=0 cells at every precision,
	// only where no stop already populated the cell. Zero-count cells keep flex
	// feeds visible in the bbox filter without inflating stop density; flex
	// zones are small enough in practice that the finer precisions stay cheap.
	// Conservative bbox-cover; finer polygon-cell intersection is a possible
	// future refinement.
	for _, loc := range pp.locations {
		bbox, ok := tlxy.BboxFromFlatCoords(loc.FlatCoords())
		if !ok {
			continue
		}
		for _, p := range pp.precisions {
			for _, cell := range tlxy.CellsCoveringBbox(bbox, p, 0) {
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
