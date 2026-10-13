package rtfinder

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/caches/kvcache"
	"github.com/interline-io/transitland-lib/server/caches/tzcache"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/jmoiron/sqlx"
)

// Lookup answers realtime matching's questions about static GTFS data. Each
// method returns false when it can't answer.
type Lookup interface {
	// GetFeedVersionRTFeeds returns the onestop_ids of the feeds that may hold
	// a feed version's realtime data.
	GetFeedVersionRTFeeds(ctx context.Context, fvid int) ([]string, bool)
	// FeedVersionTimezone returns a feed version's timezone.
	FeedVersionTimezone(ctx context.Context, fvid int) (*time.Location, bool)
	// GetTripSpan returns when a trip runs on its service day: zero for a trip
	// that isn't found, or has no timed stop times.
	GetTripSpan(ctx context.Context, fvid int, tripId int) (TripSpan, bool)
	// GetGtfsStopID returns a stop's GTFS stop_id.
	GetGtfsStopID(ctx context.Context, id int) (string, bool)
	// StopTimezone returns a stop's timezone, or known when it is given.
	StopTimezone(ctx context.Context, id int, known string) (*time.Location, bool)
	// GetGtfsAgencyID returns an agency's GTFS agency_id.
	GetGtfsAgencyID(ctx context.Context, id int) (string, bool)
	// GetRoute returns a route by id.
	GetRoute(ctx context.Context, id int) (GtfsRoute, bool)
	// GetFeedVersionRoute returns a route by its GTFS route_id within a feed version.
	GetFeedVersionRoute(ctx context.Context, fvid int, routeId string) (GtfsRoute, bool)
	// GetFeedVersionAgencyIDs returns the GTFS agency_ids of a feed version's agencies.
	GetFeedVersionAgencyIDs(ctx context.Context, fvid int) ([]string, bool)
	// GetFeedVersionRouteTypes returns the distinct route types of a feed version's routes.
	GetFeedVersionRouteTypes(ctx context.Context, fvid int) ([]int, bool)
	// GetTripRouteIDs returns the GTFS route_ids of trips, by trip_id, within a
	// feed version.
	GetTripRouteIDs(ctx context.Context, fvid int, tripIds []string) (map[string]string, bool)
	// GetAgencyRouteIDs returns the set of GTFS route_ids of an agency's routes.
	GetAgencyRouteIDs(ctx context.Context, id int) (map[string]bool, bool)
	// GetFeedOperatorCount returns the number of operators associated with a feed.
	GetFeedOperatorCount(ctx context.Context, onestopId string) (int, bool)
}

// GtfsRoute is a route's id, GTFS route_id and route_type, and its agency's
// GTFS agency_id.
type GtfsRoute struct {
	ID        int    `db:"id"`
	RouteID   string `db:"route_id"`
	RouteType int    `db:"route_type"`
	AgencyID  string `db:"agency_id"`
}

// TripSpan is when a trip runs on its service day, in seconds into the day.
type TripSpan struct {
	FirstDeparture int `db:"first_departure"`
	LastArrival    int `db:"last_arrival"`
}

// lookupTimeout bounds one read-through lookup query, including the wait for
// a pooled connection, so a hung query cannot hold up every caller of its key.
const lookupTimeout = 5 * time.Second

// errLookupFailed fails a cache load whose Lookup couldn't answer, so the
// failure isn't cached.
var errLookupFailed = errors.New("rtfinder: lookup failed")

