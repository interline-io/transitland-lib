package dbfinder

import (
	"context"
	"strings"
	"time"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
)

func (f *Finder) FindTrips(ctx context.Context, limit *int, after *model.Cursor, ids []int, where *model.TripFilter) ([]*model.Trip, error) {
	var ents []*model.Trip
	active := true
	if len(ids) > 0 || (where != nil && where.FeedVersionSha1 != nil) || (where != nil && len(where.RouteIds) > 0) {
		active = false
	}
	q, tripDates, err := tripSelect(limit, after, ids, active, f.PermFilter(ctx), where, nil)
	if err != nil {
		return nil, err
	}
	if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
		return nil, queryErr(ctx, err)
	}
	expandTripServiceDates(ents, tripDates)
	return ents, nil
}

func (f *Finder) TripsByIDs(ctx context.Context, ids []int) ([]*model.Trip, []error) {
	ents, err := f.FindTrips(ctx, nil, nil, ids, nil)
	if err != nil {
		return nil, queryErrs(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.Trip) int { return ent.ID }), nil
}

// TripsByFeedVersionTripIDs looks up trips by GTFS trip_id within a feed version.
func (f *Finder) TripsByFeedVersionTripIDs(ctx context.Context, keys []model.FVEntityID) ([]*model.Trip, []error) {
	var ents []*model.Trip
	for fvid, tripIds := range groupFVEntityIDs(keys) {
		q, _, err := tripSelect(nil, nil, nil, false, f.PermFilter(ctx), nil, nil)
		if err != nil {
			return nil, queryErrs(ctx, len(keys), err)
		}
		q = q.Where(sq.Eq{"gtfs_trips.feed_version_id": fvid}).Where(In("gtfs_trips.trip_id", tripIds))
		var group []*model.Trip
		if err := dbutil.Select(ctx, f.db, q, &group); err != nil {
			return nil, queryErrs(ctx, len(keys), err)
		}
		ents = append(ents, group...)
	}
	return arrangeBy(keys, ents, func(ent *model.Trip) model.FVEntityID {
		return model.FVEntityID{FeedVersionID: ent.FeedVersionID, EntityID: ent.TripID.Val}
	}), nil
}

func (f *Finder) FrequenciesByTripIDs(ctx context.Context, limit *int, keys []int) ([][]*model.Frequency, error) {
	var ents []*model.Frequency
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			frequencySelect(limit, nil, f.PermFilter(ctx), nil),
			"gtfs_trips",
			"id",
			"gtfs_frequencies",
			"trip_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.Frequency) int { return ent.TripID.Int() }), queryErr(ctx, err)
}

// FrequenciesByFeedVersionIDs returns the frequencies of each feed version.
func (f *Finder) FrequenciesByFeedVersionIDs(ctx context.Context, limit *int, after *model.Cursor, where *model.FrequencyFilter, keys []int) ([][]*model.Frequency, error) {
	var ents []*model.Frequency
	q := lateralWrap(frequencySelect(limit, after, f.PermFilter(ctx), where), "feed_versions", "id", "gtfs_frequencies", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.Frequency) int { return ent.FeedVersionID }), queryErr(ctx, err)
}

