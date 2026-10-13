package dbfinder

import (
	"context"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	sq "github.com/irees/squirrel"
)

func (f *Finder) LevelsByIDs(ctx context.Context, ids []int) ([]*model.Level, []error) {
	var ents []*model.Level
	err := dbutil.Select(ctx,
		f.db,
		levelSelect(nil, nil, ids, f.PermFilter(ctx), nil),
		&ents,
	)
	if err != nil {
		return nil, queryErrs(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.Level) int { return ent.ID }), nil
}

func (f *Finder) PathwaysByIDs(ctx context.Context, ids []int) ([]*model.Pathway, []error) {
	var ents []*model.Pathway
	err := dbutil.Select(ctx,
		f.db,
		pathwaySelect(nil, nil, ids, f.PermFilter(ctx), nil),
		&ents,
	)
	if err != nil {
		return nil, queryErrs(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.Pathway) int { return ent.ID }), nil
}

func (f *Finder) LevelsByParentStationIDs(ctx context.Context, limit *int, keys []int) ([][]*model.Level, error) {
	var ents []*model.Level
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			levelSelect(limit, nil, nil, f.PermFilter(ctx), nil),
			"gtfs_stops",
			"id",
			"gtfs_levels",
			"parent_station",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.Level) int { return ent.ParentStation.Int() }), queryErr(ctx, err)
}

// LevelsByFeedVersionIDs returns the levels of each feed version.
func (f *Finder) LevelsByFeedVersionIDs(ctx context.Context, limit *int, after *model.Cursor, where *model.LevelFilter, keys []int) ([][]*model.Level, error) {
	var ents []*model.Level
	q := lateralWrap(levelSelect(limit, after, nil, f.PermFilter(ctx), where), "feed_versions", "id", "gtfs_levels", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.Level) int { return ent.FeedVersionID }), queryErr(ctx, err)
}

func levelSelect(limit *int, after *model.Cursor, ids []int, permFilter *model.PermFilter, where *model.LevelFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_levels.id",
		"gtfs_levels.feed_version_id",
		"gtfs_levels.level_id",
		"gtfs_levels.level_index",
		"gtfs_levels.level_name",
		"gtfs_levels.parent_station",
		"gtfs_levels.geometry",
		"gtfs_levels.created_at",
		"gtfs_levels.updated_at",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_levels").
		Join("feed_versions ON feed_versions.id = gtfs_levels.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
	if len(ids) > 0 {
		q = q.Where(In("gtfs_levels.id", ids))
	}
	if after != nil && after.Valid && after.ID > 0 {
		q = q.Where(sq.Gt{"gtfs_levels.id": after.ID})
	}
	if where != nil {
		if len(where.Ids) > 0 {
			q = q.Where(In("gtfs_levels.id", where.Ids))
		}
		if where.LevelID != nil {
			q = q.Where(sq.Eq{"gtfs_levels.level_id": *where.LevelID})
		}
	}
	q = joinImported(q)
	q = pfJoinCheckFv(q, permFilter)
	return q.OrderBy("gtfs_levels.id ASC").Limit(finderCheckLimit(limit))
}

// PathwaysByFeedVersionIDs returns the pathways of each feed version.
func (f *Finder) PathwaysByFeedVersionIDs(ctx context.Context, limit *int, after *model.Cursor, where *model.PathwayFilter, keys []int) ([][]*model.Pathway, error) {
	var ents []*model.Pathway
	q := lateralWrap(pathwaySelect(limit, after, nil, f.PermFilter(ctx), where), "feed_versions", "id", "gtfs_pathways", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.Pathway) int { return ent.FeedVersionID }), queryErr(ctx, err)
}

func (f *Finder) PathwaysByFromStopIDs(ctx context.Context, limit *int, where *model.PathwayFilter, keys []int) ([][]*model.Pathway, error) {
	var ents []*model.Pathway
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			pathwaySelect(limit, nil, nil, f.PermFilter(ctx), where),
			"gtfs_stops",
			"id",
			"gtfs_pathways",
			"from_stop_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.Pathway) int { return ent.FromStopID.Int() }), queryErr(ctx, err)
}

func (f *Finder) PathwaysByToStopIDs(ctx context.Context, limit *int, where *model.PathwayFilter, keys []int) ([][]*model.Pathway, error) {
	var ents []*model.Pathway
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			pathwaySelect(limit, nil, nil, f.PermFilter(ctx), where),
			"gtfs_stops",
			"id",
			"gtfs_pathways",
			"to_stop_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.Pathway) int { return ent.ToStopID.Int() }), queryErr(ctx, err)
}

func pathwaySelect(limit *int, after *model.Cursor, ids []int, permFilter *model.PermFilter, where *model.PathwayFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.
		Select(
			"gtfs_pathways.id",
			"gtfs_pathways.feed_version_id",
			"gtfs_pathways.pathway_id",
			"gtfs_pathways.from_stop_id",
			"gtfs_pathways.to_stop_id",
			"gtfs_pathways.pathway_mode",
			"gtfs_pathways.is_bidirectional",
			"gtfs_pathways.length",
			"gtfs_pathways.traversal_time",
			"gtfs_pathways.stair_count",
			"gtfs_pathways.max_slope",
			"gtfs_pathways.min_width",
			"gtfs_pathways.signposted_as",
			"gtfs_pathways.reverse_signposted_as",
			"gtfs_pathways.created_at",
			"gtfs_pathways.updated_at",
			"feed_versions.sha1 AS feed_version_sha1",
			"current_feeds.onestop_id AS feed_onestop_id",
		).
		From("gtfs_pathways").
		Join("feed_versions on feed_versions.id = gtfs_pathways.feed_version_id").
		Join("current_feeds on current_feeds.id = feed_versions.feed_id").
		Limit(finderCheckLimit(limit)).
		OrderBy("gtfs_pathways.id")

	if where != nil {
		if len(where.Ids) > 0 {
			q = q.Where(In("gtfs_pathways.id", where.Ids))
		}
		if where.PathwayID != nil {
			q = q.Where(sq.Eq{"gtfs_pathways.pathway_id": *where.PathwayID})
		}
		if where.PathwayMode != nil {
			q = q.Where(sq.Eq{"pathway_mode": where.PathwayMode})
		}
	}
	if len(ids) > 0 {
		q = q.Where(In("gtfs_pathways.id", ids))
	}
	if after != nil && after.Valid && after.ID > 0 {
		q = q.Where(sq.Gt{"gtfs_pathways.id": after.ID})
	}

	// Handle permissions
	q = joinImported(q)
	q = pfJoinCheckFv(q, permFilter)
	return q
}
