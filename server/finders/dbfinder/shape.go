package dbfinder

import (
	"context"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	sq "github.com/irees/squirrel"
)

// FindShapesByFeedVersion pages a feed version's shapes by integer-id cursor.
// Direct query, not a batched loader, so the per-parent `after` cursor works.
func (f *Finder) FindShapesByFeedVersion(ctx context.Context, fvid int, limit *int, after *model.Cursor, where *model.ShapeFilter) ([]*model.Shape, error) {
	var ents []*model.Shape
	if err := dbutil.Select(ctx, f.db, shapeSelect(limit, after, fvid, f.PermFilter(ctx), where), &ents); err != nil {
		return nil, logErr(ctx, err)
	}
	return ents, nil
}

func shapeSelect(limit *int, after *model.Cursor, fvid int, permFilter *model.PermFilter, where *model.ShapeFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.
		Select(
			"gtfs_shapes.id",
			"gtfs_shapes.feed_version_id",
			"gtfs_shapes.shape_id",
			"gtfs_shapes.geometry",
			"gtfs_shapes.generated",
		).
		From("gtfs_shapes").
		Join("feed_versions on feed_versions.id = gtfs_shapes.feed_version_id").
		Join("current_feeds on current_feeds.id = feed_versions.feed_id").
		Join(joinImportedShapes).
		Where(sq.Eq{"gtfs_shapes.feed_version_id": fvid}).
		OrderBy("gtfs_shapes.id asc").
		Limit(finderCheckLimit(limit))
	if after != nil && after.Valid && after.ID > 0 {
		q = q.Where(sq.Gt{"gtfs_shapes.id": after.ID})
	}
	if where != nil {
		if len(where.Ids) > 0 {
			q = q.Where(In("gtfs_shapes.id", where.Ids))
		}
		if where.ShapeID != nil {
			q = q.Where(sq.Eq{"gtfs_shapes.shape_id": *where.ShapeID})
		}
		if where.RouteType != nil {
			// Shapes used by a trip on a route of this route_type (trip->route join).
			q = q.Where(sq.Expr(
				"exists (select 1 from gtfs_trips inner join gtfs_routes on gtfs_routes.id = gtfs_trips.route_id where gtfs_trips.shape_id = gtfs_shapes.id and gtfs_routes.route_type = ?)",
				*where.RouteType,
			))
		}
	}
	q = pfJoinCheckFv(q, permFilter)
	return q
}
