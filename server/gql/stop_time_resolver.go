package gql

import (
	"context"
	"errors"
	"time"

	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
)

// STOP TIME

type stopTimeResolver struct {
	*Resolver
}

func (r *stopTimeResolver) Stop(ctx context.Context, obj *model.StopTime) (*model.Stop, error) {
	return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.StopID.Int())()
}

func (r *stopTimeResolver) PickupBookingRule(ctx context.Context, obj *model.StopTime) (*model.BookingRule, error) {
	if !obj.PickupBookingRuleID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).BookingRulesByIDs.Load(ctx, obj.PickupBookingRuleID.Int())()
}

func (r *stopTimeResolver) DropOffBookingRule(ctx context.Context, obj *model.StopTime) (*model.BookingRule, error) {
	if !obj.DropOffBookingRuleID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).BookingRulesByIDs.Load(ctx, obj.DropOffBookingRuleID.Int())()
}

func (r *stopTimeResolver) ScheduleRelationship(ctx context.Context, obj *model.StopTime) (*model.ScheduleRelationship, error) {
	stu := obj.RTStopTimeUpdate
	// Use StopTimeUpdate ScheduleRelationship value if explicitly provided
	// if stu != nil && stu.StopTimeUpdate != nil && stu.StopTimeUpdate.ScheduleRelationship != nil {
	// 	return convertScheduleRelationship(stu.StopTimeUpdate.ScheduleRelationship.String()), nil
	// }
	// Otherwise, try defaulting to TripUpdate ScheduleRelationship value
	if stu != nil && stu.TripUpdate != nil && stu.TripUpdate.Trip != nil && stu.TripUpdate.Trip.ScheduleRelationship != nil {
		return convertScheduleRelationship(stu.TripUpdate.Trip.ScheduleRelationship.String()), nil
	}
	// Otherwise, if ANY RT data is present (e.g. a propagated delay), default to SCHEDULED
	if stu != nil && (stu.StopTimeUpdate != nil || stu.EstimatedArrival != 0 || stu.EstimatedDeparture != 0) {
		return ptr(model.ScheduleRelationshipScheduled), nil
	}
	// Otherwise, default to STATIC
	return ptr(model.ScheduleRelationshipStatic), nil
}

func (r *stopTimeResolver) Trip(ctx context.Context, obj *model.StopTime) (*model.Trip, error) {
	if obj.RTTrip != nil {
		return obj.RTTrip, nil
	}
	trip, err := LoaderFor(ctx).TripsByIDs.Load(ctx, obj.TripID.Int())()
	return tripRun(trip, obj.ServiceDate), err
}

// tripRun returns a trip as its run on a service date, so its realtime data is
// matched to that run; without a date, the trip unchanged. A copy, as the loader
// shares one trip across every request.
func tripRun(trip *model.Trip, serviceDate tt.Date) *model.Trip {
	if trip == nil || !serviceDate.Valid {
		return trip
	}
	run := *trip
	run.RunDates = []tt.Date{serviceDate}
	return &run
}

func (r *stopTimeResolver) Arrival(ctx context.Context, obj *model.StopTime) (*model.StopTimeEvent, error) {
	var ste *pb.TripUpdate_StopTimeEvent
	var estimated int64
	if u := obj.RTStopTimeUpdate; u != nil {
		ste, estimated = u.ArrivalEvent(), u.EstimatedArrival
	}
	return stopTimeEventAt(ctx, obj, ste, estimated, obj.ArrivalTime)
}

func (r *stopTimeResolver) Departure(ctx context.Context, obj *model.StopTime) (*model.StopTimeEvent, error) {
	var ste *pb.TripUpdate_StopTimeEvent
	var estimated int64
	if u := obj.RTStopTimeUpdate; u != nil {
		ste, estimated = u.DepartureEvent(), u.EstimatedDeparture
	}
	return stopTimeEventAt(ctx, obj, ste, estimated, obj.DepartureTime)
}

