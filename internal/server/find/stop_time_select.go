package find

import (
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/internal/server/model"
	"github.com/lib/pq"
)

func StopTimeSelect(limit *int, after *int, ids []int, where *model.StopTimeFilter) sq.SelectBuilder {
	qView := sq.StatementBuilder.Select(
		"gtfs_trips.journey_pattern_id",
		"gtfs_trips.journey_pattern_offset",
		"gtfs_trips.id AS trip_id",
		"gtfs_trips.feed_version_id",
		"st.stop_id",
		"st.arrival_time + gtfs_trips.journey_pattern_offset AS arrival_time",
		"st.departure_time + gtfs_trips.journey_pattern_offset AS departure_time",
		"st.stop_sequence",
		"st.shape_dist_traveled",
		"st.pickup_type",
		"st.drop_off_type",
		"st.timepoint",
		"st.interpolated",
		"st.stop_headsign",
	).
		From("gtfs_trips").
		Join("gtfs_trips t2 ON t2.trip_id::text = gtfs_trips.journey_pattern_id AND gtfs_trips.feed_version_id = t2.feed_version_id").
		Join("gtfs_stop_times st ON st.trip_id = t2.id")

	q := sq.StatementBuilder.Select("*").
		FromSelect(qView, "t").
		Limit(checkLimit(limit))
	q = q.OrderBy("id asc, stop_sequence asc")
	return q
}

func StopDeparturesSelect(limit *int, after *int, ids []int, where *model.StopTimeFilter) sq.SelectBuilder {
	// TODO: support journey patterns properly
	serviceDate := time.Now()
	if where != nil && where.ServiceDate != nil {
		serviceDate = where.ServiceDate.Time
	}
	q := sq.StatementBuilder.Select("sts.*").
		Prefix(`WITH fvids as (select distinct on(feed_version_id) feed_version_id id from gtfs_stops where id = ANY(?))`, pq.Array(ids)).
		From("gtfs_stops").
		Join("gtfs_stop_times sts on gtfs_stops.id = sts.stop_id and sts.feed_version_id = gtfs_stops.feed_version_id").
		Join("gtfs_trips on gtfs_trips.id = sts.trip_id").
		JoinClause(`inner join (
			SELECT
				id
			FROM
				gtfs_calendars
			WHERE
				start_date <= ?
				AND end_date >= ?
				AND (CASE EXTRACT(isodow FROM ?::date)
					WHEN 1 THEN monday = 1
					WHEN 2 THEN tuesday = 1
					WHEN 3 THEN wednesday = 1
					WHEN 4 THEN thursday = 1
					WHEN 5 THEN friday = 1
					WHEN 6 THEN saturday = 1
					WHEN 7 THEN sunday = 1
				END)
				AND feed_version_id IN (select id from fvids)
				AND id NOT IN (
					SELECT service_id 
					FROM gtfs_calendar_dates 
					WHERE service_id = gtfs_calendars.id AND date = ? AND exception_type = 2 AND feed_version_id in (select id from fvids)
				)
			UNION
			SELect
				service_id as id
			FROM
				gtfs_calendar_dates
			WHERE
				date = ?
				AND exception_type = 1
				AND feed_version_id in (select id from fvids)
		) gc on gc.id = gtfs_trips.service_id`,
			serviceDate,
			serviceDate,
			serviceDate,
			serviceDate,
			serviceDate).
		Where(sq.Eq{"gtfs_stops.id": ids})
		// AND sts.feed_version_id IN (select id from fvids)
	if where != nil {
		if where.StartTime != nil {
			q = q.Where(sq.GtOrEq{"sts.departure_time": where.StartTime})
		}
		if where.EndTime != nil {
			q = q.Where(sq.LtOrEq{"sts.departure_time": where.EndTime})
		}
	}
	return q
}
