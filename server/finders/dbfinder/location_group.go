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
	return arrangeGroup(keys, ents, func(ent *model.LocationGroup) int { return ent.FeedVersionID }), err
}

func (f *Finder) LocationGroupsByIDs(ctx context.Context, ids []int) ([]*model.LocationGroup, []error) {
	var ents []*model.LocationGroup
	q := locationGroupSelect(nil, nil, ids, nil)
	if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
		return nil, []error{err}
	}
	return arrangeBy(ids, ents, func(ent *model.LocationGroup) int { return ent.ID }), nil
}

func (f *Finder) LocationGroupsByStopIDs(ctx context.Context, limit *int, keys []int) ([][]*model.LocationGroup, error) {
	var ents []*model.LocationGroup
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			locationGroupSelect(limit, nil, nil, nil).
				Join("gtfs_location_group_stops ON gtfs_location_group_stops.location_group_id = gtfs_location_groups.id"),
			"gtfs_stops",
			"id",
			"gtfs_location_group_stops",
			"stop_id",
			keys,
		),
		&ents,
	)
	// Group results by stop_id through the join table
	// We need to query the join table to get the mapping
	type locationGroupWithStop struct {
		LocationGroupID int `db:"location_group_id"`
		StopID          int `db:"stop_id"`
	}
	var mappings []locationGroupWithStop
	mappingQuery := sq.StatementBuilder.
		Select("location_group_id", "stop_id").
		From("gtfs_location_group_stops").
		Where(In("stop_id", keys))
	_ = dbutil.Select(ctx, f.db, mappingQuery, &mappings)

	// Build location_group_id -> stop_id map
	lgToStop := make(map[int]int)
	for _, m := range mappings {
		lgToStop[m.LocationGroupID] = m.StopID
	}

	return arrangeGroup(keys, ents, func(ent *model.LocationGroup) int { return lgToStop[ent.ID] }), err
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
