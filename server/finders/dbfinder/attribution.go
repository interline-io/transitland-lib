package dbfinder

import (
	"context"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	sq "github.com/irees/squirrel"
)

// AttributionsByFeedVersionIDs returns the attributions of each feed version.
func (f *Finder) AttributionsByFeedVersionIDs(ctx context.Context, limit *int, after *model.Cursor, where *model.AttributionFilter, keys []int) ([][]*model.Attribution, error) {
	var ents []*model.Attribution
	q := lateralWrap(attributionSelect(limit, after, f.PermFilter(ctx), where), "feed_versions", "id", "gtfs_attributions", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.Attribution) int { return ent.FeedVersionID }), err
}

func attributionSelect(limit *int, after *model.Cursor, permFilter *model.PermFilter, where *model.AttributionFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_attributions.id",
		"gtfs_attributions.feed_version_id",
		"gtfs_attributions.attribution_id",
		"gtfs_attributions.organization_name",
		"gtfs_attributions.agency_id",
		"gtfs_attributions.route_id",
		"gtfs_attributions.trip_id",
		"gtfs_attributions.is_producer",
		"gtfs_attributions.is_operator",
		"gtfs_attributions.is_authority",
		"gtfs_attributions.attribution_url",
		"gtfs_attributions.attribution_email",
		"gtfs_attributions.attribution_phone",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_attributions").
		Join("feed_versions ON feed_versions.id = gtfs_attributions.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
	if after != nil && after.Valid && after.ID > 0 {
		q = q.Where(sq.Gt{"gtfs_attributions.id": after.ID})
	}
	if where != nil {
		if len(where.Ids) > 0 {
			q = q.Where(In("gtfs_attributions.id", where.Ids))
		}
		if where.AttributionID != nil {
			q = q.Where(sq.Eq{"gtfs_attributions.attribution_id": *where.AttributionID})
		}
	}
	q = joinImported(q)
	q = pfJoinCheckFv(q, permFilter)
	return q.OrderBy("gtfs_attributions.id ASC").Limit(finderCheckLimit(limit))
}
