package gql

import (
	"context"
	"time"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
)

// TRIP

type tripResolver struct{ *Resolver }

func (r *tripResolver) Cursor(ctx context.Context, obj *model.Trip) (*model.Cursor, error) {
	c := model.NewCursor(obj.FeedVersionID, obj.ID)
	return &c, nil
}

func (r *tripResolver) Route(ctx context.Context, obj *model.Trip) (*model.Route, error) {
	return LoaderFor(ctx).RoutesByIDs.Load(ctx, obj.RouteID.Int())()
}

func (r *tripResolver) FeedVersion(ctx context.Context, obj *model.Trip) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

func (r *tripResolver) Shape(ctx context.Context, obj *model.Trip) (*model.Shape, error) {
	if !obj.ShapeID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).ShapesByIDs.Load(ctx, obj.ShapeID.Int())()
}

func (r *tripResolver) Calendar(ctx context.Context, obj *model.Trip) (*model.Calendar, error) {
	return LoaderFor(ctx).CalendarsByIDs.Load(ctx, obj.ServiceID.Int())()
}

func (r *tripResolver) ServiceDate(ctx context.Context, obj *model.Trip) (*tt.Date, error) {
	if d := tripRunDate(ctx, obj); d.Valid {
		return &d, nil
	}
	return nil, nil
}

// tripRunDate returns the service date of the run a trip stands for: the one
// date it was found on, or without one, its current run. Invalid for a trip
// found on several dates, or one that stands for no run.
func tripRunDate(ctx context.Context, trip *model.Trip) tt.Date {
	if trip.NoRealtime {
		return tt.Date{}
	}
	switch len(trip.RunDates) {
	case 0:
		if d, ok := model.ForContext(ctx).RTFinder.CurrentRun(ctx, trip); ok {
			return d
		}
	case 1:
		return trip.RunDates[0]
	}
	return tt.Date{}
}

func (r *tripResolver) StopTimes(ctx context.Context, obj *model.Trip, limit *int, where *model.TripStopTimeFilter) ([]*model.StopTime, error) {
	loaded, err := LoaderFor(ctx).StopTimesByTripIDs.Load(ctx, tripStopTimeLoaderParam{
		FeedVersionID: obj.FeedVersionID,
		TripID:        obj.ID,
		Limit:         resolverCheckLimit(limit),
		Where:         where,
	})()
	// A trip found without a date takes a lookup to find its current run, so only
	// a query that reads a stop time's date, or the times built from it, pays.
	wantsRT := wantsRTStopTimeUpdate(ctx)
	if !wantsRT && len(obj.RunDates) != 1 && !wantsStopTimeDate(ctx) {
		return loaded, err
	}
	sts := runStopTimes(loaded, tripRunDate(ctx, obj))
	if wantsRT {
		for i, ste := range model.ForContext(ctx).RTFinder.FindStopTimeUpdates(ctx, obj, sts) {
			sts[i].RTStopTimeUpdate = ste
		}
	}
	return sts, err
}

// runStopTimes returns copies of a trip's stop times, on its run's service date
// where it has one. Copies, as the loader shares them across every run of the
// trip in a request, and each run has its own date and realtime data.
func runStopTimes(loaded []*model.StopTime, serviceDate tt.Date) []*model.StopTime {
	sts := make([]*model.StopTime, 0, len(loaded))
	for _, st := range loaded {
		c := *st
		if serviceDate.Valid {
			c.SetServiceDate(serviceDate)
		}
		sts = append(sts, &c)
	}
	return sts
}

func (r *tripResolver) FlexStopTimes(ctx context.Context, obj *model.Trip, limit *int, where *model.TripStopTimeFilter) ([]*model.FlexStopTime, error) {
	loaded, err := LoaderFor(ctx).FlexStopTimesByTripIDs.Load(ctx, tripStopTimeLoaderParam{
		FeedVersionID: obj.FeedVersionID,
		TripID:        obj.ID,
		Limit:         resolverCheckLimit(limit),
		Where:         where,
	})()
	if len(obj.RunDates) != 1 && !wantsStopTimeDate(ctx) {
		return loaded, err
	}
	return runStopTimes(loaded, tripRunDate(ctx, obj)), err
}

func (r *tripResolver) Frequencies(ctx context.Context, obj *model.Trip, limit *int) ([]*model.Frequency, error) {
	return LoaderFor(ctx).FrequenciesByTripIDs.Load(ctx, frequencyLoaderParam{TripID: obj.ID, Limit: resolverCheckLimit(limit)})()
}

func (r *tripResolver) ScheduleRelationship(ctx context.Context, obj *model.Trip) (*model.ScheduleRelationship, error) {
	if rtt := model.ForContext(ctx).RTFinder.FindTrip(ctx, obj); rtt != nil {
		// If TripUpdate TripDescriptor has schedule relationship, use that
		if rtt.Trip != nil && rtt.Trip.ScheduleRelationship != nil {
			sr := rtt.Trip.ScheduleRelationship.String()
			return convertScheduleRelationship(sr), nil
		}
		// Otherwise default to SCHEDULED
		return ptr(model.ScheduleRelationshipScheduled), nil
	}
	return ptr(model.ScheduleRelationshipStatic), nil
}

func (r *tripResolver) Timestamp(ctx context.Context, obj *model.Trip) (*time.Time, error) {
	if rtt := model.ForContext(ctx).RTFinder.FindTrip(ctx, obj); rtt != nil {
		t := time.Unix(int64(rtt.GetTimestamp()), 0).In(time.UTC)
		return &t, nil
	}
	return nil, nil
}

func (r *tripResolver) Alerts(ctx context.Context, obj *model.Trip, active *bool, limit *int) ([]*model.Alert, error) {
	rtAlerts := model.ForContext(ctx).RTFinder.FindAlertsForTrip(ctx, obj, resolverCheckLimit(limit), active)
	return rtAlerts, nil
}

// FREQUENCY

type frequencyResolver struct{ *Resolver }

func (r *frequencyResolver) FeedVersion(ctx context.Context, obj *model.Frequency) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

func (r *frequencyResolver) Trip(ctx context.Context, obj *model.Frequency) (*model.Trip, error) {
	return LoaderFor(ctx).TripsByIDs.Load(ctx, obj.TripID.Int())()
}
