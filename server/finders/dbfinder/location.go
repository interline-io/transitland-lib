package dbfinder

import (
	"context"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	sq "github.com/irees/squirrel"
)

func (f *Finder) LocationsByFeedVersionIDs(ctx context.Context, limit *int, where *model.LocationFilter, keys []int) ([][]*model.Location, error) {
	var ents []*model.Location
	q := locationSelect(limit, nil, nil, where).Where(In("gtfs_locations.feed_version_id", keys))
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.Location) int { return ent.FeedVersionID }), err
}

func (f *Finder) LocationsByIDs(ctx context.Context, ids []int) ([]*model.Location, []error) {
	var ents []*model.Location
	q := locationSelect(nil, nil, ids, nil)
	if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
		return nil, []error{err}
	}
	return arrangeBy(ids, ents, func(ent *model.Location) int { return ent.ID }), nil
}

func locationSelect(limit *int, after *model.Cursor, ids []int, where *model.LocationFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_locations.id",
		"gtfs_locations.feed_version_id",
		"gtfs_locations.created_at",
		"gtfs_locations.updated_at",
		"gtfs_locations.location_id",
		"gtfs_locations.stop_name",
		"gtfs_locations.stop_desc",
		"gtfs_locations.zone_id",
		"gtfs_locations.stop_url",
		"gtfs_locations.geometry",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_locations").
		Join("feed_versions ON feed_versions.id = gtfs_locations.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")

	if len(ids) > 0 {
		q = q.Where(In("gtfs_locations.id", ids))
	}
	if where != nil {
		if len(where.LocationID) > 0 {
			q = q.Where(In("location_id", where.LocationID))
		}
	}
	q = q.OrderBy("gtfs_locations.id ASC")
	q = q.Limit(finderCheckLimit(limit))
	return q
}
