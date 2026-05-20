package dbfinder

import (
	sq "github.com/irees/squirrel"

	"github.com/interline-io/transitland-lib/tlxy"
)

// geohashBboxFilterPrecision is the geohash precision used for the bbox/within/near
// secondary filter on tl_feed_version_geometries queries. p3 cells are ~156×156 km
// (square) and decompose typical city bboxes to ~1–4 cells; sufficient to reject
// the gross polygon false positives caused by bad-coordinate stops.
const geohashBboxFilterPrecision uint = 3

// geohashCellsExists returns an EXISTS clause matching feed_versions that have
// at least one stop geohash cell overlapping bbox. fvCorrelation is the
// constant join predicate linking the subquery's feed_version_id to the outer
// query (e.g. sq.Expr("tl_feed_version_geohashes.feed_version_id = feed_versions.id")).
func geohashCellsExists(bbox tlxy.BoundingBox, fvCorrelation sq.Sqlizer) sq.Sqlizer {
	cells := tlxy.CellsCoveringBbox(bbox, geohashBboxFilterPrecision)
	sub := sq.Select("1").
		From("tl_feed_version_geohashes").
		Where(fvCorrelation).
		Where(sq.Eq{"tl_feed_version_geohashes.geohash": cells})
	return sq.Expr("EXISTS (?)", sub)
}
