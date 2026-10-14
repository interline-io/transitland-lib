package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
)

type flexStopTimeResolver struct{ *Resolver }

func (r *flexStopTimeResolver) PickupBookingRule(ctx context.Context, obj *model.FlexStopTime) (*model.BookingRule, error) {
	if !obj.PickupBookingRuleID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).BookingRulesByIDs.Load(ctx, obj.PickupBookingRuleID.Int())()
}

func (r *flexStopTimeResolver) DropOffBookingRule(ctx context.Context, obj *model.FlexStopTime) (*model.BookingRule, error) {
	if !obj.DropOffBookingRuleID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).BookingRulesByIDs.Load(ctx, obj.DropOffBookingRuleID.Int())()
}

func (r *flexStopTimeResolver) Location(ctx context.Context, obj *model.FlexStopTime) (*model.Location, error) {
	if !obj.LocationID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).LocationsByIDs.Load(ctx, obj.LocationID.Int())()
}

func (r *flexStopTimeResolver) LocationGroup(ctx context.Context, obj *model.FlexStopTime) (*model.LocationGroup, error) {
	if !obj.LocationGroupID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).LocationGroupsByIDs.Load(ctx, obj.LocationGroupID.Int())()
}

func (r *flexStopTimeResolver) Trip(ctx context.Context, obj *model.FlexStopTime) (*model.Trip, error) {
	trip, err := LoaderFor(ctx).TripsByIDs.Load(ctx, obj.TripID.Int())()
	return tripRun(trip, obj.ServiceDate, obj.RunStart, obj.RunOffset), err
}

func (r *flexStopTimeResolver) Arrival(ctx context.Context, obj *model.FlexStopTime) (*model.StopTimeEvent, error) {
	return stopTimeEventAt(ctx, obj, nil, 0, obj.ArrivalTime)
}

func (r *flexStopTimeResolver) Departure(ctx context.Context, obj *model.FlexStopTime) (*model.StopTimeEvent, error) {
	return stopTimeEventAt(ctx, obj, nil, 0, obj.DepartureTime)
}

func (r *flexStopTimeResolver) ScheduleRelationship(ctx context.Context, obj *model.FlexStopTime) (*model.ScheduleRelationship, error) {
	return ptr(model.ScheduleRelationshipStatic), nil
}
