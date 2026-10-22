package dbfinder

import (
	sq "github.com/irees/squirrel"

	"github.com/interline-io/transitland-lib/tlxy"
)

// geohashCellsExists returns an EXISTS clause matching feed_versions that have
// at least one stop geohash cell overlapping bbox, and ok=true. It returns
// ok=false (skip the filter) when bbox decomposes into zero cells or more than
// tlxy.GeohashBboxFilterMaxCells. fvCorrelation is the constant join predicate
// linking the subquery's feed_version_id to the outer query (e.g.
// sq.Expr("tl_feed_version_geohashes.feed_version_id = feed_versions.id")).
func geohashCellsExists(bbox tlxy.BoundingBox, fvCorrelation sq.Sqlizer) (sq.Sqlizer, bool) {
	cells := tlxy.GeohashFilterCells(bbox)
	if len(cells) == 0 {
		return nil, false
	}
	sub := sq.Select("1").
		From("tl_feed_version_geohashes").
		Where(fvCorrelation).
		Where(sq.Eq{"tl_feed_version_geohashes.geohash": cells})
	return sq.Expr("EXISTS (?)", sub), true
}