// stopTimeEventAt reports one of a stop time's scheduled times, with the
// estimate from its realtime event, in the stop time's timezones.
func stopTimeEventAt(ctx context.Context, obj *model.StopTime, ste *pb.TripUpdate_StopTimeEvent, estimated int64, sched tt.Seconds) (*model.StopTimeEvent, error) {
	stopLoc, agencyLoc, ok := stopTimeLocations(ctx, obj.StopID, obj.FeedVersionID)
	if !ok {
		return nil, errors.New("timezone not available for stop")
	}
	return stopTimeEvent(ste, estimated, sched, obj.ServiceDate, agencyLoc, stopLoc), nil
}

// stopTimeLocations returns the timezone a stop time's local times are shown
// in, its stop's, and the agency timezone its scheduled times count in.
// Without a stop, as at a flex location, both are the agency's; without the
// agency's, both are the stop's.
func stopTimeLocations(ctx context.Context, stopId tt.Key, fvid int) (stopLoc *time.Location, agencyLoc *time.Location, ok bool) {
	rtf := model.ForContext(ctx).RTFinder
	agencyLoc, agencyOK := rtf.FeedVersionTimezone(ctx, fvid)
	agencyOK = agencyOK && agencyLoc != nil
	if !stopId.Valid {
		return agencyLoc, agencyLoc, agencyOK
	}
	stopLoc, ok = rtf.StopTimezone(ctx, stopId.Int(), "")
	if stopLoc == nil || !ok {
		return nil, nil, false
	}
	if !agencyOK {
		agencyLoc = stopLoc
	}
	return stopLoc, agencyLoc, true
}

// stopTimeEvent reports a stop time's scheduled time and its estimate, if
// any, with the realtime event the estimate came from. The scheduled instant
// counts sched from the service date's start in agencyLoc, as GTFS defines
// it, and the local times are in stopLoc.
func stopTimeEvent(ste *pb.TripUpdate_StopTimeEvent, estimated int64, sched tt.Seconds, serviceDate tt.Date, agencyLoc *time.Location, stopLoc *time.Location) *model.StopTimeEvent {
	a := model.StopTimeEvent{
		StopTimezone: stopLoc.String(),
		Scheduled:    &sched,
	}
	if serviceDate.Valid && sched.Valid {
		schedUtc := tt.ServiceDayStart(serviceDate, agencyLoc).Add(time.Duration(sched.Val) * time.Second).UTC()
		a.ScheduledUtc = &schedUtc
		a.ScheduledUnix = ptr(int(schedUtc.Unix()))
		a.ScheduledLocal = ptr(schedUtc.In(stopLoc))
	}
	if estimated != 0 {
		estUtc := time.Unix(estimated, 0).UTC()
		estLocal := estUtc.In(stopLoc)
		a.EstimatedUtc = &estUtc
		a.EstimatedUnix = ptr(int(estimated))
		a.EstimatedLocal = &estLocal
		if a.ScheduledUnix != nil {
			// Against the schedule, within a day; anything more is a bad prediction.
			delay := int(estimated) - *a.ScheduledUnix
			if delay > -(24*3600) && delay < 24*3600 {
				a.EstimatedDelay = ptr(delay)
			}
			// As a time of day, an estimate given as a delay is the scheduled
			// time shifted by it.
			if ste == nil || ste.Time == nil {
				a.Estimated = ptr(tt.NewSeconds(int(sched.Val) + delay))
			}
		}
		if a.Estimated == nil {
			// An estimate given as a time, or without a schedule, is a clock time.
			a.Estimated = ptr(tt.NewSeconds(estLocal.Hour()*3600 + estLocal.Minute()*60 + estLocal.Second()))
		}
	}
	// Only pass through actual message time, delay, uncertainty
	if ste != nil {
		if ste.Time != nil {
			t := time.Unix(*ste.Time, 0).UTC()
			a.TimeUtc = ptr(t)
			a.TimeUnix = ptr(int(t.Unix()))
		}
		if ste.Delay != nil {
			a.Delay = ptr(int(ste.GetDelay()))
		}
		if ste.Uncertainty != nil {
			a.Uncertainty = ptr(int(ste.GetUncertainty()))
		}
	}
	return &a
}
