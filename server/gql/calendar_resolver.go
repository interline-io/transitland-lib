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
