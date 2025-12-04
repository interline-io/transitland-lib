package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
)

type flexStopTimeResolver struct{ *Resolver }

func (r *flexStopTimeResolver) PickupBookingRule(ctx context.Context, obj *model.FlexStopTime) (*model.BookingRule, error) {
	// TODO
	return nil, nil
}

func (r *flexStopTimeResolver) DropOffBookingRule(ctx context.Context, obj *model.FlexStopTime) (*model.BookingRule, error) {
	// TODO
	return nil, nil
}

func (r *flexStopTimeResolver) Location(ctx context.Context, obj *model.FlexStopTime) (*model.Location, error) {
	// TODO
	return nil, nil
}

func (r *flexStopTimeResolver) LocationGroup(ctx context.Context, obj *model.FlexStopTime) (*model.LocationGroup, error) {
	// TODO
	return nil, nil
}

func (r *flexStopTimeResolver) Trip(ctx context.Context, obj *model.FlexStopTime) (*model.Trip, error) {
	// TODO
	return nil, nil
}

func (r *flexStopTimeResolver) Arrival(ctx context.Context, obj *model.FlexStopTime) (*model.StopTimeEvent, error) {
	// TODO
	return nil, nil
}

func (r *flexStopTimeResolver) Departure(ctx context.Context, obj *model.FlexStopTime) (*model.StopTimeEvent, error) {
	// TODO
	return nil, nil
}

func (r *flexStopTimeResolver) ScheduleRelationship(ctx context.Context, obj *model.FlexStopTime) (*model.ScheduleRelationship, error) {
	// TODO
	return nil, nil
}
