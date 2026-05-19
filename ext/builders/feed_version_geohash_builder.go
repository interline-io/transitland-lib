package builders

import (
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/mmcloughlin/geohash"
)

// DefaultGeohashPrecisions are the geohash precisions computed for each feed
// version: p3 (~156×156 km, square — used for bbox discovery filtering) and
// p5 (~4.9×4.9 km, square — reserved for future fingerprint/comparison use).
var DefaultGeohashPrecisions = []uint{3, 5}

type FeedVersionGeohash struct {
	Geohash   tt.String
	StopCount tt.Int
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
// polygons during the copier pass and emits per-(precision, geohash)
// FeedVersionGeohash entities in Copy(). Designed to be registered with the
// stats copier (fetch / rebuild-stats path); at fetch time the copier's writer
// discards entities, so callers should read computed cells via Cells() and
// persist them directly.
//
// Each emitted FeedVersionGeohash row carries a stop_count: positive when the
// cell contains one or more fixed stops, zero when the cell is reached only
// via a flex location polygon's bounding box. Zero-count cells preserve flex
// feed visibility in bbox_stops queries without inflating density metrics.
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

func (pp *FeedVersionGeohashBuilder) Copy(copier adapters.EntityCopier) error {
	// Stops contribute cells with stop_count >= 1.
	for _, s := range pp.stops {
		if !tlxy.IsValidStopCoord(s.lon, s.lat) {
			continue
		}
		for _, p := range pp.precisions {
			pp.cells[geohash.EncodeWithPrecision(s.lat, s.lon, p)]++
		}
	}
	// Flex location polygons contribute cells with stop_count=0, only for
	// cells not already populated by a stop. Conservative bbox-cover; finer
	// polygon-cell intersection is a possible future refinement.
	for _, loc := range pp.locations {
		bbox, ok := flatCoordsBbox(loc.FlatCoords())
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
	for cell, count := range pp.cells {
		ent := FeedVersionGeohash{
			Geohash:   tt.NewString(cell),
			StopCount: tt.NewInt(count),
		}
		if err := copier.CopyEntity(&ent); err != nil {
			return err
		}
	}
	return nil
}

// flatCoordsBbox computes a bounding box from a flat (lon,lat) coordinate
// slice, ignoring vertices that fail IsValidStopCoord. Returns ok=false if
// no valid coordinates remain.
func flatCoordsBbox(coords []float64) (tlxy.BoundingBox, bool) {
	var bbox tlxy.BoundingBox
	initialized := false
	for i := 0; i+1 < len(coords); i += 2 {
		lon, lat := coords[i], coords[i+1]
		if !tlxy.IsValidStopCoord(lon, lat) {
			continue
		}
		if !initialized {
			bbox = tlxy.BoundingBox{MinLon: lon, MaxLon: lon, MinLat: lat, MaxLat: lat}
			initialized = true
			continue
		}
		if lon < bbox.MinLon {
			bbox.MinLon = lon
		}
		if lon > bbox.MaxLon {
			bbox.MaxLon = lon
		}
		if lat < bbox.MinLat {
			bbox.MinLat = lat
		}
		if lat > bbox.MaxLat {
			bbox.MaxLat = lat
		}
	}
	return bbox, initialized
}

// Cells returns the accumulated geohash→stop_count map after Copy() has run.
// Used by callers (e.g. static_fetch) where the copier's writer is empty.
func (pp *FeedVersionGeohashBuilder) Cells() map[string]int {
	return pp.cells
}