// The lookup queries, together so the package's SQL reads in one place.
const (
	gtfsStopIdQuery = `select stop_id from gtfs_stops where id = $1 limit 1`

	gtfsAgencyIdQuery = `select agency_id from gtfs_agencies where id = $1 limit 1`

	feedOperatorCountQuery = `
	select count(distinct coif.resolved_onestop_id)
	from current_feeds cf
	join current_operators_in_feed coif on coif.feed_id = cf.id
	where cf.onestop_id = $1`

	agencyRouteIdsQuery = `select route_id from gtfs_routes where agency_id = $1`

	feedVersionAgencyIdsQuery = `select agency_id from gtfs_agencies where feed_version_id = $1`

	feedVersionRouteTypesQuery = `select distinct route_type from gtfs_routes where feed_version_id = $1`

	feedVersionRouteQuery = `
	select gtfs_routes.id, gtfs_routes.route_id, gtfs_routes.route_type, gtfs_agencies.agency_id
	from gtfs_routes
	join gtfs_agencies on gtfs_agencies.id = gtfs_routes.agency_id
	where gtfs_routes.feed_version_id = $1 and gtfs_routes.route_id = $2`

	routeQuery = `
	select gtfs_routes.id, gtfs_routes.route_id, gtfs_routes.route_type, gtfs_agencies.agency_id
	from gtfs_routes
	join gtfs_agencies on gtfs_agencies.id = gtfs_routes.agency_id
	where gtfs_routes.id = $1`

	tripRouteIdsQuery = `
	select gtfs_trips.trip_id, gtfs_routes.route_id
	from gtfs_trips
	join gtfs_routes on gtfs_routes.id = gtfs_trips.route_id
	where gtfs_trips.feed_version_id = $1 and gtfs_trips.trip_id = any($2)`

	feedVersionRTFeedsQuery = `
	select 
		distinct on(cf.onestop_id)
		cf.onestop_id 
	from feed_versions fv 
	join current_operators_in_feed coif on coif.feed_id = fv.feed_id 
	join current_operators_in_feed coif2 on coif2.resolved_onestop_id = coif.resolved_onestop_id 
	join current_feeds cf on coif2.feed_id = cf.id
	where fv.id = $1 
	order by cf.onestop_id
	`

	stopTimezoneQuery = `
		select COALESCE(nullif(s.stop_timezone, ''), nullif(p.stop_timezone, ''), a.agency_timezone)
		from gtfs_stops s
		left join gtfs_stops p on p.id = s.parent_station
		left join lateral (
			select gtfs_agencies.agency_timezone
			from gtfs_agencies
			where gtfs_agencies.feed_version_id = s.feed_version_id
			limit 1
		) a on true
		where s.id = $1
		limit 1`

	feedVersionTimezoneQuery = `SELECT agency_timezone FROM gtfs_agencies WHERE feed_version_id = $1 LIMIT 1`

	// When a trip runs on its service day: from its first departure to its last
	// arrival, or for a frequency-based trip, from its first start to its last
	// start plus one run's length. A frequency starts a run every headway_secs
	// from start_time, up to but not including end_time. A trip's stop times are
	// stored once per journey pattern, on the pattern's first trip, and shifted
	// by each other trip's offset. Zero for a trip with no timed stop times, or
	// none at all.
	tripSpanQuery = `
	select
		coalesce(fr.first_start, st.first_departure, 0) as first_departure,
		coalesce(fr.last_start + st.last_arrival - st.first_departure, st.last_arrival, 0) as last_arrival
	from (
		select
			min(sts.departure_time + t.journey_pattern_offset) as first_departure,
			max(sts.arrival_time + t.journey_pattern_offset) as last_arrival
		from gtfs_trips t
		join gtfs_trips t2 on t2.trip_id::text = t.journey_pattern_id and t2.feed_version_id = t.feed_version_id
		join gtfs_stop_times sts on sts.trip_id = t2.id and sts.feed_version_id = $2
		where t.id = $1 and t.feed_version_id = $2
	) st, (
		select
			min(start_time) as first_start,
			max(start_time + (end_time - 1 - start_time) / nullif(headway_secs, 0) * headway_secs) as last_start
		from gtfs_frequencies
		where trip_id = $1 and end_time > start_time
	) fr`
)

