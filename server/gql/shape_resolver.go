package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlxy"
)

type shapeResolver struct{ *Resolver }

// Distances is the distance along the shape at each of its points.
func (r *shapeResolver) Distances(ctx context.Context, obj *model.Shape) ([]float64, error) {
	return tlxy.ShapeDistances(obj.Geometry.ToPoints()), nil
}

func (r *shapeResolver) Trips(ctx context.Context, obj *model.Shape, limit *int, where *model.TripFilter) ([]*model.Trip, error) {
	return LoaderFor(ctx).TripsByShapeIDs.Load(ctx, tripLoaderParam{
		ShapeID:       obj.ID,
		FeedVersionID: obj.FeedVersionID,
		Limit:         resolverCheckLimit(limit),
		Where:         where,
	})()
}
