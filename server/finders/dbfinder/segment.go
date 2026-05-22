package dbfinder

import (
	"context"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	sq "github.com/irees/squirrel"
)

func (f *Finder) SegmentsByFeedVersionIDs(ctx context.Context, limit *int, where *model.SegmentFilter, keys []int) ([][]*model.Segment, error) {
	var ents []*model.Segment
	// row_number/partition is intentional: lets the planner use the
	// (feed_version_id, id) index. The PermFilter check is layered on
	// outside the subquery so we don't disturb that index path.
	subq := sq.StatementBuilder.
		Select(
			"tl_segments.id",
			"tl_segments.feed_version_id",
			"tl_segments.way_id",
			"tl_segments.geometry",
			"row_number() over (partition by tl_segments.feed_version_id order by tl_segments.id) as rn",
		).
		From("tl_segments").
		Where(In("tl_segments.feed_version_id", keys))
	q := sq.StatementBuilder.
		Select("t.id", "t.feed_version_id", "t.way_id", "t.geometry").
		FromSelect(subq, "t").
		Join("feed_versions on feed_versions.id = t.feed_version_id").
		Join("current_feeds on current_feeds.id = feed_versions.feed_id").
		Where(sq.LtOrEq{"t.rn": finderCheckLimit(limit)})
	q = pfJoinCheckFv(q, f.PermFilter(ctx))
	err := dbutil.Select(ctx,
		f.db,
		q,
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.Segment) int { return ent.FeedVersionID }), err
}

func (f *Finder) SegmentsByIDs(ctx context.Context, ids []int) ([]*model.Segment, []error) {
	var ents []*model.Segment
	err := dbutil.Select(ctx,
		f.db,
		segmentSelect(nil, nil, ids, f.PermFilter(ctx)),
		&ents,
	)
	if err != nil {
		return nil, logExtendErr(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.Segment) int { return ent.ID }), nil
}

func (f *Finder) SegmentsByRouteIDs(ctx context.Context, limit *int, where *model.SegmentFilter, keys []int) ([][]*model.Segment, error) {
	var ents []*model.Segment
	inner := sq.StatementBuilder.
		Select(
			"tl_segments.id",
			"tl_segments.way_id",
			"tl_segments.geometry",
			"tl_segment_patterns.route_id",
		).
		Distinct().Options("on (tl_segments.id, tl_segment_patterns.route_id)").
		From("tl_segments").
		Join("tl_segment_patterns on tl_segment_patterns.segment_id = tl_segments.id").
		Join("feed_versions on feed_versions.id = tl_segments.feed_version_id").
		Join("current_feeds on current_feeds.id = feed_versions.feed_id").
		Where("tl_segment_patterns.route_id = gtfs_routes.id").
		Limit(finderCheckLimit(limit))
	inner = pfJoinCheckFv(inner, f.PermFilter(ctx))
	q := sq.StatementBuilder.
		Select("s.id", "s.way_id", "s.geometry", "s.route_id", "s.route_id as with_route_id").
		From("gtfs_routes").
		JoinClause(inner.Prefix("JOIN LATERAL (").Suffix(") s on true")).
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
			segmentPatternSelect(limit, nil, nil, f.PermFilter(ctx)),
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
			segmentPatternSelect(limit, nil, nil, f.PermFilter(ctx)),
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

func segmentSelect(limit *int, after *model.Cursor, ids []int, permFilter *model.PermFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.
		Select(
			"tl_segments.id",
			"tl_segments.feed_version_id",
			"tl_segments.way_id",
			"tl_segments.geometry",
		).
		From("tl_segments").
		Join("feed_versions on feed_versions.id = tl_segments.feed_version_id").
		Join("current_feeds on current_feeds.id = feed_versions.feed_id").
		OrderBy("tl_segments.id").
		Limit(finderCheckLimit(limit))
	if len(ids) > 0 {
		q = q.Where(In("tl_segments.id", ids))
	}
	if after != nil && after.Valid && after.ID > 0 {
		q = q.Where(sq.Gt{"tl_segments.id": after.ID})
	}
	q = pfJoinCheckFv(q, permFilter)
	return q
}

func segmentPatternSelect(limit *int, after *model.Cursor, ids []int, permFilter *model.PermFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.
		Select("tl_segment_patterns.*").
		From("tl_segment_patterns").
		Join("feed_versions on feed_versions.id = tl_segment_patterns.feed_version_id").
		Join("current_feeds on current_feeds.id = feed_versions.feed_id").
		OrderBy("tl_segment_patterns.id").
		Limit(finderCheckLimit(limit))
	if len(ids) > 0 {
		q = q.Where(In("tl_segment_patterns.id", ids))
	}
	if after != nil && after.Valid && after.ID > 0 {
		q = q.Where(sq.Gt{"tl_segment_patterns.id": after.ID})
	}
	q = pfJoinCheckFv(q, permFilter)
	return q
}
