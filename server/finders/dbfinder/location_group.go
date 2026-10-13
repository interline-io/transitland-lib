package dbfinder

import (
	"context"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	sq "github.com/irees/squirrel"
)

func (f *Finder) LocationGroupsByFeedVersionIDs(ctx context.Context, limit *int, where *model.LocationGroupFilter, keys []int) ([][]*model.LocationGroup, error) {
	var ents []*model.LocationGroup
	q := locationGroupSelect(limit, nil, nil, where).Where(In("gtfs_location_groups.feed_version_id", keys))
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.LocationGroup) int { return ent.FeedVersionID }), queryErr(ctx, err)
}

func (f *Finder) LocationGroupsByIDs(ctx context.Context, ids []int) ([]*model.LocationGroup, []error) {
	var ents []*model.LocationGroup
	q := locationGroupSelect(nil, nil, ids, nil)
	if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
		return nil, queryErrs(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.LocationGroup) int { return ent.ID }), nil
}

func (f *Finder) LocationGroupsByStopIDs(ctx context.Context, limit *int, keys []int) ([][]*model.LocationGroup, error) {
	var ents []*model.LocationGroup
	q := locationGroupSelect(limit, nil, nil, nil).
		Join("gtfs_location_group_stops ON gtfs_location_group_stops.location_group_id = gtfs_location_groups.id").
		Column("gtfs_location_group_stops.stop_id AS with_stop_id")
	err := dbutil.Select(ctx, f.db, lateralWrap(q, "gtfs_stops", "id", "gtfs_location_group_stops", "stop_id", keys), &ents)
	return arrangeGroup(keys, ents, func(ent *model.LocationGroup) int { return ent.WithStopID.Int() }), queryErr(ctx, err)
}

// LocationGroupStopsByFeedVersionIDs returns the location group stops of each feed version.
func (f *Finder) LocationGroupStopsByFeedVersionIDs(ctx context.Context, limit *int, after *model.Cursor, where *model.LocationGroupStopFilter, keys []int) ([][]*model.LocationGroupStop, error) {
	var ents []*model.LocationGroupStop
	q := lateralWrap(locationGroupStopSelect(limit, after, f.PermFilter(ctx), where), "feed_versions", "id", "gtfs_location_group_stops", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.LocationGroupStop) int { return ent.FeedVersionID }), queryErr(ctx, err)
}

func locationGroupStopSelect(limit *int, after *model.Cursor, permFilter *model.PermFilter, where *model.LocationGroupStopFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_location_group_stops.id",
		"gtfs_location_group_stops.feed_version_id",
		"gtfs_location_group_stops.location_group_id",
		"gtfs_location_group_stops.stop_id",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_location_group_stops").
		Join("feed_versions ON feed_versions.id = gtfs_location_group_stops.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
	if after != nil && after.Valid && after.ID > 0 {
		q = q.Where(sq.Gt{"gtfs_location_group_stops.id": after.ID})
	}
	if where != nil {
		if len(where.Ids) > 0 {
			q = q.Where(In("gtfs_location_group_stops.id", where.Ids))
		}
	}
	q = joinImported(q)
	q = pfJoinCheckFv(q, permFilter)
	return q.OrderBy("gtfs_location_group_stops.id ASC").Limit(finderCheckLimit(limit))
}

func locationGroupSelect(limit *int, _ *model.Cursor, ids []int, where *model.LocationGroupFilter) sq.SelectBuilder {
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
	if where != nil {
		if len(where.Ids) > 0 {
			q = q.Where(In("gtfs_location_groups.id", where.Ids))
		}
		if where.LocationGroupID != nil && *where.LocationGroupID != "" {
			q = q.Where(sq.Eq{"location_group_id": *where.LocationGroupID})
		}
	}
	q = q.OrderBy("gtfs_location_groups.id ASC")
	q = q.Limit(finderCheckLimit(limit))
	return q
}
