package resolvers

import (
	"context"

	"github.com/interline-io/transitland-lib/internal/server/find"
	"github.com/interline-io/transitland-lib/internal/server/model"
)

// TRIP

type tripResolver struct{ *Resolver }

func (r *tripResolver) Route(ctx context.Context, obj *model.Trip) (*model.Route, error) {
	return find.For(ctx).RoutesByID.Load(atoi(obj.RouteID))
}

func (r *tripResolver) FeedVersion(ctx context.Context, obj *model.Trip) (*model.FeedVersion, error) {
	return find.For(ctx).FeedVersionsByID.Load(obj.FeedVersionID)
}

func (r *tripResolver) Shape(ctx context.Context, obj *model.Trip) (*model.Shape, error) {
	if !obj.ShapeID.Valid {
		return nil, nil
	}
	return find.For(ctx).ShapesByID.Load(obj.ShapeID.Int())
}

func (r *tripResolver) Calendar(ctx context.Context, obj *model.Trip) (*model.Calendar, error) {
	return find.For(ctx).CalendarsByID.Load(atoi(obj.ServiceID))
}

func (r *tripResolver) StopTimes(ctx context.Context, obj *model.Trip, limit *int) ([]*model.StopTime, error) {
	return find.For(ctx).StopTimesByTripID.Load(model.StopTimeParam{TripID: obj.ID, Limit: limit})
}

func (r *tripResolver) Frequencies(ctx context.Context, obj *model.Trip, limit *int) ([]*model.Frequency, error) {
	return find.For(ctx).FrequenciesByTripID.Load(model.FrequencyParam{TripID: obj.ID, Limit: limit})
}
