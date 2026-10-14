package gql

import (
	"context"
	"time"

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
	return tripRun(trip, obj.ServiceDate), err
}

func (r *flexStopTimeResolver) Arrival(ctx context.Context, obj *model.FlexStopTime) (*model.StopTimeEvent, error) {
	stopLoc, agencyLoc, ok := flexStopTimeLocations(ctx, obj)
	if !ok {
		return nil, nil
	}
	return stopTimeEvent(nil, 0, obj.ArrivalTime, obj.ServiceDate, agencyLoc, stopLoc), nil
}

func (r *flexStopTimeResolver) Departure(ctx context.Context, obj *model.FlexStopTime) (*model.StopTimeEvent, error) {
	stopLoc, agencyLoc, ok := flexStopTimeLocations(ctx, obj)
	if !ok {
		return nil, nil
	}
	return stopTimeEvent(nil, 0, obj.DepartureTime, obj.ServiceDate, agencyLoc, stopLoc), nil
}

// flexStopTimeLocations returns the timezone a flex stop time's local times
// are shown in, that of its stop or else of its feed version's agency, and
// the agency timezone its scheduled times count in.
func flexStopTimeLocations(ctx context.Context, obj *model.FlexStopTime) (stopLoc *time.Location, agencyLoc *time.Location, ok bool) {
	agencyLoc, agencyOK := model.ForContext(ctx).RTFinder.FeedVersionTimezone(ctx, obj.FeedVersionID)
	if obj.StopID.Valid {
		stopLoc, ok = model.ForContext(ctx).RTFinder.StopTimezone(ctx, obj.StopID.Int(), "")
	} else if obj.LocationID.Valid || obj.LocationGroupID.Valid {
		stopLoc, ok = agencyLoc, agencyOK
	}
	if stopLoc == nil || !ok {
		return nil, nil, false
	}
	if agencyLoc == nil || !agencyOK {
		agencyLoc = stopLoc
	}
	return stopLoc, agencyLoc, true
}

func (r *flexStopTimeResolver) ScheduleRelationship(ctx context.Context, obj *model.FlexStopTime) (*model.ScheduleRelationship, error) {
	return ptr(model.ScheduleRelationshipStatic), nil
}
