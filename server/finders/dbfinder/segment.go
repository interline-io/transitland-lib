package dbfinder

import (
	"context"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	sq "github.com/irees/squirrel"
)

func (f *Finder) SegmentsByFeedVersionIDs(ctx context.Context, limit *int, where *model.SegmentFilter, keys []int) ([][]*model.Segment, error) {
	var ents []*model.Segment
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			quickSelect("tl_segments", limit, nil, nil),
			"feed_versions",
			"id",
			"tl_segments",
			"feed_version_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.Segment) int { return ent.FeedVersionID }), err
}

func (f *Finder) SegmentsByIDs(ctx context.Context, ids []int) ([]*model.Segment, []error) {
	var ents []*model.Segment
	err := dbutil.Select(ctx,
		f.db,
		quickSelect("tl_segments", nil, nil, ids),
		&ents,
	)
	if err != nil {
		return nil, logExtendErr(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.Segment) int { return ent.ID }), nil
}

func (f *Finder) SegmentsByRouteIDs(ctx context.Context, limit *int, where *model.SegmentFilter, keys []int) ([][]*model.Segment, error) {
	var ents []*model.Segment
	q := sq.Select("s.id", "s.way_id", "s.geometry", "s.route_id", "s.route_id with_route_id").
		From("gtfs_routes").
		JoinClause(
			`join lateral (select distinct on (tl_segments.id, tl_segment_patterns.route_id) tl_segments.id, tl_segments.way_id, tl_segments.geometry, tl_segment_patterns.route_id from tl_segments join tl_segment_patterns on tl_segment_patterns.segment_id = tl_segments.id where tl_segment_patterns.route_id = gtfs_routes.id limit ?) s on true`,
			checkLimit(limit),
		).
		Where(In("gtfs_routes.id", keys))
	err := dbutil.Select(ctx,
		f.db,
		q,
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.Segment) int { return ent.WithRouteID }), err
}

func (f *Finder) SegmentPatternsByRouteIDs(ctx context.Context, limit *int, where *model.SegmentPatternFilter, keys []int) ([][]*model.SegmentPattern, error) {
	var ents []*model.SegmentPattern
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			quickSelect("tl_segment_patterns", limit, nil, nil),
			"gtfs_routes",
			"id",
			"tl_segment_patterns",
			"route_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.SegmentPattern) int { return ent.RouteID }), err
}

func (f *Finder) SegmentPatternsBySegmentIDs(ctx context.Context, limit *int, where *model.SegmentPatternFilter, keys []int) ([][]*model.SegmentPattern, error) {
	var ents []*model.SegmentPattern
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			quickSelect("tl_segment_patterns", limit, nil, nil),
			"tl_segments",
			"id",
			"tl_segment_patterns",
			"segment_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.SegmentPattern) int { return ent.SegmentID }), err
}