// PostgresLookup answers a Finder's lookups from the Transitland database.
type PostgresLookup struct {
	db                     tldb.Ext
	rtFeedsCache           *kvcache.Cache[int, []string]                  // by feed version id
	stopTzCache            *kvcache.Cache[int, string]                    // timezone names by stop id
	fvTzCache              *kvcache.Cache[int, string]                    // timezone names by feed version id
	fvAgencyCache          *kvcache.Cache[int, []string]                  // GTFS agency_ids by feed version id
	fvRouteTypeCache       *kvcache.Cache[int, []int]                     // route types by feed version id
	fvRouteCache           *kvcache.Cache[feedVersionRouteKey, GtfsRoute] // routes by GTFS route_id within a feed version
	routeCache             *kvcache.Cache[int, GtfsRoute]                 // routes by database id
	tripSpanCache          *kvcache.Cache[model.FVPair, TripSpan]         // when trips run, by database id
	gtfsStopIdCache        *kvcache.Cache[int, string]                    // GTFS stop_ids by stop id
	gtfsAgencyIdCache      *kvcache.Cache[int, string]                    // GTFS agency_ids by agency id
	feedOperatorCountCache *kvcache.Cache[string, int]                    // operator counts by feed onestop_id
	agencyRouteIdCache     *kvcache.Cache[int, map[string]bool]           // sets of GTFS route_ids by agency id
	tzCache                *tzcache.Cache[int]
}

var _ Lookup = (*PostgresLookup)(nil)

// NewPostgresLookup returns a Lookup over the Transitland database.
func NewPostgresLookup(db tldb.Ext) *PostgresLookup {
	f := &PostgresLookup{
		db:      db,
		tzCache: tzcache.NewCache[int](),
	}
	f.rtFeedsCache = newRefreshCache("rtfeeds", f.queryFeedVersionRTFeeds)
	f.stopTzCache = newRefreshCache("stoptz", f.queryStopTimezone)
	f.fvTzCache = newRefreshCache("fvtz", f.queryFeedVersionTimezone)
	f.fvAgencyCache = newRefreshCache("fvagencies", f.queryFeedVersionAgencyIDs)
	f.fvRouteTypeCache = newRefreshCache("fvroutetypes", f.queryFeedVersionRouteTypes)
	f.fvRouteCache = newRefreshCache("fvroutes", f.queryFeedVersionRoute)
	f.routeCache = newRefreshCache("routes", f.queryRoute)
	f.tripSpanCache = newRefreshCache("tripspans", f.queryTripSpan)
	f.gtfsStopIdCache = newRefreshCache("gtfsstopids", f.queryGtfsStopID)
	f.gtfsAgencyIdCache = newRefreshCache("gtfsagencyids", f.queryGtfsAgencyID)
	f.feedOperatorCountCache = newRefreshCache("feedoperatorcounts", f.queryFeedOperatorCount)
	f.agencyRouteIdCache = newRefreshCache("agencyrouteids", f.queryAgencyRouteIDs)
	return f
}

// newRefreshCache returns a local cache that loads each key with fn, bounded
// by lookupTimeout. A failed load isn't cached. A key that fn reports as
// kvcache.ErrNotFound is remembered as missing, as feed versions don't change.
func newRefreshCache[K comparable, V any](topic string, fn func(context.Context, K) (V, error)) *kvcache.Cache[K, V] {
	c := kvcache.NewRefreshCache(nil, topic, fn)
	c.RefreshTimeout = lookupTimeout
	c.NegativeTTL = c.Expires
	return c
}

// GetGtfsStopID returns the GTFS stop_id of a stop by database id.
func (f *PostgresLookup) GetGtfsStopID(ctx context.Context, id int) (string, bool) {
	return getUnlessGone(ctx, f.gtfsStopIdCache, id)
}

