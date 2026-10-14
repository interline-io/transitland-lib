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

func (r *tripResolver) StopTimes(ctx context.Context, obj *model.Trip, limit *int, where *model.TripStopTimeFilter) ([]*model.StopTime, error) {
	loaded, err := LoaderFor(ctx).StopTimesByTripIDs.Load(ctx, tripStopTimeLoaderParam{
		FeedVersionID: obj.FeedVersionID,
		TripID:        obj.ID,
		Limit:         ptr(RESOLVER_MAXLIMIT),
		Where:         tripStopTimeLoadFilter(where),
	})()
	if err != nil {
		return nil, err
	}
	sts := windowStopTimes(runStopTimes(obj, loaded), where, resolverCheckLimit(limit))
	if wantsRTStopTimeUpdate(ctx) {
		for i, ste := range model.ForContext(ctx).RTFinder.FindStopTimeUpdates(ctx, obj, sts) {
			sts[i].RTStopTimeUpdate = ste
		}
	}
	return sts, nil
}

// tripStopTimeLoadFilter returns the part of a trip's stop time filter that
// the loader applies, its stops. The time window and the caller's limit apply
// once the run's times are known, so every run of a trip, whatever its window
// or limit, loads the whole trip from one query, capped only by the API's
// maximum.
func tripStopTimeLoadFilter(where *model.TripStopTimeFilter) *model.TripStopTimeFilter {
	if where == nil || len(where.StopIds) == 0 {
		return nil
	}
	return &model.TripStopTimeFilter{StopIds: where.StopIds}
}

// windowStopTimes returns up to limit of the stop times within a filter's time
// window: departing at or after its start, and arriving at or before its end,
// by the run's own times.
func windowStopTimes(sts []*model.StopTime, where *model.TripStopTimeFilter, limit *int) []*model.StopTime {
	ret := make([]*model.StopTime, 0, len(sts))
	for _, st := range sts {
		if limit != nil && len(ret) >= *limit {
			break
		}
		if where != nil && where.Start != nil && (!st.DepartureTime.Valid || st.DepartureTime.Val < where.Start.Val) {
			continue
		}
		if where != nil && where.End != nil && (!st.ArrivalTime.Valid || st.ArrivalTime.Val > where.End.Val) {
			continue
		}
		ret = append(ret, st)
	}
	return ret
}

// runStopTimes returns copies of a trip's stop times, on its run's service date
// when the trip is one run, and shifted to the run's own start when the trip
// is an instance of a frequency-based trip. Copies, as the loader shares them
// across every run of the trip in a request, and each run has its own date,
// times and realtime data.
func runStopTimes(trip *model.Trip, loaded []*model.StopTime) []*model.StopTime {
	sts := make([]*model.StopTime, 0, len(loaded))
	for _, st := range loaded {
		c := *st
		if trip.RunOffset.Valid {
			c.ArrivalTime = shifted(c.ArrivalTime, trip.RunOffset.Val)
			c.DepartureTime = shifted(c.DepartureTime, trip.RunOffset.Val)
			c.RunStart, c.RunOffset = trip.RunStart, trip.RunOffset
		}
		if len(trip.RunDates) == 1 {
			c.SetServiceDate(trip.RunDates[0])
		}
		sts = append(sts, &c)
	}
	return sts
}

// shifted returns a scheduled time moved by offset seconds, or an unknown time
// unchanged.
func shifted(t tt.Seconds, offset int64) tt.Seconds {
	if t.Valid {
		t.Val += offset
	}
	return t
}

func (r *tripResolver) FlexStopTimes(ctx context.Context, obj *model.Trip, limit *int, where *model.TripStopTimeFilter) ([]*model.FlexStopTime, error) {
	loaded, err := LoaderFor(ctx).FlexStopTimesByTripIDs.Load(ctx, tripStopTimeLoaderParam{
		FeedVersionID: obj.FeedVersionID,
		TripID:        obj.ID,
		Limit:         ptr(RESOLVER_MAXLIMIT),
		Where:         tripStopTimeLoadFilter(where),
	})()
	if err != nil {
		return nil, err
	}
	return windowStopTimes(runStopTimes(obj, loaded), where, resolverCheckLimit(limit)), nil
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
