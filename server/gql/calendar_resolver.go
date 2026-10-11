package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
)

// CALENDAR

type calendarResolver struct{ *Resolver }

func (r *calendarResolver) AddedDates(ctx context.Context, obj *model.Calendar, limit *int) ([]*tt.Date, error) {
	ents, err := LoaderFor(ctx).CalendarDatesByServiceIDs.Load(ctx, calendarDateLoaderParam{ServiceID: obj.ID, Limit: resolverCheckLimit(limit), Where: nil})()
	if err != nil {
		return nil, err
	}
	ret := []*tt.Date{}
	for _, ent := range ents {
		if ent.ExceptionType.Val == 1 {
			x := tt.NewDate(ent.Date.Val)
			ret = append(ret, &x)
		}
	}
	return ret, nil
}

func (r *calendarResolver) RemovedDates(ctx context.Context, obj *model.Calendar, limit *int) ([]*tt.Date, error) {
	ents, err := LoaderFor(ctx).CalendarDatesByServiceIDs.Load(ctx, calendarDateLoaderParam{ServiceID: obj.ID, Limit: resolverCheckLimit(limit), Where: nil})()
	if err != nil {
		return nil, err
	}
	ret := []*tt.Date{}
	for _, ent := range ents {
		if ent.ExceptionType.Val == 2 {
			x := tt.NewDate(ent.Date.Val)
			ret = append(ret, &x)
		}
	}
	return ret, nil
}

// CALENDAR DATE

type calendarDateResolver struct{ *Resolver }

func (r *calendarDateResolver) FeedVersion(ctx context.Context, obj *model.CalendarDate) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

func (r *calendarDateResolver) Service(ctx context.Context, obj *model.CalendarDate) (*model.Calendar, error) {
	return LoaderFor(ctx).CalendarsByIDs.Load(ctx, obj.ServiceID.Int())()
}
