package dbfinder

import (
	"context"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	sq "github.com/irees/squirrel"
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

// CalendarsByServiceIDs looks up calendars by (feed_version_id, calendar_id) pairs
func (f *Finder) CalendarsByServiceIDs(ctx context.Context, keys []model.FVServicePair) ([]*model.Calendar, []error) {
	if len(keys) == 0 {
		return nil, nil
	}

	// Group by feed_version_id for more efficient querying
	groups := map[int][]int{}
	for _, key := range keys {
		groups[key.FeedVersionID] = append(groups[key.FeedVersionID], key.ServiceID)
	}

	// Query each feed version group with IN clause
	var ents []*model.Calendar
	for fvid, calendarIds := range groups {
		var groupEnts []*model.Calendar
		q := sq.StatementBuilder.
			Select(
				"gtfs_calendars.id",
				"gtfs_calendars.service_id",
				"gtfs_calendars.monday",
				"gtfs_calendars.tuesday",
				"gtfs_calendars.wednesday",
				"gtfs_calendars.thursday",
				"gtfs_calendars.friday",
				"gtfs_calendars.saturday",
				"gtfs_calendars.sunday",
				"gtfs_calendars.start_date",
				"gtfs_calendars.end_date",
				"gtfs_calendars.created_at",
				"gtfs_calendars.updated_at",
				"gtfs_calendars.feed_version_id",
				"gtfs_calendars.generated",
			).
			From("gtfs_calendars").
			Where(sq.Eq{"feed_version_id": fvid}).
			Where(In("id", calendarIds))

		if err := dbutil.Select(ctx, f.db, q, &groupEnts); err != nil {
			return nil, logExtendErr(ctx, len(keys), err)
		}
		ents = append(ents, groupEnts...)
	}

	// Arrange results to match input order
	return arrangeBy(keys, ents, func(ent *model.Calendar) model.FVServicePair {
		return model.FVServicePair{FeedVersionID: ent.FeedVersionID, ServiceID: ent.ServiceID.Int()}
	}), nil
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
