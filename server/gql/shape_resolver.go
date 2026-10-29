package gql

import (
	"context"
	"math"
	"slices"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
)

type shapeResolver struct{ *Resolver }

// Distances is the distance along the shape at each of its points, or none
// for a shape with a point without finite coordinates.
func (r *shapeResolver) Distances(ctx context.Context, obj *model.Shape) ([]float64, error) {
	return tlxy.ShapeDistances(obj.Geometry.ToPoints()), nil
}

// stopShapeDistances places a trip's stops along its shape in order, in whole
// meters, with their stop times' shape_dist_traveled as the hint. A stop with
// no place on the shape has none.
func stopShapeDistances(shape *model.Shape, stops []tlxy.Point, dists []tt.Float) []tt.Int {
	var hint []float64
	if slices.ContainsFunc(dists, func(d tt.Float) bool { return d.Valid }) {
		hint = make([]float64, len(dists))
		for i, d := range dists {
			hint[i] = d.Val
		}
	}
	ret := make([]tt.Int, len(stops))
	for i, p := range tlxy.StopPositions(shape.Geometry.ToPoints(), stops, hint) {
		if p.Placed {
			ret[i] = tt.NewInt(int(math.Round(p.Position)))
		}
	}
	return ret
}

func (r *shapeResolver) Trips(ctx context.Context, obj *model.Shape, limit *int, where *model.TripFilter) ([]*model.Trip, error) {
	return LoaderFor(ctx).TripsByShapeIDs.Load(ctx, tripLoaderParam{
		ShapeID:       obj.ID,
		FeedVersionID: obj.FeedVersionID,
		Limit:         resolverCheckLimit(limit),
		Where:         where,
	})()
}
