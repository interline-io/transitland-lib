package gql

import (
	"context"
	"errors"
	"time"

	dataloader "github.com/graph-gophers/dataloader/v7"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
)

// STOP TIME

type stopTimeResolver struct {
	*Resolver
}

func (r *stopTimeResolver) Stop(ctx context.Context, obj *model.StopTime) (*model.Stop, error) {
	return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.StopID.Int())()
}

// ShapeDistance is the stop's distance along its trip's shape, placed with the
// trip's other stops.
func (r *stopTimeResolver) ShapeDistance(ctx context.Context, obj *model.StopTime) (*int, error) {
	// A trip added in real time has no shape to place it on.
	if obj.RTTrip != nil {
		return nil, nil
	}
	dists, err := LoaderFor(ctx).ShapeDistancesByTripIDs.Load(ctx, tripShapeDistanceLoaderParam{
		FeedVersionID: obj.FeedVersionID,
		TripID:        obj.TripID.Int(),
	})()
	if err != nil {
		return nil, err
	}
	if d := dists[obj.StopSequence.Int()]; d.Valid {
		return ptr(d.Int()), nil
	}
	return nil, nil
}

// shapeDistancesByTripIDs places each trip's stops along its shape, as a
// pattern timetable does, by stop_sequence. A trip without a shape, or with a
// stop that can't be loaded, has none.
func shapeDistancesByTripIDs(ctx context.Context, params []tripShapeDistanceLoaderParam) ([]map[int]tt.Int, []error) {
	ldr := LoaderFor(ctx)
	ret := make([]map[int]tt.Int, len(params))
	errs := make([]error, len(params))
	// Every load is asked for before any is read, so the loaders batch them:
	// the trips and their stop times first, then their shapes and stops.
	tripLoads := make([]dataloader.Thunk[*model.Trip], len(params))
	stopTimeLoads := make([]dataloader.Thunk[[]*model.StopTime], len(params))
	for i, p := range params {
		tripLoads[i] = ldr.TripsByIDs.Load(ctx, p.TripID)
		stopTimeLoads[i] = ldr.StopTimesByTripIDs.Load(ctx, tripStopTimeLoaderParam{
			FeedVersionID: p.FeedVersionID,
			TripID:        p.TripID,
			Limit:         ptr(RESOLVER_PATTERN_MAXLIMIT),
		})
	}
	trips := make([]*model.Trip, len(params))
	stopTimes := make([][]*model.StopTime, len(params))
	shapeLoads := map[int]dataloader.Thunk[*model.Shape]{}
	stopLoads := map[int]dataloader.Thunk[*model.Stop]{}
	for i := range params {
		trip, err := tripLoads[i]()
		if err == nil {
			stopTimes[i], err = stopTimeLoads[i]()
		}
		if err != nil {
			errs[i] = err
			continue
		}
		if trip == nil || !trip.ShapeID.Valid || len(stopTimes[i]) == 0 {
			continue
		}
		trips[i] = trip
		if id := trip.ShapeID.Int(); shapeLoads[id] == nil {
			shapeLoads[id] = ldr.ShapesByIDs.Load(ctx, id)
		}
		for _, st := range stopTimes[i] {
			if id := st.StopID.Int(); stopLoads[id] == nil {
				stopLoads[id] = ldr.StopsByIDs.Load(ctx, id)
			}
		}
	}
	for i, trip := range trips {
		if trip == nil {
			continue
		}
		shape, err := shapeLoads[trip.ShapeID.Int()]()
		if err != nil {
			errs[i] = err
			continue
		}
		if shape == nil {
			continue
		}
		stops := make([]tlxy.Point, len(stopTimes[i]))
		dists := make([]tt.Float, len(stopTimes[i]))
		complete := true
		for k, st := range stopTimes[i] {
			stop, err := stopLoads[st.StopID.Int()]()
			if err != nil {
				errs[i] = err
			}
			if stop == nil {
				complete = false
				break
			}
			stops[k] = stop.ToPoint()
			dists[k] = st.ShapeDistTraveled
		}
		if !complete {
			continue
		}
		ret[i] = map[int]tt.Int{}
		for k, d := range stopShapeDistances(shape, stops, dists) {
			ret[i][stopTimes[i][k].StopSequence.Int()] = d
		}
	}
	return ret, errs
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
	if stu == nil {
		return ptr(model.ScheduleRelationshipStatic), nil
	}
	// A canceled or deleted trip serves none of its stops.
	var tripRel *pb.TripDescriptor_ScheduleRelationship
	if td := stu.TripUpdate.GetTrip(); td != nil {
		tripRel = td.ScheduleRelationship
	}
	if tripRel != nil && (*tripRel == pb.TripDescriptor_CANCELED || *tripRel == pb.TripDescriptor_DELETED) {
		return convertScheduleRelationship(tripRel.String()), nil
	}
	// A skipped stop, or one without data, says so whatever the trip reports.
	if stopRel := stu.StopTimeUpdate.GetScheduleRelationship(); stu.StopTimeUpdate != nil && (stopRel == pb.TripUpdate_StopTimeUpdate_SKIPPED || stopRel == pb.TripUpdate_StopTimeUpdate_NO_DATA) {
		return convertScheduleRelationship(stopRel.String()), nil
	}
	// Otherwise the trip's relationship, when it gives one, or SCHEDULED for
	// any other realtime data, such as a carried delay.
	if tripRel != nil {
		return convertScheduleRelationship(tripRel.String()), nil
	}
	if stu.StopTimeUpdate != nil || stu.EstimatedArrival != 0 || stu.EstimatedDeparture != 0 {
		return ptr(model.ScheduleRelationshipScheduled), nil
	}
	return ptr(model.ScheduleRelationshipStatic), nil
}

func (r *stopTimeResolver) Trip(ctx context.Context, obj *model.StopTime) (*model.Trip, error) {
	if obj.RTTrip != nil {
		return obj.RTTrip, nil
	}
	trip, err := LoaderFor(ctx).TripsByIDs.Load(ctx, obj.TripID.Int())()
	return tripRun(trip, obj.ServiceDate, obj.RunStart, obj.RunOffset), err
}

// tripRun returns a trip as its run on a service date, and for an instance of
// a frequency-based trip, the instance starting at start and shifted by
// offset, so its realtime data is matched to that run. Without a date, the
// run is the trip's current one. A copy, as the loader shares one trip across
// every request.
func tripRun(trip *model.Trip, serviceDate tt.Date, start tt.Seconds, offset tt.Int) *model.Trip {
	if trip == nil || (!serviceDate.Valid && !start.Valid) {
		return trip
	}
	run := *trip
	if serviceDate.Valid {
		run.RunDates = []tt.Date{serviceDate}
	}
	run.RunStart = start
	run.RunOffset = offset
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

// stopTimeLocations returns the stop's timezone, which shows a stop time's
// local times, and the agency's timezone, which counts its scheduled times.
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
// any, with the realtime event that produced the estimate. The scheduled instant
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
			// As a time of the service day, in the scheduled time's clock.
			a.Estimated = ptr(tt.NewSeconds(int(sched.Val) + delay))
		} else {
			// Without a schedule, as a clock time in the stop's timezone.
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
