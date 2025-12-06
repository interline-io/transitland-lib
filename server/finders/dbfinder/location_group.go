package dbfinder

import (
	"context"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	sq "github.com/irees/squirrel"
)

func (f *Finder) LocationGroupsByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.LocationGroup, error) {
	var ents []*model.LocationGroup
	q := locationGroupSelect(limit, nil, nil).Where(In("gtfs_location_groups.feed_version_id", keys))
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.LocationGroup) int { return ent.FeedVersionID }), err
}

func (f *Finder) LocationGroupsByIDs(ctx context.Context, ids []int) ([]*model.LocationGroup, []error) {
	var ents []*model.LocationGroup
	q := locationGroupSelect(nil, nil, ids)
	if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
		return nil, []error{err}
	}
	return arrangeBy(ids, ents, func(ent *model.LocationGroup) int { return ent.ID }), nil
}

func locationGroupSelect(limit *int, after *model.Cursor, ids []int) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_location_groups.id",
		"gtfs_location_groups.feed_version_id",
		"gtfs_location_groups.created_at",
		"gtfs_location_groups.updated_at",
		"gtfs_location_groups.location_group_id",
		"gtfs_location_groups.location_group_name",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_location_groups").
		Join("feed_versions ON feed_versions.id = gtfs_location_groups.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")

	if len(ids) > 0 {
		q = q.Where(In("gtfs_location_groups.id", ids))
	}
	q = q.Limit(finderCheckLimit(limit))
	return q
}
