package dbfinder

import (
	"context"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
)

func (f *Finder) CalendarsByIDs(ctx context.Context, ids []int) ([]*model.Calendar, []error) {
	var ents []*model.Calendar
	err := dbutil.Select(ctx,
		f.db,
		quickSelect("gtfs_calendars", nil, nil, ids),
		&ents,
	)
	if err != nil {
		return nil, logExtendErr(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.Calendar) int { return ent.ID }), nil
}

func (f *Finder) CalendarDatesByServiceIDs(ctx context.Context, limit *int, where *model.CalendarDateFilter, keys []int) ([][]*model.CalendarDate, error) {
	var ents []*model.CalendarDate
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			quickSelectOrder("gtfs_calendar_dates", limit, nil, nil, "date").Where(In("service_id", keys)),
			"gtfs_calendars",
			"id",
			"gtfs_calendar_dates",
			"service_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.CalendarDate) int { return ent.ServiceID.Int() }), err
}
