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
	if stu != nil && (stu.StopTimeUpdate != nil || stu.LastDelay != nil) {
		return ptr(model.ScheduleRelationshipScheduled), nil
	}
	// Otherwise, default to STATIC
	return ptr(model.ScheduleRelationshipStatic), nil
}

func (r *stopTimeResolver) Trip(ctx context.Context, obj *model.StopTime) (*model.Trip, error) {
	if obj.TripID.Val == "0" && obj.RTTripID != "" {
		t := model.Trip{}
		t.FeedVersionID = obj.FeedVersionID
		t.TripID.Set(obj.RTTripID)
		a, err := model.ForContext(ctx).RTFinder.MakeTrip(ctx, &t)
		return a, err
	}
	return LoaderFor(ctx).TripsByIDs.Load(ctx, obj.TripID.Int())()
}

func (r *stopTimeResolver) Arrival(ctx context.Context, obj *model.StopTime) (*model.StopTimeEvent, error) {
	// Lookup timezone
	loc, ok := model.ForContext(ctx).RTFinder.StopTimezone(ctx, obj.StopID.Int(), "")
	if loc == nil || !ok {
		return nil, errors.New("timezone not available for stop")
	}
	// Create arrival; fallback to RT departure if arrival is not present
	var ste *pb.TripUpdate_StopTimeEvent
	var delay *int32
	if rtStu := obj.RTStopTimeUpdate; rtStu != nil {
		delay = rtStu.LastDelay
		if stu := rtStu.StopTimeUpdate; stu == nil {
		} else if stu.Arrival != nil {
			ste = stu.Arrival
		} else if stu.Departure != nil {
			ste = stu.Departure
		}
	}
	return fromSte(ste, delay, obj.DepartureTime, obj.ServiceDate, loc), nil
}

func (r *stopTimeResolver) Departure(ctx context.Context, obj *model.StopTime) (*model.StopTimeEvent, error) {
	// Lookup timezone
	loc, ok := model.ForContext(ctx).RTFinder.StopTimezone(ctx, obj.StopID.Int(), "")
	if loc == nil || !ok {
		return nil, errors.New("timezone not available for stop")
	}
	// Create departure; fallback to RT arrival if departure is not present
	var ste *pb.TripUpdate_StopTimeEvent
	var delay *int32
	if rtStu := obj.RTStopTimeUpdate; rtStu != nil {
		delay = rtStu.LastDelay
		if stu := rtStu.StopTimeUpdate; stu == nil {
		} else if stu.Departure != nil {
			ste = stu.Departure
		} else if stu.Arrival != nil {
			ste = stu.Arrival
		}
	}
	return fromSte(ste, delay, obj.DepartureTime, obj.ServiceDate, loc), nil
}

func fromSte(ste *pb.TripUpdate_StopTimeEvent, lastDelay *int32, sched tt.Seconds, serviceDate tt.Date, loc *time.Location) *model.StopTimeEvent {
	a := model.StopTimeEvent{
		StopTimezone: loc.String(),
		Scheduled:    &sched,
	}

	// Nothing else to do without timezone
	if loc == nil {
		return &a
	}

	// Apply local timezone
	// Hours, minutes, seconds in local scheduled time
	if serviceDate.Valid && sched.Valid {
		sd := serviceDate.Val
		h, m, s := sched.HMS()
		schedLocal := time.Date(sd.Year(), sd.Month(), sd.Day(), h, m, s, 0, loc)
		schedUtc := schedLocal.In(time.UTC)
		a.ScheduledUtc = &schedUtc
		a.ScheduledUnix = ptr(int(schedUtc.Unix()))
		a.ScheduledLocal = &schedLocal
	}

	// Get timestamp and delay
	var useDelay *int32 = lastDelay
	if ste != nil && ste.Delay != nil {
		useDelay = ste.Delay
	}
	var useTime *int64 = nil
	if ste != nil && ste.Time != nil {
		useTime = ste.Time
	}

	// Apply time or delay value
	if useTime != nil {
		// Use explicit timestamp
		estUtc := time.Unix(*useTime, 0).UTC()
		estLocal := estUtc.In(loc)
		est := tt.NewSeconds(estLocal.Hour()*3600 + estLocal.Minute()*60 + estLocal.Second())
		a.Estimated = ptr(est)
		a.EstimatedUtc = ptr(estUtc)
		a.EstimatedUnix = ptr(int(estUtc.Unix()))
		a.EstimatedLocal = ptr(estLocal)
	} else if useDelay != nil && a.ScheduledUtc != nil {
		// Create a time based on STE delay
		estUtc := a.ScheduledUtc.Add(time.Second * time.Duration(int(*useDelay)))
		estLocal := estUtc.In(loc)
		est := tt.NewSeconds(int(sched.Val) + int(*useDelay))
		a.Estimated = ptr(est)
		a.EstimatedUtc = ptr(estUtc)
		a.EstimatedUnix = ptr(int(estUtc.Unix()))
		a.EstimatedLocal = ptr(estLocal)
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

	// Set EstimatedDelay inclusive of all possible sources
	if a.EstimatedUtc != nil && a.ScheduledUtc != nil {
		// Clamp to reasonable bounds
		estDelay := int(a.EstimatedUtc.Unix() - a.ScheduledUtc.Unix())
		if estDelay > -(24*3600) && estDelay < (24*3600) {
			a.EstimatedDelay = ptr(estDelay)
		}
	}
	return &a
}