func frequencySelect(limit *int, after *model.Cursor, permFilter *model.PermFilter, where *model.FrequencyFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_frequencies.id",
		"gtfs_frequencies.feed_version_id",
		"gtfs_frequencies.trip_id",
		"gtfs_frequencies.start_time",
		"gtfs_frequencies.end_time",
		"gtfs_frequencies.headway_secs",
		"gtfs_frequencies.exact_times",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_frequencies").
		Join("feed_versions ON feed_versions.id = gtfs_frequencies.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
	if after != nil && after.Valid && after.ID > 0 {
		q = q.Where(sq.Gt{"gtfs_frequencies.id": after.ID})
	}
	if where != nil {
		if len(where.Ids) > 0 {
			q = q.Where(In("gtfs_frequencies.id", where.Ids))
		}
	}
	q = joinImported(q)
	q = pfJoinCheckFv(q, permFilter)
	return q.OrderBy("gtfs_frequencies.id ASC").Limit(finderCheckLimit(limit))
}

// Restrict a query over gtfs_trips to the trips running on a service date.
//
// The join is inner, so a trip whose calendar does not cover the date drops out.
// LIMIT 1 keeps a service with several calendar_dates rows from multiplying the
// trip, which is what makes the result safe to aggregate over.
func serviceDateLateral(q sq.SelectBuilder, serviceDate tt.Date) sq.SelectBuilder {
	sd := serviceDate.Val
	return q.JoinClause(`
	join lateral (
		select gc.id
		from gtfs_calendars gc
		left join gtfs_calendar_dates gcda on gcda.service_id = gc.id and gcda.exception_type = 1 and gcda.date = ?::date
		left join gtfs_calendar_dates gcdb on gcdb.service_id = gc.id and gcdb.exception_type = 2 and gcdb.date = ?::date
		where
			gc.id = gtfs_trips.service_id
			AND ((
				gc.start_date <= ?::date AND gc.end_date >= ?::date
				AND (CASE EXTRACT(isodow FROM ?::date)
				WHEN 1 THEN monday = 1
				WHEN 2 THEN tuesday = 1
				WHEN 3 THEN wednesday = 1
				WHEN 4 THEN thursday = 1
				WHEN 5 THEN friday = 1
				WHEN 6 THEN saturday = 1
				WHEN 7 THEN sunday = 1
				END)
			) OR gcda.date IS NOT NULL)
			AND gcdb.date is null
		LIMIT 1
	) gc on true
	`, sd, sd, sd, sd, sd)
}

// Record each trip's runs from the aggregated service_dates column: the query
// dates its service ran on.
func expandTripServiceDates(ents []*model.Trip, dates []model.TripDate) {
	for _, ent := range ents {
		ran := map[string]bool{}
		if ent.ServiceDatesAgg.Valid && ent.ServiceDatesAgg.Val != "" {
			for _, s := range strings.Split(ent.ServiceDatesAgg.Val, ",") {
				ran[s] = true
			}
		}
		model.SetTripRunDates(ent, dates, func(query time.Time) bool { return ran[query.Format("2006-01-02")] })
	}
}

func (f *Finder) TripsByRouteIDs(ctx context.Context, limit *int, where *model.TripFilter, keys []model.FVPair) ([][]*model.Trip, error) {
	var ents []*model.Trip
	// Group by fvid
	groups := map[int][]int{}
	for _, key := range keys {
		groups[key.FeedVersionID] = append(groups[key.FeedVersionID], key.EntityID)
	}
	for fvid, entityIds := range groups {
		fvsw, err := f.FindFeedVersionServiceWindow(ctx, fvid)
		if err != nil {
			return nil, err
		}
		inner, tripDates, err := tripSelect(limit, nil, nil, false, f.PermFilter(ctx), where, fvsw)
		if err != nil {
			return nil, err
		}
		// Always in the route's feed version; see lateralWrap.
		inner = inner.Where("gtfs_trips.feed_version_id = out.feed_version_id")
		var q []*model.Trip
		if err := dbutil.Select(ctx,
			f.db,
			lateralWrap(
				inner,
				"gtfs_routes",
				"id",
				"gtfs_trips",
				"route_id",
				entityIds,
			),
			&q,
		); err != nil {
			return nil, queryErr(ctx, err)
		}
		// Per feed version: each resolves the requested dates against its own
		// service window.
		expandTripServiceDates(q, tripDates)
		ents = append(ents, q...)
	}
	return arrangeGroup(keys, ents, func(ent *model.Trip) model.FVPair {
		return model.FVPair{FeedVersionID: ent.FeedVersionID, EntityID: ent.RouteID.Int()}
	}), nil
}

func (f *Finder) TripsByShapeIDs(ctx context.Context, limit *int, where *model.TripFilter, keys []model.FVPair) ([][]*model.Trip, error) {
	var ents []*model.Trip
	// Group by fvid so the per-feed-version service window can be applied.
	groups := map[int][]int{}
	for _, key := range keys {
		groups[key.FeedVersionID] = append(groups[key.FeedVersionID], key.EntityID)
	}
	for fvid, entityIds := range groups {
		fvsw, err := f.FindFeedVersionServiceWindow(ctx, fvid)
		if err != nil {
			return nil, err
		}
		inner, tripDates, err := tripSelect(limit, nil, nil, false, f.PermFilter(ctx), where, fvsw)
		if err != nil {
			return nil, err
		}
		// Always in the shape's feed version; see lateralWrap.
		inner = inner.Where("gtfs_trips.feed_version_id = out.feed_version_id")
		var q []*model.Trip
		if err := dbutil.Select(ctx,
			f.db,
			lateralWrap(
				inner,
				"gtfs_shapes",
				"id",
				"gtfs_trips",
				"shape_id",
				entityIds,
			),
			&q,
		); err != nil {
			return nil, queryErr(ctx, err)
		}
		expandTripServiceDates(q, tripDates)
		ents = append(ents, q...)
	}
	return arrangeGroup(keys, ents, func(ent *model.Trip) model.FVPair {
		return model.FVPair{FeedVersionID: ent.FeedVersionID, EntityID: ent.ShapeID.Int()}
	}), nil
}

func (f *Finder) TripsByFeedVersionIDs(ctx context.Context, limit *int, where *model.TripFilter, keys []int) ([][]*model.Trip, error) {
	var ents []*model.Trip
	for _, fvid := range keys {
		fvsw, err := f.FindFeedVersionServiceWindow(ctx, fvid)
		if err != nil {
			return nil, err
		}
		inner, tripDates, err := tripSelect(limit, nil, nil, false, f.PermFilter(ctx), where, fvsw)
		if err != nil {
			return nil, err
		}
		var q []*model.Trip
		err = dbutil.Select(ctx,
			f.db,
			lateralWrap(
				inner,
				"feed_versions",
				"id",
				"gtfs_trips",
				"feed_version_id",
				[]int{fvid},
			),
			&q,
		)
		if err != nil {
			return nil, queryErr(ctx, err)
		}
		expandTripServiceDates(q, tripDates)
		ents = append(ents, q...)
	}
	return arrangeGroup(keys, ents, func(ent *model.Trip) int { return ent.FeedVersionID }), nil
}

// Returns the query and how each matched service date should be reported.
func tripSelect(limit *int, after *model.Cursor, ids []int, active bool, permFilter *model.PermFilter, where *model.TripFilter, fvsw *model.ServiceWindow) (sq.SelectBuilder, []model.TripDate, error) {
	q := sq.StatementBuilder.Select(
		"gtfs_trips.id",
		"gtfs_trips.feed_version_id",
		"gtfs_trips.route_id",
		"gtfs_trips.service_id",
		"gtfs_trips.shape_id",
		"gtfs_trips.trip_id",
		"gtfs_trips.trip_headsign",
		"gtfs_trips.trip_short_name",
		"gtfs_trips.direction_id",
		"gtfs_trips.block_id",
		"gtfs_trips.wheelchair_accessible",
		"gtfs_trips.bikes_allowed",
		"gtfs_trips.cars_allowed",
		"gtfs_trips.stop_pattern_id",
		"gtfs_trips.journey_pattern_id",
		"gtfs_trips.journey_pattern_offset",
		"gtfs_trips.safe_duration_factor",
		"gtfs_trips.safe_duration_offset",
		"current_feeds.id AS feed_id",
		"current_feeds.onestop_id AS feed_onestop_id",
		"feed_versions.sha1 AS feed_version_sha1",
	).
		From("gtfs_trips").
		Join("feed_versions ON feed_versions.id = gtfs_trips.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id").
		OrderBy("gtfs_trips.feed_version_id,gtfs_trips.id").
		Limit(finderCheckLimit(limit))

	// Resolved into a local: `where` is reused for every feed version.
	tripDates, err := model.ResolveTripDates(where, fvsw)
	if err != nil {
		return q, nil, err
	}

	// Process other parameters
	if where != nil {
		if where.StopPatternID != nil {
			q = q.Where(sq.Eq{"stop_pattern_id": where.StopPatternID})
		}
		if len(where.RouteOnestopIds) > 0 {
			q = q.
				Join("gtfs_routes on gtfs_routes.id = gtfs_trips.route_id").
				Join("feed_version_route_onestop_ids on feed_version_route_onestop_ids.entity_id = gtfs_routes.route_id and feed_version_route_onestop_ids.feed_version_id = gtfs_trips.feed_version_id")
			q = q.Where(In("feed_version_route_onestop_ids.onestop_id", where.RouteOnestopIds))
		}
		if where.FeedVersionSha1 != nil {
			q = q.Where("feed_versions.id = (select id from feed_versions where sha1 = ? limit 1)", *where.FeedVersionSha1)
		}
		if where.FeedOnestopID != nil {
			q = q.Where(sq.Eq{"current_feeds.onestop_id": *where.FeedOnestopID})
		}
		if where.TripID != nil {
			q = q.Where(sq.Eq{"gtfs_trips.trip_id": *where.TripID})
		}
		if len(where.RouteIds) > 0 {
			q = q.Where(In("gtfs_trips.route_id", where.RouteIds))
		}
		if len(tripDates) == 1 && tripDates[0].Single {
			q = serviceDateLateral(q, tt.NewDate(tripDates[0].Query))
		} else if len(tripDates) > 0 {
			// One row per trip carrying the dates it runs, instead of one query
			// per date. Aggregated as text to ride back on a single column.
			var dates []string
			for _, d := range tripDates {
				dates = append(dates, d.Query.Format("2006-01-02"))
			}
			q = q.Column("svc.service_dates_agg").JoinClause(`
			join lateral (
				select string_agg(to_char(d.day, 'YYYY-MM-DD'), ',' order by d.day) as service_dates_agg
				from unnest(string_to_array(?, ',')::date[]) as d(day)
				join gtfs_calendars gc on gc.id = gtfs_trips.service_id
				left join gtfs_calendar_dates gcda on gcda.service_id = gc.id and gcda.exception_type = 1 and gcda.date = d.day
				left join gtfs_calendar_dates gcdb on gcdb.service_id = gc.id and gcdb.exception_type = 2 and gcdb.date = d.day
				where ((
						gc.start_date <= d.day AND gc.end_date >= d.day
						AND (CASE EXTRACT(isodow FROM d.day)
						WHEN 1 THEN monday = 1
						WHEN 2 THEN tuesday = 1
						WHEN 3 THEN wednesday = 1
						WHEN 4 THEN thursday = 1
						WHEN 5 THEN friday = 1
						WHEN 6 THEN saturday = 1
						WHEN 7 THEN sunday = 1
						END)
					) OR gcda.date IS NOT NULL)
					AND gcdb.date is null
			) svc on true
			`, strings.Join(dates, ",")).Where("svc.service_dates_agg is not null")
		}
		// Handle license filtering
		q = licenseFilter(where.License, q)
	}
	if active {
		q = q.Join("feed_states on feed_states.materialized_feed_version_id = gtfs_trips.feed_version_id")
	}
	if len(ids) > 0 {
		q = q.Where(In("gtfs_trips.id", ids))
	}

	// Handle cursor. A request can name any row, so the cursor's row goes through gatedRows.
	if after != nil && after.Valid && after.ID > 0 {
		if after.FeedVersionID == 0 {
			cursorRow := gatedRows("gtfs_trips", permFilter).
				Columns("gtfs_trips.feed_version_id", "gtfs_trips.id").
				Where(sq.Eq{"gtfs_trips.id": after.ID})
			q = q.Where(sq.Expr("(gtfs_trips.feed_version_id, gtfs_trips.id) > (?)", cursorRow))
		} else {
			q = q.Where(sq.Expr("(gtfs_trips.feed_version_id, gtfs_trips.id) > (?,?)", after.FeedVersionID, after.ID))
		}
	}

	// Handle permissions
	q = joinImported(q)
	q = pfJoinCheckFv(q, permFilter)
	return q, tripDates, nil
}
