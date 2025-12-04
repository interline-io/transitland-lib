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
	return LoaderFor(ctx).TripsByIDs.Load(ctx, obj.TripID.Int())()
}

func (r *flexStopTimeResolver) Arrival(ctx context.Context, obj *model.FlexStopTime) (*model.StopTimeEvent, error) {
	// TODO: Handle timezone for Location/LocationGroup
	if !obj.StopID.Valid {
		return nil, nil
	}
	loc, ok := model.ForContext(ctx).RTFinder.StopTimezone(ctx, obj.StopID.Int(), "")
	if loc == nil || !ok {
		return nil, nil
	}
	return fromSte(nil, nil, obj.ArrivalTime, obj.ServiceDate, loc), nil
}

func (r *flexStopTimeResolver) Departure(ctx context.Context, obj *model.FlexStopTime) (*model.StopTimeEvent, error) {
	// TODO: Handle timezone for Location/LocationGroup
	if !obj.StopID.Valid {
		return nil, nil
	}
	loc, ok := model.ForContext(ctx).RTFinder.StopTimezone(ctx, obj.StopID.Int(), "")
	if loc == nil || !ok {
		return nil, nil
	}
	return fromSte(nil, nil, obj.DepartureTime, obj.ServiceDate, loc), nil
}

func (r *flexStopTimeResolver) ScheduleRelationship(ctx context.Context, obj *model.FlexStopTime) (*model.ScheduleRelationship, error) {
	return ptr(model.ScheduleRelationshipStatic), nil
}
