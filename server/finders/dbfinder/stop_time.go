package dbfinder

import (
	"context"
	"time"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
)

func (f *Finder) StopTimesByTripIDs(ctx context.Context, limit *int, where *model.TripStopTimeFilter, keys []model.FVPair) ([][]*model.StopTime, error) {
	var ents []*model.StopTime
	err := dbutil.Select(ctx,
		f.db,
		stopTimeSelect(keys, stopTimeEntityTrip, where),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.StopTime) model.FVPair {
		return model.FVPair{FeedVersionID: ent.FeedVersionID, EntityID: ent.TripID.Int()}
	}), err
}

func (f *Finder) StopTimesByStopIDs(ctx context.Context, limit *int, where *model.StopTimeFilter, keys []model.FVPair) ([][]*model.StopTime, error) {
	ents, err := f.stopTimesByEntityIDs(ctx, stopTimeEntityStop, where, keys)
	if err != nil {
		return nil, err
	}
	return arrangeGroup(keys, ents, func(ent *model.StopTime) model.FVPair {
		return model.FVPair{FeedVersionID: ent.FeedVersionID, EntityID: ent.StopID.Int()}
	}), nil
}

// FlexStopTimesByTripIDs returns flex stop times for the given trip IDs.
// Since FlexStopTime is an alias for StopTime, this just delegates to StopTimesByTripIDs.
func (f *Finder) FlexStopTimesByTripIDs(ctx context.Context, limit *int, where *model.TripStopTimeFilter, keys []model.FVPair) ([][]*model.FlexStopTime, error) {
	return f.StopTimesByTripIDs(ctx, limit, where, keys)
}

// FlexStopTimesByStopIDs returns flex stop times for the given stop IDs.
// Since FlexStopTime is an alias for StopTime, this just delegates to StopTimesByStopIDs.
func (f *Finder) FlexStopTimesByStopIDs(ctx context.Context, limit *int, where *model.StopTimeFilter, keys []model.FVPair) ([][]*model.FlexStopTime, error) {
	return f.StopTimesByStopIDs(ctx, limit, where, keys)
}

// FlexStopTimesByLocationIDs returns flex stop times for the given location IDs.
// Used by the Location resolver to get stop_times for flex service areas.
func (f *Finder) FlexStopTimesByLocationIDs(ctx context.Context, limit *int, where *model.StopTimeFilter, keys []model.FVPair) ([][]*model.FlexStopTime, error) {
	ents, err := f.stopTimesByEntityIDs(ctx, stopTimeEntityLocation, where, keys)
	if err != nil {
		return nil, err
	}
	return arrangeGroup(keys, ents, func(ent *model.StopTime) model.FVPair {
		return model.FVPair{FeedVersionID: ent.FeedVersionID, EntityID: ent.LocationID.Int()}
	}), nil
}

// FlexStopTimesByLocationGroupIDs returns flex stop times for the given location group IDs.
// Used by the LocationGroup resolver to get stop_times for flex service location groups.
func (f *Finder) FlexStopTimesByLocationGroupIDs(ctx context.Context, limit *int, where *model.StopTimeFilter, keys []model.FVPair) ([][]*model.FlexStopTime, error) {
	ents, err := f.stopTimesByEntityIDs(ctx, stopTimeEntityLocationGroup, where, keys)
	if err != nil {
		return nil, err
	}
	return arrangeGroup(keys, ents, func(ent *model.StopTime) model.FVPair {
		return model.FVPair{FeedVersionID: ent.FeedVersionID, EntityID: ent.LocationGroupID.Int()}
	}), nil
}