func (f *PostgresLookup) queryGtfsStopID(ctx context.Context, id int) (string, error) {
	eid := ""
	err := sqlx.GetContext(ctx, f.db, &eid, gtfsStopIdQuery, id)
	if errors.Is(err, sql.ErrNoRows) {
		return eid, kvcache.ErrNotFound
	} else if err != nil {
		log.For(ctx).Error().Err(err).Int("stop_id", id).Msg("rtfinder: stop id lookup failed")
	}
	return eid, err
}

// GetGtfsAgencyID returns the GTFS agency_id of an agency by database id.
func (f *PostgresLookup) GetGtfsAgencyID(ctx context.Context, id int) (string, bool) {
	return getUnlessGone(ctx, f.gtfsAgencyIdCache, id)
}

func (f *PostgresLookup) queryGtfsAgencyID(ctx context.Context, id int) (string, error) {
	eid := ""
	err := sqlx.GetContext(ctx, f.db, &eid, gtfsAgencyIdQuery, id)
	if errors.Is(err, sql.ErrNoRows) {
		return eid, kvcache.ErrNotFound
	} else if err != nil {
		log.For(ctx).Error().Err(err).Int("agency_id", id).Msg("rtfinder: agency id lookup failed")
	}
	return eid, err
}

// GetFeedOperatorCount returns the number of operators a feed is associated
// with. A feed serving more than one operator is a shared feed, whose contents
// cannot be attributed to any single one of them.
func (f *PostgresLookup) GetFeedOperatorCount(ctx context.Context, onestopId string) (int, bool) {
	return getUnlessGone(ctx, f.feedOperatorCountCache, onestopId)
}

func (f *PostgresLookup) queryFeedOperatorCount(ctx context.Context, onestopId string) (int, error) {
	count := 0
	if err := sqlx.GetContext(ctx, f.db, &count, feedOperatorCountQuery, onestopId); err != nil {
		log.For(ctx).Error().Err(err).Str("feed_onestop_id", onestopId).Msg("rtfinder: feed operator count lookup failed")
		return 0, err
	}
	return count, nil
}

// GetAgencyRouteIDs returns the set of GTFS route_ids operated by an agency.
func (f *PostgresLookup) GetAgencyRouteIDs(ctx context.Context, id int) (map[string]bool, bool) {
	return getUnlessGone(ctx, f.agencyRouteIdCache, id)
}

func (f *PostgresLookup) queryAgencyRouteIDs(ctx context.Context, id int) (map[string]bool, error) {
	var routeIds []string
	if err := sqlx.SelectContext(ctx, f.db, &routeIds, agencyRouteIdsQuery, id); err != nil {
		log.For(ctx).Error().Err(err).Int("agency_id", id).Msg("rtfinder: agency route id lookup failed")
		return nil, err
	}
	ret := make(map[string]bool, len(routeIds))
	for _, rid := range routeIds {
		ret[rid] = true
	}
	return ret, nil
}

// GetFeedVersionAgencyIDs returns the GTFS agency_ids of a feed version's agencies.
func (f *PostgresLookup) GetFeedVersionAgencyIDs(ctx context.Context, id int) ([]string, bool) {
	return getUnlessGone(ctx, f.fvAgencyCache, id)
}

func (f *PostgresLookup) queryFeedVersionAgencyIDs(ctx context.Context, id int) ([]string, error) {
	var ret []string
	if err := sqlx.SelectContext(ctx, f.db, &ret, feedVersionAgencyIdsQuery, id); err != nil {
		log.For(ctx).Error().Err(err).Int("feed_version_id", id).Msg("rtfinder: feed version agency lookup failed")
		return nil, err
	}
	return ret, nil
}

// GetFeedVersionRouteTypes returns the distinct route types of a feed version's routes.
func (f *PostgresLookup) GetFeedVersionRouteTypes(ctx context.Context, id int) ([]int, bool) {
	return getUnlessGone(ctx, f.fvRouteTypeCache, id)
}

