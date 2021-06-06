package resolvers

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
)

// CALENDAR

type calendarResolver struct{ *Resolver }

func (r *calendarResolver) AddedDates(ctx context.Context, obj *model.Calendar) ([]*model.CalendarDate, error) {
	// TODO
	return nil, nil
}

func (r *calendarResolver) RemovedDates(ctx context.Context, obj *model.Calendar) ([]*model.CalendarDate, error) {
	// TODO
	return nil, nil
}
