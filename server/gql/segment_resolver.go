package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
)

// SEGMENTS

type segmentResolver struct{ *Resolver }

func (r *segmentResolver) SegmentPatterns(ctx context.Context, obj *model.Segment) ([]*model.SegmentPattern, error) {
	return LoaderFor(ctx).SegmentPatternsBySegmentIDs.Load(ctx, segmentPatternLoaderParam{SegmentID: obj.ID})()
}

// SEGMENT PATTERNS

type segmentPatternResolver struct{ *Resolver }

func (r *segmentPatternResolver) Segment(ctx context.Context, obj *model.SegmentPattern) (*model.Segment, error) {
	return LoaderFor(ctx).SegmentsByIDs.Load(ctx, obj.SegmentID)()
}

func (r *segmentPatternResolver) Route(ctx context.Context, obj *model.SegmentPattern) (*model.Route, error) {
	return LoaderFor(ctx).RoutesByIDs.Load(ctx, obj.RouteID)()
}

func (r *segmentPatternResolver) Shape(ctx context.Context, obj *model.SegmentPattern) (*model.Shape, error) {
	return LoaderFor(ctx).ShapesByIDs.Load(ctx, obj.ShapeID)()
}