func (f *PostgresLookup) queryFeedVersionRouteTypes(ctx context.Context, id int) ([]int, error) {
	var ret []int
	if err := sqlx.SelectContext(ctx, f.db, &ret, feedVersionRouteTypesQuery, id); err != nil {
		log.For(ctx).Error().Err(err).Int("feed_version_id", id).Msg("rtfinder: feed version route type lookup failed")
		return nil, err
	}
	return ret, nil
}

// feedVersionRouteKey names a route by GTFS route_id within a feed version.
// Exported fields, as kvcache encodes keys as JSON.
type feedVersionRouteKey struct {
	FeedVersionID int
	RouteID       string
}

// GetFeedVersionRoute returns a route by its GTFS route_id within a feed version.
func (f *PostgresLookup) GetFeedVersionRoute(ctx context.Context, fvid int, routeId string) (GtfsRoute, bool) {
	return getUnlessGone(ctx, f.fvRouteCache, feedVersionRouteKey{FeedVersionID: fvid, RouteID: routeId})
}

func (f *PostgresLookup) queryFeedVersionRoute(ctx context.Context, key feedVersionRouteKey) (GtfsRoute, error) {
	var ret GtfsRoute
	err := sqlx.GetContext(ctx, f.db, &ret, feedVersionRouteQuery, key.FeedVersionID, key.RouteID)
	if errors.Is(err, sql.ErrNoRows) {
		return ret, kvcache.ErrNotFound
	} else if err != nil {
		log.For(ctx).Error().Err(err).Int("feed_version_id", key.FeedVersionID).Str("route_id", key.RouteID).Msg("rtfinder: route lookup failed")
	}
	return ret, err
}

// GetTripSpan returns when a trip runs on its service day: zero for a trip that
// isn't found, or has no timed stop times.
func (f *PostgresLookup) GetTripSpan(ctx context.Context, fvid int, tripId int) (TripSpan, bool) {
	return getUnlessGone(ctx, f.tripSpanCache, model.FVPair{FeedVersionID: fvid, EntityID: tripId})
}

func (f *PostgresLookup) queryTripSpan(ctx context.Context, key model.FVPair) (TripSpan, error) {
	var ret TripSpan
	err := sqlx.GetContext(ctx, f.db, &ret, tripSpanQuery, key.EntityID, key.FeedVersionID)
	if err != nil {
		log.For(ctx).Error().Err(err).Int("feed_version_id", key.FeedVersionID).Int("trip_id", key.EntityID).Msg("rtfinder: trip span lookup failed")
	}
	return ret, err
}

// GetRoute returns a route by database id.
func (f *PostgresLookup) GetRoute(ctx context.Context, id int) (GtfsRoute, bool) {
	return getUnlessGone(ctx, f.routeCache, id)
}

func (f *PostgresLookup) queryRoute(ctx context.Context, id int) (GtfsRoute, error) {
	var ret GtfsRoute
	err := sqlx.GetContext(ctx, f.db, &ret, routeQuery, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ret, kvcache.ErrNotFound
	} else if err != nil {
		log.For(ctx).Error().Err(err).Int("route_id", id).Msg("rtfinder: route lookup failed")
	}
	return ret, err
}

// getUnlessGone returns a cached value, loading it on a miss unless the caller
// has gone away. As with GetSource, a held value is served whatever state the
// caller is in.
func getUnlessGone[K comparable, V any](ctx context.Context, c *kvcache.Cache[K, V], key K) (V, bool) {
	if ret, ok := c.Peek(key); ok {
		return ret, true
	}
	if ctx.Err() != nil {
		var zero V
		return zero, false
	}
	return c.Get(ctx, key)
}

