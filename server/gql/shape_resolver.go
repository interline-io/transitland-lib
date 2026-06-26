package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
)

type shapeResolver struct{ *Resolver }

// Trips returns the trips that use this shape. The after argument is accepted
// for forward-compatibility but not yet applied — cursor pagination will be
// added uniformly across resolvers later.
func (r *shapeResolver) Trips(ctx context.Context, obj *model.Shape, limit *int, after *int, where *model.TripFilter) ([]*model.Trip, error) {
	_ = after
	return LoaderFor(ctx).TripsByShapeIDs.Load(ctx, tripLoaderParam{
		ShapeID:       obj.ID,
		FeedVersionID: obj.FeedVersionID,
		Limit:         resolverCheckLimit(limit),
		Where:         where,
	})()
}
