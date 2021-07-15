package find

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/jmoiron/sqlx"
)

func FindFeedVersions(atx sqlx.Ext, limit *int, after *int, ids []int, where *model.FeedVersionFilter) (ents []*model.FeedVersion, err error) {
	MustSelect(model.DB, FeedVersionSelect(limit, after, ids, where), &ents)
	return ents, nil
}

func FeedVersionSelect(limit *int, after *int, ids []int, where *model.FeedVersionFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.
		Select("t.*, tl_feed_version_geometries.geometry").
		Join("current_feeds cf on cf.id = t.feed_id").Where(sq.Eq{"cf.deleted_at": nil}).
		JoinClause("left join tl_feed_version_geometries on tl_feed_version_geometries.feed_version_id = t.id").
		From("feed_versions t").
		Limit(checkLimit(limit))
	if len(ids) > 0 {
		q = q.Where(sq.Eq{"t.id": ids})
	}
	if after != nil {
		q = q.Where(sq.Gt{"t.id": *after})
	}
	q = q.OrderBy("fetched_at desc")
	if where != nil {
		if where.Sha1 != nil {
			q = q.Where(sq.Eq{"sha1": *where.Sha1})
		}
		if len(where.FeedIds) > 0 {
			q = q.Where(sq.Eq{"feed_id": where.FeedIds})
		}
		if where.FeedOnestopID != nil {
			q = q.Where(sq.Eq{"cf.onestop_id": *where.FeedOnestopID})
		}
	}
	return q
}

func FeedVersionServiceLevelSelect(limit *int, after *int, ids []int, where *model.FeedVersionServiceLevelFilter) sq.SelectBuilder {
	q := quickSelectOrder("feed_version_service_levels", limit, after, nil, "")
	if where == nil {
		where = &model.FeedVersionServiceLevelFilter{}
	}
	if where.DistinctOn != nil {
		q = q.Distinct().Options("ON (feed_version_id,route_id)").OrderBy("feed_version_id,route_id")
	} else {
		q = q.OrderBy("id")
	}
	if where.AllRoutes != nil && *where.AllRoutes == true {
		// default
	} else if len(where.RouteIds) > 0 {
		q = q.Where(sq.Eq{"route_id": where.RouteIds})
	} else {
		q = q.Where(sq.Eq{"route_id": nil})
	}
	if where.StartDate != nil {
		q = q.Where(sq.GtOrEq{"start_date": where.StartDate})
	}
	if where.EndDate != nil {
		q = q.Where(sq.LtOrEq{"end_date": where.EndDate})
	}
	return q
}
