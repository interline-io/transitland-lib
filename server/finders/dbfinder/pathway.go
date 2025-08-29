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
		quickSelect("gtfs_levels", nil, nil, ids),
		&ents,
	)
	if err != nil {
		return nil, logExtendErr(ctx, len(ids), err)
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
		return nil, logExtendErr(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.Pathway) int { return ent.ID }), nil
}

func (f *Finder) LevelsByParentStationIDs(ctx context.Context, limit *int, keys []int) ([][]*model.Level, error) {
	var ents []*model.Level
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			quickSelect("gtfs_levels", limit, nil, nil),
			"gtfs_stops",
			"id",
			"gtfs_levels",
			"parent_station",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.Level) int { return ent.ParentStation.Int() }), err
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
	return arrangeGroup(keys, ents, func(ent *model.Pathway) int { return ent.FromStopID.Int() }), err
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
	return arrangeGroup(keys, ents, func(ent *model.Pathway) int { return ent.ToStopID.Int() }), err
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
		).
		From("gtfs_pathways").
		Join("feed_versions on feed_versions.id = gtfs_pathways.feed_version_id").
		Join("current_feeds on current_feeds.id = feed_versions.feed_id").
		Limit(checkLimit(limit)).
		OrderBy("gtfs_pathways.id")

	if where != nil {
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
	q = pfJoinCheckFv(q, permFilter)
	return q
}
