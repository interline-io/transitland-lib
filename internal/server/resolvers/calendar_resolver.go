package resolvers

import (
	"context"

	"github.com/interline-io/transitland-lib/internal/server/find"
	"github.com/interline-io/transitland-lib/internal/server/model"
	"github.com/interline-io/transitland-lib/tl"
)

// CALENDAR

type calendarResolver struct{ *Resolver }

// StartDate map time.Time to tl.ODate
func (r *calendarResolver) StartDate(ctx context.Context, obj *model.Calendar) (*tl.ODate, error) {
	a := tl.NewODate(obj.StartDate)
	return &a, nil
}

// EndDate map time.Time to tl.ODate
func (r *calendarResolver) EndDate(ctx context.Context, obj *model.Calendar) (*tl.ODate, error) {
	a := tl.NewODate(obj.EndDate)
	return &a, nil
}

func (r *calendarResolver) AddedDates(ctx context.Context, obj *model.Calendar, limit *int) ([]*tl.ODate, error) {
	ents, err := find.For(ctx).CalendarDatesByServiceID.Load(model.CalendarDateParam{ServiceID: obj.ID, Limit: limit, Where: nil})
	if err != nil {
		return nil, err
	}
	ret := []*tl.ODate{}
	for _, ent := range ents {
		if ent.ExceptionType == 1 {
			x := tl.NewODate(ent.Date)
			ret = append(ret, &x)
		}
	}
	return ret, nil
}

func (r *calendarResolver) RemovedDates(ctx context.Context, obj *model.Calendar, limit *int) ([]*tl.ODate, error) {
	ents, err := find.For(ctx).CalendarDatesByServiceID.Load(model.CalendarDateParam{ServiceID: obj.ID, Limit: limit, Where: nil})
	if err != nil {
		return nil, err
	}
	ret := []*tl.ODate{}
	for _, ent := range ents {
		if ent.ExceptionType == 2 {
			x := tl.NewODate(ent.Date)
			ret = append(ret, &x)
		}
	}
	return ret, nil
}
