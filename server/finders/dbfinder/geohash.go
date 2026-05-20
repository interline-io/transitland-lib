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

// geohashBboxFilterMaxCells caps the number of cells the secondary filter will
// expand a bbox into. Country/continent-scale bboxes blow past this; at that
// size the filter adds no selectivity (it matches nearly everything) and only
// bloats the IN list, so we skip it and rely on the primary ST_* predicate.
// 1000 p3 cells is roughly continent scale (the contiguous US is ~760); we may
// lower this in the future if large-bbox IN lists prove costly in practice.
const geohashBboxFilterMaxCells = 1000

// geohashCellsExists returns an EXISTS clause matching feed_versions that have
// at least one stop geohash cell overlapping bbox, and ok=true. It returns
// ok=false (skip the filter) when bbox decomposes into zero cells or more than
// geohashBboxFilterMaxCells. fvCorrelation is the constant join predicate
// linking the subquery's feed_version_id to the outer query (e.g.
// sq.Expr("tl_feed_version_geohashes.feed_version_id = feed_versions.id")).
func geohashCellsExists(bbox tlxy.BoundingBox, fvCorrelation sq.Sqlizer) (sq.Sqlizer, bool) {
	cells := tlxy.CellsCoveringBbox(bbox, geohashBboxFilterPrecision)
	if len(cells) == 0 || len(cells) > geohashBboxFilterMaxCells {
		return nil, false
	}
	sub := sq.Select("1").
		From("tl_feed_version_geohashes").
		Where(fvCorrelation).
		Where(sq.Eq{"tl_feed_version_geohashes.geohash": cells})
	return sq.Expr("EXISTS (?)", sub), true
}
