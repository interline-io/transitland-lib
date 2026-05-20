package tlxy

import (
	"math"
	"sort"

	"github.com/mmcloughlin/geohash"
)

// GeohashBboxFilterPrecision is the geohash precision shared by both halves of
// the feed-version bbox filter: the builder stores per-feed-version stop cells
// at this precision, and queries expand a bbox into cells at the same precision
// to match against them. The two MUST agree — a stored set that omits this
// precision yields no matches — so both sides reference this constant instead
// of independent literals.
//
// p3 cells are ~156×156 km (square) and decompose typical city bboxes into ~1–4
// cells; coarse enough to reject the gross convex-hull false positives caused
// by bad-coordinate stops without dropping legitimate city-scale matches.
const GeohashBboxFilterPrecision uint = 3

// CellsCoveringBbox returns the sorted, deduplicated set of geohash cells at
// the given precision whose tiles intersect bbox.
//
// Bboxes crossing the antimeridian (MinLon > MaxLon) are not supported: the
// longitude loop never executes, so the whole bbox returns an empty result
// (not just the wrapped portion).
func CellsCoveringBbox(bbox BoundingBox, precision uint) []string {
	if precision == 0 {
		return nil
	}
	lonStep, latStep := geohashCellSize(precision)
	// Anchor iteration at the SW corner of the cell containing the bbox's SW corner,
	// then walk cell-by-cell sampling each cell's center.
	swCell := geohash.EncodeWithPrecision(bbox.MinLat, bbox.MinLon, precision)
	swBox := geohash.BoundingBox(swCell)
	latStart := swBox.MinLat + latStep/2
	lonStart := swBox.MinLng + lonStep/2

	cells := map[string]struct{}{}
	for lat := latStart; lat-latStep/2 <= bbox.MaxLat; lat += latStep {
		for lon := lonStart; lon-lonStep/2 <= bbox.MaxLon; lon += lonStep {
			cells[geohash.EncodeWithPrecision(lat, lon, precision)] = struct{}{}
		}
	}
	out := make([]string, 0, len(cells))
	for c := range cells {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// BboxFromFlatCoords computes a bounding box from a flat (lon, lat) coordinate
// slice. Returns ok=false if the slice contains no coordinate pairs.
func BboxFromFlatCoords(coords []float64) (BoundingBox, bool) {
	var bbox BoundingBox
	initialized := false
	for i := 0; i+1 < len(coords); i += 2 {
		lon, lat := coords[i], coords[i+1]
		if !initialized {
			bbox = BoundingBox{MinLon: lon, MaxLon: lon, MinLat: lat, MaxLat: lat}
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

// BboxFromPointRadius returns the smallest axis-aligned bounding box that
// encloses a circle of radius meters around (lon, lat). The longitude delta
// uses cos(lat) so the box widens at the equator and narrows near the poles,
// capped at full longitude coverage at very high latitudes. Latitude is clamped
// to [-90, 90] so a large radius near a pole cannot produce an out-of-range
// coordinate.
func BboxFromPointRadius(lon, lat, radiusMeters float64) BoundingBox {
	const metersPerDegLat = 111320.0
	latDelta := radiusMeters / metersPerDegLat
	cosLat := math.Cos(lat * math.Pi / 180.0)
	var lonDelta float64
	if cosLat < 1e-4 {
		lonDelta = 180.0
	} else {
		lonDelta = radiusMeters / (metersPerDegLat * cosLat)
		if lonDelta > 180.0 {
			lonDelta = 180.0
		}
	}
	return BoundingBox{
		MinLon: lon - lonDelta,
		MinLat: math.Max(lat-latDelta, -90.0),
		MaxLon: lon + lonDelta,
		MaxLat: math.Min(lat+latDelta, 90.0),
	}
}

// geohashCellSize returns the (lon, lat) cell dimensions in degrees at the
// given precision. Each character contributes 5 bits, allocated alternately
// starting with longitude: lonBits = ceil(5N/2), latBits = floor(5N/2).
func geohashCellSize(precision uint) (lonStep, latStep float64) {
	bits := 5 * int(precision)
	lonBits := (bits + 1) / 2
	latBits := bits / 2
	lonStep = 360.0 / float64(int(1)<<lonBits)
	latStep = 180.0 / float64(int(1)<<latBits)
	return
}