// stopTimesByEntityIDs is the internal method that fetches stop_times for any entity type.
// Public methods call this and arrange results by the appropriate grouping key.
func (f *Finder) stopTimesByEntityIDs(ctx context.Context, entityType stopTimeEntityType, where *model.StopTimeFilter, keys []model.FVPair) ([]*model.StopTime, error) {
	pairGroups := map[int][]model.FVPair{}
	for _, v := range keys {
		pairGroups[v.FeedVersionID] = append(pairGroups[v.FeedVersionID], v)
	}
	var ents []*model.StopTime
	for fvid, entityPairs := range pairGroups {
		fvsw, err := f.FindFeedVersionServiceWindow(ctx, fvid)
		if err != nil {
			return nil, err
		}
		// Run separate queries for each possible service day
		for _, w := range stopTimeFilterExpand(where, fvsw) {
			var serviceDate *tt.Date
			if w != nil && w.ServiceDate != nil {
				serviceDate = w.ServiceDate
			}
			var sts []*model.StopTime
			var q sq.SelectBuilder
			if serviceDate != nil {
				// Get stop_times on a specified day
				var entityKeys []int
				for _, k := range entityPairs {
					entityKeys = append(entityKeys, k.EntityID)
				}
				q = stopDeparturesSelect(fvid, entityKeys, entityType, w)
			} else {
				// Otherwise get all stop_times for entity
				q = stopTimeSelect(entityPairs, entityType, nil)
			}
			// Run query
			if err := dbutil.Select(ctx, f.db, q, &sts); err != nil {
				return nil, err
			}
			// Set service date based on StopTimeFilter, and adjust calendar date if needed
			if serviceDate != nil {
				for _, ent := range sts {
					ent.ServiceDate.Set(serviceDate.Val)
					if ent.ArrivalTime.Val > 24*60*60 {
						ent.Date.Set(serviceDate.Val.AddDate(0, 0, 1))
					} else {
						ent.Date.Set(serviceDate.Val)
					}
				}
			}
			ents = append(ents, sts...)
		}
	}
	return ents, nil
}

// stopTimeEntityType specifies which entity type to filter stop_times by
type stopTimeEntityType int

const (
	stopTimeEntityTrip stopTimeEntityType = iota
	stopTimeEntityStop
	stopTimeEntityLocation
	stopTimeEntityLocationGroup
)

func stopTimeSelect(pairs []model.FVPair, entityType stopTimeEntityType, where *model.TripStopTimeFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_trips.journey_pattern_id",
		"gtfs_trips.journey_pattern_offset",
		"gtfs_trips.id AS trip_id",
		"gtfs_trips.feed_version_id",
		"sts.stop_id",
		"sts.location_id",
		"sts.location_group_id",
		"sts.arrival_time + gtfs_trips.journey_pattern_offset AS arrival_time",
		"sts.departure_time + gtfs_trips.journey_pattern_offset AS departure_time",
		"sts.stop_sequence",
		"sts.shape_dist_traveled",
		"sts.pickup_type",
		"sts.drop_off_type",
		"sts.timepoint",
		"sts.interpolated",
		"sts.stop_headsign",
		"sts.continuous_pickup",
		"sts.continuous_drop_off",
		"sts.start_pickup_drop_off_window",
		"sts.end_pickup_drop_off_window",
		"sts.pickup_booking_rule_id",
		"sts.drop_off_booking_rule_id",
	).
		From("gtfs_trips").
		Join("feed_versions on feed_versions.id = gtfs_trips.feed_version_id").
		Join("current_feeds on current_feeds.id = feed_versions.feed_id").
		Join("gtfs_trips t2 ON t2.trip_id::text = gtfs_trips.journey_pattern_id AND gtfs_trips.feed_version_id = t2.feed_version_id").
		Join("gtfs_stop_times sts ON sts.trip_id = t2.id AND sts.feed_version_id = t2.feed_version_id").
		OrderBy("sts.stop_sequence, sts.arrival_time")

	if where != nil {
		if where.Start != nil {
			q = q.Where(sq.GtOrEq{"sts.departure_time + gtfs_trips.journey_pattern_offset": where.Start.Int()})
		}
		if where.End != nil {
			q = q.Where(sq.LtOrEq{"sts.arrival_time + gtfs_trips.journey_pattern_offset": where.End.Int()})
		}
	}
	if len(pairs) > 0 {
		eids, fvids := pairKeys(pairs)
		q = q.Where(In("sts.feed_version_id", fvids))
		switch entityType {
		case stopTimeEntityTrip:
			q = q.Where(In("gtfs_trips.id", eids), In("gtfs_trips.feed_version_id", fvids))
		case stopTimeEntityStop:
			q = q.Where(In("sts.stop_id", eids))
		case stopTimeEntityLocation:
			q = q.Where(In("sts.location_id", eids))
		case stopTimeEntityLocationGroup:
			q = q.Where(In("sts.location_group_id", eids))
		}
	}
	return q
}

