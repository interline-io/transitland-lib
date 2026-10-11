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

func (f *Finder) CalendarsByFeedVersionIDs(ctx context.Context, limit *int, after *model.Cursor, where *model.CalendarFilter, keys []int) ([][]*model.Calendar, error) {
	var ents []*model.Calendar
	q := lateralWrap(calendarSelect(limit, after, where), "feed_versions", "id", "gtfs_calendars", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.Calendar) int { return ent.FeedVersionID }), err
}

func calendarSelect(limit *int, after *model.Cursor, where *model.CalendarFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_calendars.id",
		"gtfs_calendars.feed_version_id",
		"gtfs_calendars.service_id",
		"gtfs_calendars.start_date",
		"gtfs_calendars.end_date",
		"gtfs_calendars.monday",
		"gtfs_calendars.tuesday",
		"gtfs_calendars.wednesday",
		"gtfs_calendars.thursday",
		"gtfs_calendars.friday",
		"gtfs_calendars.saturday",
		"gtfs_calendars.sunday",
		"gtfs_calendars.generated",
	).From("gtfs_calendars").
		Join("feed_versions ON feed_versions.id = gtfs_calendars.feed_version_id")
	if after != nil && after.Valid && after.ID > 0 {
		q = q.Where(sq.Gt{"gtfs_calendars.id": after.ID})
	}
	if where != nil {
		if len(where.Ids) > 0 {
			q = q.Where(In("gtfs_calendars.id", where.Ids))
		}
		if where.ServiceID != nil {
			q = q.Where(sq.Eq{"gtfs_calendars.service_id": *where.ServiceID})
		}
	}
	return joinImported(q).OrderBy("gtfs_calendars.id ASC").Limit(finderCheckLimit(limit))
}

func (f *Finder) CalendarDatesByFeedVersionIDs(ctx context.Context, limit *int, after *model.Cursor, where *model.CalendarDateFilter, keys []int) ([][]*model.CalendarDate, error) {
	var ents []*model.CalendarDate
	q := lateralWrap(calendarDateSelect(limit, after, where), "feed_versions", "id", "gtfs_calendar_dates", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.CalendarDate) int { return ent.FeedVersionID }), err
}

func calendarDateSelect(limit *int, after *model.Cursor, where *model.CalendarDateFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_calendar_dates.id",
		"gtfs_calendar_dates.feed_version_id",
		"gtfs_calendar_dates.service_id",
		"gtfs_calendar_dates.date",
		"gtfs_calendar_dates.exception_type",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_calendar_dates").
		Join("feed_versions ON feed_versions.id = gtfs_calendar_dates.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
	if after != nil && after.Valid && after.ID > 0 {
		q = q.Where(sq.Gt{"gtfs_calendar_dates.id": after.ID})
	}
	if where != nil {
		if len(where.Ids) > 0 {
			q = q.Where(In("gtfs_calendar_dates.id", where.Ids))
		}
		if where.Date != nil && where.Date.Valid {
			q = q.Where(sq.Expr("gtfs_calendar_dates.date = ?::date", where.Date.Val))
		}
		if where.ExceptionType != nil {
			q = q.Where(sq.Eq{"gtfs_calendar_dates.exception_type": *where.ExceptionType})
		}
	}
	return joinImported(q).OrderBy("gtfs_calendar_dates.id ASC").Limit(finderCheckLimit(limit))
}
