package gql

import (
	"context"
	"time"

	"github.com/interline-io/transitland-lib/server/model"
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
	sts, err := LoaderFor(ctx).StopTimesByTripIDs.Load(ctx, tripStopTimeLoaderParam{
		FeedVersionID: obj.FeedVersionID,
		TripID:        obj.ID,
		Limit:         resolverCheckLimit(limit),
		Where:         where,
	})()
	for _, st := range sts {
		if ste, ok := model.ForContext(ctx).RTFinder.FindStopTimeUpdate(ctx, obj, st); ok {
			st.RTStopTimeUpdate = ste
		}
	}
	return sts, err
}

func (r *tripResolver) FlexibleStopTimes(ctx context.Context, obj *model.Trip, limit *int, where *model.TripStopTimeFilter) ([]*model.FlexStopTime, error) {
	return LoaderFor(ctx).FlexStopTimesByTripIDs.Load(ctx, tripStopTimeLoaderParam{
		FeedVersionID: obj.FeedVersionID,
		TripID:        obj.ID,
		Limit:         resolverCheckLimit(limit),
		Where:         where,
	})()
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
