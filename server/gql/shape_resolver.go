package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
)

type shapeResolver struct{ *Resolver }

// Trips returns the trips that use this shape.
func (r *shapeResolver) Trips(ctx context.Context, obj *model.Shape, limit *int, where *model.TripFilter) ([]*model.Trip, error) {
	return LoaderFor(ctx).TripsByShapeIDs.Load(ctx, tripLoaderParam{
		ShapeID:       obj.ID,
		FeedVersionID: obj.FeedVersionID,
		Limit:         resolverCheckLimit(limit),
		Where:         where,
	})()
}
