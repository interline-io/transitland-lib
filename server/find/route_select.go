package find

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/jmoiron/sqlx"
)

func FindRoutes(atx sqlx.Ext, limit *int, after *int, ids []int, where *model.RouteFilter) (ents []*model.Route, err error) {
	q := RouteSelect(limit, after, ids, where)
	if len(ids) == 0 && (where == nil || where.FeedVersionSha1 == nil) {
		q = q.Where(sq.NotEq{"active": nil})
	}
	MustSelect(model.DB, q, &ents)
	return ents, nil
}

func RouteSelect(limit *int, after *int, ids []int, where *model.RouteFilter) sq.SelectBuilder {
	qView := sq.StatementBuilder.Select(
		"gtfs_routes.*",
		"g.geometry",
		"g.centroid AS geometry_centroid",
		"g.generated AS geometry_generated",
		"current_feeds.id AS feed_id",
		"current_feeds.onestop_id AS feed_onestop_id",
		"feed_versions.sha1 AS feed_version_sha1",
		"tl_agency_onestop_ids.onestop_id AS operator_onestop_id",
		"tl_route_onestop_ids.onestop_id",
		"feed_states.feed_version_id AS active",
		"rh.headway_seconds_weekday_morning",
	).
		From("gtfs_routes").
		Join("feed_versions ON feed_versions.id = gtfs_routes.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id").
		JoinClause("LEFT JOIN tl_route_onestop_ids ON tl_route_onestop_ids.route_id = gtfs_routes.id").
		JoinClause("LEFT JOIN tl_agency_onestop_ids ON tl_agency_onestop_ids.agency_id = gtfs_routes.agency_id").
		JoinClause("LEFT JOIN feed_states ON feed_states.feed_version_id = gtfs_routes.feed_version_id").
		JoinClause(`LEFT JOIN LATERAL ( SELECT tl_route_geometries.route_id,
            tl_route_geometries.feed_version_id,
            tl_route_geometries.shape_id,
            tl_route_geometries.direction_id,
            tl_route_geometries.generated,
            tl_route_geometries.geometry,
            tl_route_geometries.centroid
           FROM tl_route_geometries
          WHERE tl_route_geometries.route_id = gtfs_routes.id
          ORDER BY tl_route_geometries.route_id, tl_route_geometries.direction_id
         LIMIT 1) g ON true`).
		JoinClause(`LEFT JOIN LATERAL ( SELECT rh_1.headway_seconds_morning_mid AS headway_seconds_weekday_morning
           FROM tl_route_headways rh_1
          WHERE rh_1.dow_category = 1 AND rh_1.route_id = gtfs_routes.id) rh ON true`).
		Where(sq.Eq{"current_feeds.deleted_at": nil})

	q := sq.StatementBuilder.Select("*").FromSelect(qView, "t")
	if len(ids) > 0 {
		q = q.Where(sq.Eq{"t.id": ids})
	}
	if after != nil {
		q = q.Where(sq.Gt{"t.id": *after})
	}
	q = q.Limit(checkLimit(limit))
	if where != nil {
		if where.Search != nil && len(*where.Search) > 0 {
			rank, wc := tsQuery(*where.Search)
			q = q.Column(rank).Where(wc)
		}
		if len(where.AgencyIds) > 0 {
			q = q.Where(sq.Eq{"agency_id": where.AgencyIds})
		}
		if where.FeedVersionSha1 != nil {
			q = q.Where(sq.Eq{"feed_version_sha1": *where.FeedVersionSha1})
		}
		if where.FeedOnestopID != nil {
			q = q.Where(sq.Eq{"feed_onestop_id": *where.FeedOnestopID})
		}
		if where.RouteID != nil {
			q = q.Where(sq.Eq{"route_id": *where.RouteID})
		}
		if where.OnestopID != nil {
			q = q.Where(sq.Eq{"onestop_id": *where.OnestopID})
		}
		if where.OperatorOnestopID != nil {
			q = q.Where(sq.Eq{"operator_onestop_id": *where.OperatorOnestopID})
		}
		if where.RouteType != nil {
			q = q.Where(sq.Eq{"route_type": where.RouteType})
		}
		if where.Within != nil && where.Within.Valid {
			q = q.Where("ST_Intersects(t.geometry, ?)", where.Within)
		}
		if where.Near != nil {
			q = q.Where("ST_DWithin(t.geometry, ST_MakePoint(?,?), ?)", where.Near.Lat, where.Near.Lon, where.Near.Radius)
		}
	}
	return q
}

func RouteStopBufferSelect(param model.RouteStopBufferParam) sq.SelectBuilder {
	r := checkFloat(param.Radius, 0, 2000.0)
	q := sq.StatementBuilder.
		Select(
			"ST_Collect(gtfs_stops.geometry::geometry)::geography AS stop_points",
			"ST_ConvexHull(ST_Collect(gtfs_stops.geometry::geometry))::geography AS stop_convexhull",
		).
		Column(sq.Expr("ST_Buffer(ST_Collect(gtfs_stops.geometry::geometry)::geography, ?, 4)::geography AS stop_buffer", r)). // column expr
		From("gtfs_stops").
		InnerJoin("tl_route_stops on tl_route_stops.stop_id = gtfs_stops.id").
		Where(sq.Eq{"tl_route_stops.route_id": param.EntityID})
	return q
}
