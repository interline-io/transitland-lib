package dbfinder

import (
	"context"
	"fmt"
	"sort"
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
		return nil, logErr(ctx, err)
	}
	expandTripServiceDates(ents, tripDates)
	return ents, nil
}

func (f *Finder) TripsByIDs(ctx context.Context, ids []int) ([]*model.Trip, []error) {
	ents, err := f.FindTrips(ctx, nil, nil, ids, nil)
	if err != nil {
		return nil, logExtendErr(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.Trip) int { return ent.ID }), nil
}

func (f *Finder) FrequenciesByTripIDs(ctx context.Context, limit *int, keys []int) ([][]*model.Frequency, error) {
	var ents []*model.Frequency
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			quickSelect("gtfs_frequencies", limit, nil, nil),
			"gtfs_trips",
			"id",
			"gtfs_frequencies",
			"trip_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.Frequency) int { return ent.FeedVersionID }), err
}

// Replaces a date outside the feed version's service window with the same
// weekday in its fallback week.
func mapIntoServiceWindow(s time.Time, fvsw *model.ServiceWindow) *tt.Date {
	loc := fvsw.NowLocal.Location()
	// Compared in the feed version's timezone. The window's dates are local
	// midnight and a date scalar parses as midnight UTC, so comparing the two
	// as given puts the first or last day of the window outside it.
	local := tzTruncate(s, loc)
	if !local.Val.Before(fvsw.StartDate) && !local.Val.After(fvsw.EndDate) {
		return local
	}
	dow := int(local.Val.Weekday()) - 1
	if dow < 0 {
		dow = 6
	}
	return tzTruncate(fvsw.FallbackWeek.AddDate(0, 0, dow), loc)
}

// tripDate is a service date to match in the database and the dates it is
// reported as. They differ under use_service_window, which relocates a date
// into the fallback week; several requested dates can land on the same day.
type tripDate struct {
	query  time.Time
	report []time.Time
}

// Resolve a trip filter's dates into the service dates to match. Wall calendar
// `dates` also reach back DepartureLookbackDays.
func resolveTripDates(where *model.TripFilter, fvsw *model.ServiceWindow) []tripDate {
	if where == nil {
		return nil
	}
	src, lookback := where.ServiceDates, 0
	if len(where.Dates) > 0 {
		src, lookback = where.Dates, DepartureLookbackDays
	}
	var requested []time.Time
	for _, d := range src {
		if d != nil {
			requested = append(requested, d.Val)
		}
	}
	var out []tripDate
	at := map[string]int{}
	for _, report := range calendarServiceDates(requested, lookback) {
		query := report
		if nilOr(where.UseServiceWindow, false) && fvsw != nil {
			query = mapIntoServiceWindow(report, fvsw).Val
		}
		key := query.Format("2006-01-02")
		i, ok := at[key]
		if !ok {
			i = len(out)
			at[key] = i
			out = append(out, tripDate{query: query})
		}
		out[i].report = append(out[i].report, report)
	}
	return out
}

// Split the aggregated service_dates column into the per-trip list the API
// returns, relabelled as the dates the caller asked about.
func expandTripServiceDates(ents []*model.Trip, dates []tripDate) {
	if len(dates) == 0 {
		return
	}
	report := map[string][]time.Time{}
	for _, d := range dates {
		report[d.query.Format("2006-01-02")] = d.report
	}
	for _, ent := range ents {
		if !ent.ServiceDatesAgg.Valid || ent.ServiceDatesAgg.Val == "" {
			continue
		}
		var matched []time.Time
		for _, s := range strings.Split(ent.ServiceDatesAgg.Val, ",") {
			matched = append(matched, report[s]...)
		}
		sort.Slice(matched, func(i, j int) bool { return matched[i].Before(matched[j]) })
		for _, m := range matched {
			d := tt.NewDate(m)
			ent.ServiceDates = append(ent.ServiceDates, &d)
		}
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
			return nil, err
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
			return nil, err
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
			return nil, err
		}
		expandTripServiceDates(q, tripDates)
		ents = append(ents, q...)
	}
	return arrangeGroup(keys, ents, func(ent *model.Trip) int { return ent.FeedVersionID }), nil
}

// Returns the query and how each matched service date should be reported.
func tripSelect(limit *int, after *model.Cursor, ids []int, active bool, permFilter *model.PermFilter, where *model.TripFilter, fvsw *model.ServiceWindow) (sq.SelectBuilder, []tripDate, error) {
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
	var serviceDate *tt.Date
	if where != nil {
		serviceDate = where.ServiceDate
		if fvsw != nil {
			if where.RelativeDate != nil {
				s, err := tt.RelativeDate(fvsw.NowLocal, kebabize(string(*where.RelativeDate)))
				if err != nil {
					return q, nil, fmt.Errorf("invalid relative_date %q: %w", *where.RelativeDate, err)
				}
				serviceDate = tzTruncate(s, fvsw.NowLocal.Location())
			}
			// Guarded: use_service_window with no date at all is a valid filter.
			if nilOr(where.UseServiceWindow, false) && serviceDate != nil {
				serviceDate = mapIntoServiceWindow(serviceDate.Val, fvsw)
			}
		}
	}
	tripDates := resolveTripDates(where, fvsw)

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
		if len(tripDates) > 0 {
			// One row per trip carrying the dates it runs, instead of one query
			// per date. Aggregated as text to ride back on a single column.
			var dates []string
			for _, d := range tripDates {
				dates = append(dates, d.query.Format("2006-01-02"))
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
		} else if serviceDate != nil {
			sd := serviceDate.Val
			q = q.JoinClause(`
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
		// Handle license filtering
		q = licenseFilter(where.License, q)
	}
	if active {
		q = q.Join("feed_states on feed_states.materialized_feed_version_id = gtfs_trips.feed_version_id")
	}
	if len(ids) > 0 {
		q = q.Where(In("gtfs_trips.id", ids))
	}

	// Handle cursor
	if after != nil && after.Valid && after.ID > 0 {
		if after.FeedVersionID == 0 {
			q = q.Where(sq.Expr("(gtfs_trips.feed_version_id, gtfs_trips.id) > (select feed_version_id,id from gtfs_trips where id = ?)", after.ID))
		} else {
			q = q.Where(sq.Expr("(gtfs_trips.feed_version_id, gtfs_trips.id) > (?,?)", after.FeedVersionID, after.ID))
		}
	}

	// Handle permissions
	q = joinImported(q)
	q = pfJoinCheckFv(q, permFilter)
	return q, tripDates, nil
}