// GetTripRouteIDs returns the GTFS route_ids of trips, by trip_id, within a
// feed version.
func (f *PostgresLookup) GetTripRouteIDs(ctx context.Context, fvid int, tripIds []string) (map[string]string, bool) {
	// Not cached: each realtime message names its own trip_ids.
	var rows []struct {
		TripID  string `db:"trip_id"`
		RouteID string `db:"route_id"`
	}
	if err := sqlx.SelectContext(ctx, f.db, &rows, tripRouteIdsQuery, fvid, tripIds); err != nil {
		log.For(ctx).Error().Err(err).Int("feed_version_id", fvid).Int("trip_ids", len(tripIds)).Msg("rtfinder: trip route id lookup failed")
		return nil, false
	}
	ret := make(map[string]string, len(rows))
	for _, row := range rows {
		ret[row.TripID] = row.RouteID
	}
	return ret, true
}

// GetFeedVersionRTFeeds returns the onestop_ids of the feeds that share an
// operator with a feed version: the topics that may hold its realtime data.
func (f *PostgresLookup) GetFeedVersionRTFeeds(ctx context.Context, id int) ([]string, bool) {
	return getUnlessGone(ctx, f.rtFeedsCache, id)
}

func (f *PostgresLookup) queryFeedVersionRTFeeds(ctx context.Context, id int) ([]string, error) {
	var eid []string
	if err := sqlx.SelectContext(ctx, f.db, &eid, feedVersionRTFeedsQuery, id); err != nil {
		log.For(ctx).Error().Err(err).Int("feed_version_id", id).Msg("rtfinder: rt feeds lookup failed")
		return nil, err
	}
	return eid, nil
}

// StopTimezone looks up the timezone for a stop
func (f *PostgresLookup) StopTimezone(ctx context.Context, id int, known string) (*time.Location, bool) {
	// If a timezone is provided, save it and return immediately
	if known != "" {
		log.TraceCheck(func() {
			log.For(ctx).Trace().Int("stop_id", id).Str("known", known).Msg("tz: using known timezone")
		})
		f.stopTzCache.Set(ctx, id, known)
		return f.tzCache.Location(known)
	}
	if id == 0 {
		log.TraceCheck(func() {
			log.For(ctx).Trace().Int("stop_id", id).Msg("tz: lookup failed, cant find timezone for stops with id=0 unless speciifed explicitly")
		})
		return nil, false
	}
	tz, ok := getUnlessGone(ctx, f.stopTzCache, id)
	if !ok {
		return nil, false
	}
	return f.tzCache.Location(tz)
}

func (f *PostgresLookup) queryStopTimezone(ctx context.Context, id int) (string, error) {
	tz := ""
	err := sqlx.GetContext(ctx, f.db, &tz, stopTimezoneQuery, id)
	if errors.Is(err, sql.ErrNoRows) {
		return tz, kvcache.ErrNotFound
	} else if err != nil {
		log.For(ctx).Error().Err(err).Int("stop_id", id).Msg("tz: lookup failed")
		return tz, err
	}
	log.TraceCheck(func() {
		log.For(ctx).Trace().Int("stop_id", id).Str("tz", tz).Msg("tz: lookup successful")
	})
	return tz, nil
}

// FeedVersionTimezone looks up the timezone for a feed version using the first agency's timezone
func (f *PostgresLookup) FeedVersionTimezone(ctx context.Context, fvid int) (*time.Location, bool) {
	tz, ok := getUnlessGone(ctx, f.fvTzCache, fvid)
	if !ok {
		return nil, false
	}
	return f.tzCache.Location(tz)
}

func (f *PostgresLookup) queryFeedVersionTimezone(ctx context.Context, fvid int) (string, error) {
	tz := ""
	err := sqlx.GetContext(ctx, f.db, &tz, feedVersionTimezoneQuery, fvid)
	if errors.Is(err, sql.ErrNoRows) {
		return tz, kvcache.ErrNotFound
	} else if err != nil {
		log.For(ctx).Error().Err(err).Int("feed_version_id", fvid).Msg("tz: feed version timezone lookup failed")
	}
	return tz, err
}
