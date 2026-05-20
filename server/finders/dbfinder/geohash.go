package dbfinder

import (
	"fmt"

	sq "github.com/irees/squirrel"

	"github.com/interline-io/transitland-lib/tlxy"
)

// geohashBboxFilterPrecision is the geohash precision used for the bbox/within/near
// secondary filter on tl_feed_version_geometries queries. p3 cells are ~156×156 km
// (square) and decompose typical city bboxes to ~1–4 cells; sufficient to reject
// the gross polygon false positives caused by bad-coordinate stops.
const geohashBboxFilterPrecision uint = 3

// geohashCellsExists returns an EXISTS clause matching feed_versions that have
// at least one stop geohash cell overlapping bbox. fvIDColumn is the SQL
// expression identifying the feed_version_id in the outer query (e.g.
// "feed_versions.id" or "fs_geom.feed_version_id").
func geohashCellsExists(bbox tlxy.BoundingBox, fvIDColumn string) sq.Sqlizer {
	cells := tlxy.CellsCoveringBbox(bbox, geohashBboxFilterPrecision)
	return sq.Expr(fmt.Sprintf(`EXISTS (SELECT 1 FROM tl_feed_version_geohashes
                                    WHERE feed_version_id = %s
                                      AND geohash = ANY(?))`, fvIDColumn), cells)
}
