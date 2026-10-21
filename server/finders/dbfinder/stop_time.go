package dbfinder

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	sq "github.com/irees/squirrel"
)

// minExpandedHeadwaySecs is the shortest frequency headway that departures queries
// expand into runs. GTFS allows any positive headway; this protects the queries from
// feeds whose sub-minute headways would expand into millions of departures.
const minExpandedHeadwaySecs = 60

func (f *Finder) StopTimesByTripIDs(ctx context.Context, limit *int, where *model.TripStopTimeFilter, keys []model.FVPair) ([][]*model.StopTime, error) {
	var ents []*model.StopTime
	err := dbutil.Select(ctx,
		f.db,
		stopTimeSelect(keys, stopTimeEntityTrip, where),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.StopTime) model.FVPair {
		return model.FVPair{FeedVersionID: ent.FeedVersionID, EntityID: ent.TripID.Int()}
	}), queryErr(ctx, err)
}

func (f *Finder) StopTimesByStopIDs(ctx context.Context, limit *int, where *model.StopTimeFilter, keys []model.FVPair) ([][]*model.StopTime, error) {
	ents, err := f.stopTimesByEntityIDs(ctx, stopTimeEntityStop, limit, where, keys)
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
	ents, err := f.stopTimesByEntityIDs(ctx, stopTimeEntityLocation, limit, where, keys)
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
	ents, err := f.stopTimesByEntityIDs(ctx, stopTimeEntityLocationGroup, limit, where, keys)
	if err != nil {
		return nil, err
	}
	return arrangeGroup(keys, ents, func(ent *model.StopTime) model.FVPair {
		return model.FVPair{FeedVersionID: ent.FeedVersionID, EntityID: ent.LocationGroupID.Int()}
	}), nil
}

// stopTimesByEntityIDs is the internal method that fetches stop_times for any entity type.
// Public methods call this and arrange results by the appropriate grouping key.
func (f *Finder) stopTimesByEntityIDs(ctx context.Context, entityType stopTimeEntityType, limit *int, where *model.StopTimeFilter, keys []model.FVPair) ([]*model.StopTime, error) {
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
		var fvEnts []*model.StopTime
		ranked := false
		days, err := model.ExpandStopTimeFilter(where, fvsw)
		if err != nil {
			return nil, err
		}
		for _, day := range days {
			w := day.Where
			var sts []*model.StopTime
			var q sq.SelectBuilder
			if day.Report != nil {
				// Get stop_times on a specified day
				var entityKeys []int
				for _, k := range entityPairs {
					entityKeys = append(entityKeys, k.EntityID)
				}
				if model.ForContext(ctx).UseMaterializedDepartures {
					q = stopDeparturesMaterializedSelect(fvid, entityKeys, entityType, limit, w)
				} else {
					q = stopDeparturesSelect(fvid, entityKeys, entityType, limit, w)
				}
			} else {
				// Otherwise get all stop_times for entity
				q = stopTimeSelect(entityPairs, entityType, nil)
			}
			// Run query
			if err := dbutil.Select(ctx, f.db, q, &sts); err != nil {
				return nil, queryErr(ctx, err)
			}
			// Report the requested service date
			if day.Report != nil {
				ranked = true
				for _, ent := range sts {
					ent.SetServiceDate(*day.Report)
				}
			}
			fvEnts = append(fvEnts, sts...)
		}
		// Each service day's departures query keeps each route's first stop
		// times, so a window across days can hold more than RouteLimit of them.
		if ranked && where != nil && where.RouteLimit != nil {
			fvEnts = firstForRoutes(fvEnts, *where.RouteLimit)
		}
		ents = append(ents, fvEnts...)
	}
	return ents, nil
}