// activeServicesCTE returns a CTE that finds all active service IDs for a given date and feed version.
// This is used by both stopDeparturesSelect and locationDeparturesSelect.
func activeServicesCTE(fvid int, serviceDate time.Time) sq.CTE {
	return sq.CTE{
		Alias:        "active_services",
		Materialized: true,
		Expression: sq.Expr(`
		SELECT id
		FROM gtfs_calendars
		WHERE start_date <= ?
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
			AND feed_version_id = ?
		UNION
		SELECT service_id as id
		FROM gtfs_calendar_dates
		WHERE date = ? AND exception_type = 1 AND feed_version_id = ?
		EXCEPT
		SELECT service_id as id 
		FROM gtfs_calendar_dates
		WHERE date = ? AND exception_type = 2 AND feed_version_id = ?`,
			serviceDate, serviceDate, serviceDate, fvid, serviceDate, fvid, serviceDate, fvid),
	}
}

// stopDeparturesSelect returns stop_times for a specific date.
// Filters by the entity IDs based on entityType (stop_id, location_id, or location_group_id).
func stopDeparturesSelect(fvid int, entityIDs []int, entityType stopTimeEntityType, where *model.StopTimeFilter) sq.SelectBuilder {
	// Where must already be set for local service date and timezone
	serviceDate := time.Now()
	if where != nil && where.ServiceDate != nil {
		serviceDate = where.ServiceDate.Val
	}

	// Build main query with CTEs
	q := sq.StatementBuilder.Select(
		"gtfs_trips.journey_pattern_id",
		"gtfs_trips.journey_pattern_offset",
		"gtfs_trips.id AS trip_id",
		"gtfs_trips.feed_version_id",
		"sts.stop_id",
		"sts.location_id",
		"sts.location_group_id",
		"sts.arrival_time_freq AS arrival_time",
		"sts.departure_time_freq AS departure_time",
		"sts.stop_sequence",
		"sts.shape_dist_traveled",
		"sts.pickup_type",
		"sts.drop_off_type",
		"sts.timepoint",
		"sts.interpolated",
		"sts.stop_headsign",
		"sts.continuous_pickup",
		"sts.continuous_drop_off",
		"sts.start_pickup_drop_off_window",
		"sts.end_pickup_drop_off_window",
		"sts.pickup_booking_rule_id",
		"sts.drop_off_booking_rule_id",
	).
		WithCTE(activeServicesCTE(fvid, serviceDate)).
		From("gtfs_trips").
		Join("active_services gc on gc.id = gtfs_trips.service_id").
		Join("gtfs_trips base_trip ON base_trip.trip_id::text = gtfs_trips.journey_pattern_id AND gtfs_trips.feed_version_id = base_trip.feed_version_id").
		Join("feed_versions on feed_versions.id = gtfs_trips.feed_version_id").
		JoinClause(`left join lateral (
			select
				generate_series(start_time, end_time, headway_secs) freq_start
			from gtfs_frequencies
			where gtfs_frequencies.trip_id = gtfs_trips.id
			) freq on true`).
		JoinClause(`join lateral (
			select 
				min(sts2.departure_time) first_departure_time,
				min(sts2.stop_sequence) stop_sequence_min, 
				max(sts2.stop_sequence) stop_sequence_max 
			from gtfs_stop_times sts2 
			where sts2.trip_id = base_trip.id and sts2.feed_version_id = base_trip.feed_version_id
			) trip_stop_sequence on true`).
		JoinClause(`join lateral (
			select 
				sts.*,
				sts.arrival_time + gtfs_trips.journey_pattern_offset + coalesce(
					- trip_stop_sequence.first_departure_time + freq.freq_start,
					0
				) AS arrival_time_freq,
				sts.departure_time + gtfs_trips.journey_pattern_offset + coalesce(
					- trip_stop_sequence.first_departure_time + freq.freq_start,
					0
				) AS departure_time_freq
			from gtfs_stop_times sts
			where sts.trip_id = base_trip.id and sts.feed_version_id = base_trip.feed_version_id		
			) sts on true`).
		Where(sq.Eq{"sts.feed_version_id": fvid}).
		OrderBy("sts.departure_time_freq", "sts.trip_id") // base + offset

	// Filter by entity type
	if len(entityIDs) > 0 {
		switch entityType {
		case stopTimeEntityStop:
			q = q.Where(In("sts.stop_id", entityIDs))
		case stopTimeEntityLocation:
			q = q.Where(In("sts.location_id", entityIDs))
		case stopTimeEntityLocationGroup:
			q = q.Where(In("sts.location_group_id", entityIDs))
		}
	}

	if where != nil {
		if where.ExcludeFirst != nil && *where.ExcludeFirst {
			q = q.Where("sts.stop_sequence > trip_stop_sequence.stop_sequence_min")
		}
		if where.ExcludeLast != nil && *where.ExcludeLast {
			q = q.Where("sts.stop_sequence < trip_stop_sequence.stop_sequence_max")
		}
		if len(where.RouteOnestopIds) > 0 {
			if where.AllowPreviousRouteOnestopIds != nil && *where.AllowPreviousRouteOnestopIds {
				// Use CTE for route lookup optimization
				sub := sq.StatementBuilder.
					Select("feed_version_route_onestop_ids.entity_id", "feed_versions.feed_id").
					Distinct().Options("on (feed_version_route_onestop_ids.entity_id, feed_versions.feed_id)").
					From("feed_version_route_onestop_ids").
					Join("feed_versions on feed_versions.id = feed_version_route_onestop_ids.feed_version_id").
					Where(In("feed_version_route_onestop_ids.onestop_id", where.RouteOnestopIds)).
					OrderBy("feed_version_route_onestop_ids.entity_id, feed_versions.feed_id, feed_versions.id DESC")
				routeLookupCte := sq.CTE{
					Materialized: true,
					Alias:        "route_lookup",
					Expression:   sub,
				}
				q = q.
					WithCTE(routeLookupCte).
					Join("gtfs_routes on gtfs_routes.id = gtfs_trips.route_id and gtfs_routes.feed_version_id = gtfs_trips.feed_version_id").
					Join("route_lookup tlros on tlros.entity_id = gtfs_routes.route_id and tlros.feed_id = feed_versions.feed_id")
			} else {
				q = q.
					Join("gtfs_routes on gtfs_routes.id = gtfs_trips.route_id").
					Join("feed_version_route_onestop_ids on feed_version_route_onestop_ids.entity_id = gtfs_routes.route_id and feed_version_route_onestop_ids.feed_version_id = gtfs_trips.feed_version_id").
					Where(In("feed_version_route_onestop_ids.onestop_id", where.RouteOnestopIds))

			}
		}
		// Accept either Start/End or StartTime/EndTime
		if where.Start != nil && where.Start.Valid {
			where.StartTime = ptr(where.Start.Int())
		}
		if where.End != nil && where.End.Valid {
			where.EndTime = ptr(where.End.Int())
		}
		if where.StartTime != nil {
			q = q.Where(sq.GtOrEq{"sts.departure_time_freq": *where.StartTime})
		}
		if where.EndTime != nil {
			q = q.Where(sq.LtOrEq{"sts.departure_time_freq": *where.EndTime})
		}
	}
	return q
}

