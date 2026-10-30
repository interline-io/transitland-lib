package gql

import (
	"context"
	"errors"
	"fmt"
	"math"

	dataloader "github.com/graph-gophers/dataloader/v7"
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

// Polyline is the shape's geometry as a polyline at five digits of precision.
func (r *shapeResolver) Polyline(ctx context.Context, obj *model.Shape) (*string, error) {
	return shapePolyline(obj, 5), nil
}

// Polyline6 is the shape's geometry as a polyline at six digits of precision.
func (r *shapeResolver) Polyline6(ctx context.Context, obj *model.Shape) (*string, error) {
	return shapePolyline(obj, 6), nil
}

// shapePolyline encodes a shape's geometry at precision digits, or returns nil
// for a geometry with a point without finite coordinates.
func shapePolyline(obj *model.Shape, precision int) *string {
	pts := obj.Geometry.ToPoints()
	for _, p := range pts {
		if math.IsNaN(p.Lon) || math.IsInf(p.Lon, 0) || math.IsNaN(p.Lat) || math.IsInf(p.Lat, 0) {
			return nil
		}
	}
	s := string(tlxy.EncodePolylinePrecision(pts, precision))
	return &s
}

// stopShapeDistances places a trip's stops along its shape in order, in whole
// meters, with hint as their shape_dist_traveled. A stop with no place on the
// shape has none.
func stopShapeDistances(shape *model.Shape, stops []tlxy.Point, hint []float64) []tt.Int {
	ret := make([]tt.Int, len(stops))
	for i, p := range tlxy.StopPositions(shape.Geometry.ToPoints(), stops, hint) {
		if p.Placed {
			ret[i] = tt.NewInt(int(math.Round(p.Position)))
		}
	}
	return ret
}

// tripStops is a trip's shape, and its stops in order with their
// stop_times.shape_dist_traveled.
type tripStops struct {
	ShapeID tt.Key
	StopIDs []int
	Dists   []tt.Float
}

// placeTrips places each trip's stops along its shape, in whole meters. Trips
// with the same shape, stops and values are placed once. A trip without a
// shape has none, and neither has a stop without coordinates.
func (ldr *Loaders) placeTrips(ctx context.Context, trips []tripStops) ([][]tt.Int, []error) {
	// Every load is asked for before any is read, so the loaders batch them,
	// and their caches load each shape and stop once.
	shapeLoads := make([]dataloader.Thunk[*model.Shape], len(trips))
	stopLoads := make([][]dataloader.Thunk[*model.Stop], len(trips))
	for k, trip := range trips {
		if !trip.ShapeID.Valid {
			continue
		}
		shapeLoads[k] = ldr.ShapesByIDs.Load(ctx, trip.ShapeID.Int())
		for _, id := range trip.StopIDs {
			stopLoads[k] = append(stopLoads[k], ldr.StopsByIDs.Load(ctx, id))
		}
	}
	ret := make([][]tt.Int, len(trips))
	errs := make([]error, len(trips))
	placed := map[string][]tt.Int{}
	for k, trip := range trips {
		if shapeLoads[k] == nil {
			continue
		}
		shape, err := shapeLoads[k]()
		// A stop that didn't load, or has no geometry, has no coordinates,
		// so it has no place either.
		points := make([]tlxy.Point, len(stopLoads[k]))
		for i, load := range stopLoads[k] {
			stop, stopErr := load()
			err = errors.Join(err, stopErr)
			points[i] = tlxy.Point{Lon: math.NaN(), Lat: math.NaN()}
			if stop != nil && stop.Geometry.Valid {
				points[i] = stop.ToPoint()
			}
		}
		if err != nil {
			errs[k] = err
			continue
		}
		if shape == nil {
			continue
		}
		// A null value reads as 0, as the importer reads it, so the hint stays
		// usable only when its nulls all come before its first value.
		hint := make([]float64, len(trip.Dists))
		for i, d := range trip.Dists {
			hint[i] = d.Val
		}
		key := fmt.Sprint(trip.ShapeID.Int(), trip.StopIDs, hint)
		if placed[key] == nil {
			placed[key] = stopShapeDistances(shape, points, hint)
		}
		ret[k] = placed[key]
	}
	return ret, errs
}

func (r *shapeResolver) Trips(ctx context.Context, obj *model.Shape, limit *int, where *model.TripFilter) ([]*model.Trip, error) {
	return LoaderFor(ctx).TripsByShapeIDs.Load(ctx, tripLoaderParam{
		ShapeID:       obj.ID,
		FeedVersionID: obj.FeedVersionID,
		Limit:         resolverCheckLimit(limit),
		Where:         where,
	})()
}