// firstForRoutes keeps each route's first n stop times at each entity, across
// service days.
func firstForRoutes(sts []*model.StopTime, n int) []*model.StopTime {
	// The service days' times overlap, so order by when each stop time happens,
	// as the resolver does.
	slices.SortStableFunc(sts, func(a, b *model.StopTime) int {
		return cmp.Compare(
			a.ServiceDate.Val.Unix()+int64(a.DepartureTime.Int()),
			b.ServiceDate.Val.Unix()+int64(b.DepartureTime.Int()),
		)
	})
	type routeAtEntity struct {
		stop, location, locationGroup string
		route                         int64
	}
	seen := map[routeAtEntity]int{}
	var ret []*model.StopTime
	for _, st := range sts {
		k := routeAtEntity{st.StopID.Val, st.LocationID.Val, st.LocationGroupID.Val, st.RouteID.Val}
		if seen[k] < n {
			seen[k]++
			ret = append(ret, st)
		}
	}
	return ret
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
		"gtfs_trips.trip_id AS gtfs_trip_id",
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

	// Only the filter's stops apply here. Its time window applies to a run's
	// own times, which an instance of a frequency-based trip shifts, so the
	// resolver applies it. Without the stops, a route crossing a small query
	// area returns every stop on the route.
	if where != nil && len(where.StopIds) > 0 {
		q = q.Where(In("sts.stop_id", where.StopIds))
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

// stopDeparturesSelect returns each entity's first limit stop_times on a specific date.
// Filters by the entity IDs based on entityType (stop_id, location_id, or location_group_id).
func stopDeparturesSelect(fvid int, entityIDs []int, entityType stopTimeEntityType, limit *int, where *model.StopTimeFilter) sq.SelectBuilder {
	// Where must already be set for local service date and timezone
	serviceDate := time.Now()
	if where != nil && where.ServiceDate != nil {
		serviceDate = where.ServiceDate.Val
	}

	// Each entity's departures in order, for firstDepartures, or with RouteLimit
	// each route's departures at each entity. A stop time names exactly one
	// stop, location or location group.
	entityRank := "row_number() over (partition by sts.stop_id, sts.location_id, sts.location_group_id order by sts.departure_time_freq, gtfs_trips.id) AS entity_rank"
	if where != nil && where.RouteLimit != nil {
		entityRank = "row_number() over (partition by sts.stop_id, sts.location_id, sts.location_group_id, gtfs_trips.route_id order by sts.departure_time_freq, gtfs_trips.id) AS entity_rank"
	}

	// Build main query with CTEs
	q := sq.StatementBuilder.Select(
		"gtfs_trips.journey_pattern_id",
		"gtfs_trips.journey_pattern_offset",
		"gtfs_trips.id AS trip_id",
		"gtfs_trips.trip_id AS gtfs_trip_id",
		"gtfs_trips.feed_version_id",
		"gtfs_trips.route_id",
		"sts.stop_id",
		"sts.location_id",
		"sts.location_group_id",
		"sts.arrival_time_freq AS arrival_time",
		"sts.departure_time_freq AS departure_time",
		// The instance's start, and its shift from the trip's own stop times,
		// which include the trip's pattern offset where the run's times below
		// count from the pattern's representative.
		"freq.freq_start AS run_start",
		"freq.freq_start - trip_stop_sequence.first_departure_time - gtfs_trips.journey_pattern_offset AS run_offset",
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
		entityRank,
	).
		WithCTE(activeServicesCTE(fvid, serviceDate)).
		From("gtfs_trips").
		Join("active_services gc on gc.id = gtfs_trips.service_id").
		Join("gtfs_trips base_trip ON base_trip.trip_id::text = gtfs_trips.journey_pattern_id AND gtfs_trips.feed_version_id = base_trip.feed_version_id").
		Join("feed_versions on feed_versions.id = gtfs_trips.feed_version_id").
		// The trip's runs. A trip in frequencies.txt runs once per start: every
		// headway_secs from start_time, up to but not including end_time. Any other
		// trip runs once, at its own times, with a null freq_start. A frequency under
		// minExpandedHeadwaySecs isn't expanded, so it adds no runs.
		JoinClause(`join lateral (
			select
				generate_series(start_time, end_time - 1, headway_secs) freq_start
			from gtfs_frequencies
			where gtfs_frequencies.trip_id = gtfs_trips.id and gtfs_frequencies.headway_secs >= ?
			union all
			select null
			where not exists (select 1 from gtfs_frequencies where gtfs_frequencies.trip_id = gtfs_trips.id)
			) freq on true`, minExpandedHeadwaySecs).
		JoinClause(`join lateral (
			select 
				min(sts2.departure_time) first_departure_time,
				min(sts2.stop_sequence) stop_sequence_min, 
				max(sts2.stop_sequence) stop_sequence_max 
			from gtfs_stop_times sts2 
			where sts2.trip_id = base_trip.id and sts2.feed_version_id = base_trip.feed_version_id
			) trip_stop_sequence on true`).
		// A frequency run shifts the pattern's stop times to begin at its start;
		// any other trip shifts them by its offset in the pattern.
		JoinClause(`join lateral (
			select 
				sts.*,
				sts.arrival_time + coalesce(
					freq.freq_start - trip_stop_sequence.first_departure_time,
					gtfs_trips.journey_pattern_offset
				) AS arrival_time_freq,
				sts.departure_time + coalesce(
					freq.freq_start - trip_stop_sequence.first_departure_time,
					gtfs_trips.journey_pattern_offset
				) AS departure_time_freq
			from gtfs_stop_times sts
			where sts.trip_id = base_trip.id and sts.feed_version_id = base_trip.feed_version_id		
			) sts on true`).
		Where(sq.Eq{"sts.feed_version_id": fvid})

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
	// The ranks count each route's departures; the loader still caps each
	// entity's total at limit.
	if where != nil && where.RouteLimit != nil {
		limit = where.RouteLimit
	}
	return firstDepartures(q, limit)
}

// stopDeparturesMaterializedSelect is stopDeparturesSelect planned from a
// materialized CTE of the entities' stop_times. It checks service and route on
// each pattern's base trip, so relies on a pattern's trips sharing a service_id
// and a route_id.
func stopDeparturesMaterializedSelect(fvid int, entityIDs []int, entityType stopTimeEntityType, limit *int, where *model.StopTimeFilter) sq.SelectBuilder {
	// Where must already be set for local service date and timezone
	serviceDate := time.Now()
	if where != nil && where.ServiceDate != nil {
		serviceDate = where.ServiceDate.Val
	}

	// The stop_times at the requested entities, with their base trips.
	//
	// Materialized and filtered only by the entity ids, which are global keys, so every plan
	// starts from them. Postgres treats those ids and feed_version_id as independent filters;
	// given feed_version_id or the service check here, it starts from the whole feed version.
	atEntity := sq.StatementBuilder.
		Select("base_trip.trip_id AS base_trip_gtfs_id", "base_trip.service_id AS base_trip_service_id", "base_trip.route_id AS base_trip_route_id", "sts.*").
		From("gtfs_stop_times sts").
		Join("gtfs_trips base_trip ON base_trip.id = sts.trip_id")
	if len(entityIDs) > 0 {
		switch entityType {
		case stopTimeEntityStop:
			atEntity = atEntity.Where(In("sts.stop_id", entityIDs))
		case stopTimeEntityLocation:
			atEntity = atEntity.Where(In("sts.location_id", entityIDs))
		case stopTimeEntityLocationGroup:
			atEntity = atEntity.Where(In("sts.location_group_id", entityIDs))
		}
	}

	// Each entity's departures in order, for firstDepartures, or with RouteLimit
	// each route's departures at each entity. A stop time names exactly one
	// stop, location or location group.
	entityRank := "row_number() over (partition by sts.stop_id, sts.location_id, sts.location_group_id order by sts_freq.departure_time_freq, gtfs_trips.id) AS entity_rank"
	if where != nil && where.RouteLimit != nil {
		entityRank = "row_number() over (partition by sts.stop_id, sts.location_id, sts.location_group_id, gtfs_trips.route_id order by sts_freq.departure_time_freq, gtfs_trips.id) AS entity_rank"
	}

	// Build main query with CTEs
	q := sq.StatementBuilder.Select(
		"gtfs_trips.journey_pattern_id",
		"gtfs_trips.journey_pattern_offset",
		"gtfs_trips.id AS trip_id",
		"gtfs_trips.trip_id AS gtfs_trip_id",
		"gtfs_trips.feed_version_id",
		"gtfs_trips.route_id",
		"sts.stop_id",
		"sts.location_id",
		"sts.location_group_id",
		"sts_freq.arrival_time_freq AS arrival_time",
		"sts_freq.departure_time_freq AS departure_time",
		// The instance's start, and its shift from the trip's own stop times,
		// which include the trip's pattern offset where the run's times below
		// count from the pattern's representative.
		"freq.freq_start AS run_start",
		"freq.freq_start - trip_stop_sequence.first_departure_time - gtfs_trips.journey_pattern_offset AS run_offset",
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
		entityRank,
	).
		WithCTE(activeServicesCTE(fvid, serviceDate)).
		WithCTE(sq.CTE{Alias: "at_entity", Materialized: true, Expression: atEntity}).
		From("at_entity sts").
		Join("gtfs_trips ON gtfs_trips.feed_version_id = ? AND gtfs_trips.journey_pattern_id = sts.base_trip_gtfs_id::text", fvid).
		// The trip's runs. A trip in frequencies.txt runs once per start: every
		// headway_secs from start_time, up to but not including end_time. Any other
		// trip runs once, at its own times, with a null freq_start. A frequency under
		// minExpandedHeadwaySecs isn't expanded, so it adds no runs.
		JoinClause(`join lateral (
			select
				generate_series(start_time, end_time - 1, headway_secs) freq_start
			from gtfs_frequencies
			where gtfs_frequencies.trip_id = gtfs_trips.id and gtfs_frequencies.headway_secs >= ?
			union all
			select null
			where not exists (select 1 from gtfs_frequencies where gtfs_frequencies.trip_id = gtfs_trips.id)
			) freq on true`, minExpandedHeadwaySecs).
		JoinClause(`join lateral (
			select
				min(sts2.departure_time) first_departure_time,
				min(sts2.stop_sequence) stop_sequence_min,
				max(sts2.stop_sequence) stop_sequence_max
			from gtfs_stop_times sts2
			where sts2.trip_id = sts.trip_id and sts2.feed_version_id = ?
			) trip_stop_sequence on true`, fvid).
		// A frequency run shifts the pattern's stop times to begin at its start;
		// any other trip shifts them by its offset in the pattern.
		JoinClause(`join lateral (
			select
				sts.arrival_time + coalesce(
					freq.freq_start - trip_stop_sequence.first_departure_time,
					gtfs_trips.journey_pattern_offset
				) AS arrival_time_freq,
				sts.departure_time + coalesce(
					freq.freq_start - trip_stop_sequence.first_departure_time,
					gtfs_trips.journey_pattern_offset
				) AS departure_time_freq
			) sts_freq on true`).
		// A pattern's trips share a service_id, so checking the base trip covers them all.
		Where("sts.base_trip_service_id = ANY(ARRAY(SELECT id FROM active_services))")

	if where != nil {
		if where.ExcludeFirst != nil && *where.ExcludeFirst {
			q = q.Where("sts.stop_sequence > trip_stop_sequence.stop_sequence_min")
		}
		if where.ExcludeLast != nil && *where.ExcludeLast {
			q = q.Where("sts.stop_sequence < trip_stop_sequence.stop_sequence_max")
		}
		if len(where.RouteOnestopIds) > 0 {
			// The requested routes' ids in this feed version.
			requestedRoutes := sq.StatementBuilder.
				Select("gtfs_routes.id").
				From("gtfs_routes").
				Where(sq.Eq{"gtfs_routes.feed_version_id": fvid})
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
				q = q.WithCTE(routeLookupCte)
				requestedRoutes = requestedRoutes.
					Join("feed_versions on feed_versions.id = gtfs_routes.feed_version_id").
					Join("route_lookup tlros on tlros.entity_id = gtfs_routes.route_id and tlros.feed_id = feed_versions.feed_id")
			} else {
				requestedRoutes = requestedRoutes.
					Join("feed_version_route_onestop_ids on feed_version_route_onestop_ids.entity_id = gtfs_routes.route_id and feed_version_route_onestop_ids.feed_version_id = gtfs_routes.feed_version_id").
					Where(In("feed_version_route_onestop_ids.onestop_id", where.RouteOnestopIds))
			}
			// Checked on the base trip, like service, so the routes filter the stop times
			// rather than drive the plan. Postgres expects about one trip per route, and a
			// plan driven by the routes rescans the stop times for every trip.
			q = q.
				WithCTE(sq.CTE{Alias: "requested_routes", Materialized: true, Expression: requestedRoutes}).
				Where("sts.base_trip_route_id = ANY(ARRAY(SELECT id FROM requested_routes))")
		}
		// Accept either Start/End or StartTime/EndTime
		if where.Start != nil && where.Start.Valid {
			where.StartTime = ptr(where.Start.Int())
		}
		if where.End != nil && where.End.Valid {
			where.EndTime = ptr(where.End.Int())
		}
		if where.StartTime != nil {
			q = q.Where(sq.GtOrEq{"sts_freq.departure_time_freq": *where.StartTime})
		}
		if where.EndTime != nil {
			q = q.Where(sq.LtOrEq{"sts_freq.departure_time_freq": *where.EndTime})
		}
	}
	// The ranks count each route's departures; the loader still caps each
	// entity's total at limit.
	if where != nil && where.RouteLimit != nil {
		limit = where.RouteLimit
	}
	return firstDepartures(q, limit)
}

// firstDepartures keeps the rows of a departures query whose entity_rank is
// within limit, so frequency runs past the limit never leave the database.
func firstDepartures(q sq.SelectBuilder, limit *int) sq.SelectBuilder {
	return sq.StatementBuilder.
		Select("departures.*").
		FromSelect(q, "departures").
		Where(sq.LtOrEq{"departures.entity_rank": finderCheckLimit(limit)}).
		OrderBy("departures.departure_time", "departures.trip_id")
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