func stopTimeFilterExpand(where *model.StopTimeFilter, fvsw *model.ServiceWindow) []*model.StopTimeFilter {
	// Pre-processing
	// Convert Start, End to StartTime, EndTime
	if where != nil {
		if where.Start != nil {
			where.StartTime = ptr(where.Start.Int())
			where.Start = nil
		}
		if where.End != nil {
			where.EndTime = ptr(where.End.Int())
			where.End = nil
		}
	}

	// Further processing of the StopTimeFilter
	if where != nil {
		var nowLocal time.Time
		if fvsw != nil {
			nowLocal = fvsw.NowLocal
		}
		loc := nowLocal.Location()

		// Set ServiceDate to local timezone
		// ServiceDate is a strict GTFS calendar date
		if where.ServiceDate != nil {
			where.ServiceDate = tzTruncate(where.ServiceDate.Val, loc)
		}

		// Set Date to local timezone
		if where.Date != nil {
			where.Date = tzTruncate(where.Date.Val, loc)
		}

		// Convert relative date
		if where.RelativeDate != nil {
			if s, err := tt.RelativeDate(nowLocal, kebabize(string(*where.RelativeDate))); err != nil {
				// This should always succeed because it is an enum and will be caught earlier
				// TODO: log
			} else {
				where.Date = tzTruncate(s, loc)
			}
		}

		// Convert where.Next into departure date and time window
		if where.Next != nil {
			if where.Date == nil {
				where.Date = tzTruncate(nowLocal, loc)
			}
			st := nowLocal.Hour()*3600 + nowLocal.Minute()*60 + nowLocal.Second()
			where.StartTime = ptr(st)
			where.EndTime = ptr(st + *where.Next)
		}

		// Map date into service window
		if nilOr(where.UseServiceWindow, false) && fvsw != nil {
			startDate, endDate, fallbackWeek := fvsw.StartDate, fvsw.EndDate, fvsw.FallbackWeek
			// Check if date is outside window
			if where.Date != nil {
				s := where.Date.Val
				if s.Before(startDate) || s.After(endDate) {
					dow := int(s.Weekday()) - 1
					if dow < 0 {
						dow = 6
					}
					where.Date = tzTruncate(fallbackWeek.AddDate(0, 0, dow), loc)
				}
			}
			// Repeat for ServiceDate
			if where.ServiceDate != nil {
				s := where.ServiceDate.Val
				if s.Before(startDate) || s.After(endDate) {
					dow := int(s.Weekday()) - 1
					if dow < 0 {
						dow = 6
					}
					where.ServiceDate = tzTruncate(fallbackWeek.AddDate(0, 0, dow), loc)
				}
			}
		}
	}

	// Check if we are crossing date boundaires, and split into separate service date queries
	var whereGroups []*model.StopTimeFilter
	if where != nil && where.Date != nil {
		date := where.Date
		dayStart := 0
		dayEnd := 24 * 60 * 60
		dayEndMax := 100 * 60 * 60
		whereStartTime := dayStart
		if where.StartTime != nil {
			whereStartTime = *where.StartTime
		}
		whereEndTime := dayEnd
		if where.EndTime != nil {
			whereEndTime = *where.EndTime
		}
		lookBehind := 6 * 3600
		// Query previous day
		if whereStartTime < lookBehind {
			whereCopy := *where
			whereCopy.ServiceDate = ptr(tt.NewDate(date.Val.AddDate(0, 0, -1)))
			whereCopy.StartTime = ptr(dayEnd + whereStartTime)
			whereCopy.EndTime = ptr(dayEndMax)
			whereGroups = append(whereGroups, &whereCopy)
		}
		// Query requested day
		whereCopy := *where
		whereCopy.ServiceDate = ptr(tt.NewDate(date.Val))
		whereCopy.StartTime = ptr(max(dayStart, whereStartTime))
		whereCopy.EndTime = ptr(whereEndTime)
		whereGroups = append(whereGroups, &whereCopy)
		// Query next day
		if whereEndTime > dayEnd {
			whereCopy := *where
			whereCopy.ServiceDate = ptr(tt.NewDate(date.Val.AddDate(0, 0, 1)))
			whereCopy.StartTime = ptr(dayStart)
			whereCopy.EndTime = ptr(whereEndTime - dayEnd)
			whereGroups = append(whereGroups, &whereCopy)
		}
	}

	// Default
	if len(whereGroups) == 0 {
		whereGroups = append(whereGroups, where)
	}

	return whereGroups
}

func pairKeys(spairs []model.FVPair) ([]int, []int) {
	eids := map[int]bool{}
	fvids := map[int]bool{}
	for _, v := range spairs {
		eids[v.EntityID] = true
		fvids[v.FeedVersionID] = true
	}
	var ueids []int
	for k := range eids {
		ueids = append(ueids, k)
	}
	var ufvids []int
	for k := range fvids {
		ufvids = append(ufvids, k)
	}
	return ueids, ufvids
}
