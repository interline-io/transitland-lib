package rtfinder

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/set"
	"github.com/interline-io/transitland-lib/server/caches/kvcache"
	"github.com/interline-io/transitland-lib/server/caches/tzcache"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/jmoiron/sqlx"
)

// lookupTimeout bounds one read-through lookup query, including the wait for
// a pooled connection, so a hung query cannot hold up every caller of its key.
const lookupTimeout = 5 * time.Second

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
)

type lookupCache struct {
	db                     tldb.Ext
	rtFeedsCache           *kvcache.Cache[int, []string]                  // by feed version id
	stopTzCache            *kvcache.Cache[int, string]                    // timezone names by stop id
	fvTzCache              *kvcache.Cache[int, string]                    // timezone names by feed version id
	fvAgencyCache          *kvcache.Cache[int, []string]                  // GTFS agency_ids by feed version id
	fvRouteTypeCache       *kvcache.Cache[int, []int]                     // route types by feed version id
	fvRouteCache           *kvcache.Cache[feedVersionRouteKey, gtfsRoute] // routes by GTFS route_id within a feed version
	routeCache             *kvcache.Cache[int, gtfsRoute]                 // routes by database id
	fvidFeedCache          *simpleCache[int, string]
	feedOperatorCountCache *simpleCache[string, int]
	agencyRouteIdCache     *simpleCache[int, set.Set[string]]
	gtfsStopIdCache        *simpleCache[int, string]
	gtfsAgencyIdCache      *simpleCache[int, string]
	tzCache                *tzcache.Cache[int]
}

func newLookupCache(db tldb.Ext) *lookupCache {
	f := &lookupCache{
		db:                     db,
		tzCache:                tzcache.NewCache[int](),
		fvidFeedCache:          newSimpleCache[int, string](),
		feedOperatorCountCache: newSimpleCache[string, int](),
		agencyRouteIdCache:     newSimpleCache[int, set.Set[string]](),
		gtfsStopIdCache:        newSimpleCache[int, string](),
		gtfsAgencyIdCache:      newSimpleCache[int, string](),
	}
	f.rtFeedsCache = kvcache.NewRefreshCache(nil, "rtfeeds", f.queryFeedVersionRTFeeds)
	f.rtFeedsCache.RefreshTimeout = lookupTimeout
	f.stopTzCache = kvcache.NewRefreshCache(nil, "stoptz", f.queryStopTimezone)
	f.stopTzCache.RefreshTimeout = lookupTimeout
	f.fvTzCache = kvcache.NewRefreshCache(nil, "fvtz", f.queryFeedVersionTimezone)
	f.fvTzCache.RefreshTimeout = lookupTimeout
	f.fvAgencyCache = kvcache.NewRefreshCache(nil, "fvagencies", f.queryFeedVersionAgencyIDs)
	f.fvAgencyCache.RefreshTimeout = lookupTimeout
	f.fvRouteTypeCache = kvcache.NewRefreshCache(nil, "fvroutetypes", f.queryFeedVersionRouteTypes)
	f.fvRouteTypeCache.RefreshTimeout = lookupTimeout
	// A route that isn't there is remembered as missing, as feed versions don't change.
	f.fvRouteCache = kvcache.NewRefreshCache(nil, "fvroutes", f.queryFeedVersionRoute)
	f.fvRouteCache.RefreshTimeout = lookupTimeout
	f.fvRouteCache.NegativeTTL = f.fvRouteCache.Expires
	f.routeCache = kvcache.NewRefreshCache(nil, "routes", f.queryRoute)
	f.routeCache.RefreshTimeout = lookupTimeout
	f.routeCache.NegativeTTL = f.routeCache.Expires
	return f
}

func (f *lookupCache) GetGtfsStopID(id int) (string, bool) {
	if a, ok := f.gtfsStopIdCache.Get(id); ok {
		return a, ok
	}
	eid := ""
	err := sqlx.Get(f.db, &eid, gtfsStopIdQuery, id)
	f.gtfsStopIdCache.Set(id, eid)
	return eid, err == nil
}

// GetGtfsAgencyID returns the GTFS agency_id of an agency by database id.
func (f *lookupCache) GetGtfsAgencyID(ctx context.Context, id int) (string, bool) {
	if a, ok := f.gtfsAgencyIdCache.Get(id); ok {
		return a, ok
	}
	eid := ""
	if err := sqlx.Get(f.db, &eid, gtfsAgencyIdQuery, id); err != nil {
		log.For(ctx).Error().Err(err).Int("agency_id", id).Msg("rtfinder: agency id lookup failed")
		return "", false
	}
	f.gtfsAgencyIdCache.Set(id, eid)
	return eid, true
}

// GetFeedOperatorCount returns the number of operators a feed is associated
// with. A feed serving more than one operator is a shared feed, whose contents
// cannot be attributed to any single one of them.
func (f *lookupCache) GetFeedOperatorCount(ctx context.Context, onestopId string) (int, bool) {
	if a, ok := f.feedOperatorCountCache.Get(onestopId); ok {
		return a, true
	}
	count := 0
	if err := sqlx.Get(f.db, &count, feedOperatorCountQuery, onestopId); err != nil {
		log.For(ctx).Error().Err(err).Str("feed_onestop_id", onestopId).Msg("rtfinder: feed operator count lookup failed")
		return 0, false
	}
	f.feedOperatorCountCache.Set(onestopId, count)
	return count, true
}

// GetAgencyRouteIDs returns the set of GTFS route_ids operated by an agency.
func (f *lookupCache) GetAgencyRouteIDs(ctx context.Context, id int) set.Set[string] {
	if a, ok := f.agencyRouteIdCache.Get(id); ok {
		return a
	}
	var routeIds []string
	if err := sqlx.Select(f.db, &routeIds, agencyRouteIdsQuery, id); err != nil {
		log.For(ctx).Error().Err(err).Int("agency_id", id).Msg("rtfinder: agency route id lookup failed")
		return nil
	}
	ret := set.New(routeIds...)
	f.agencyRouteIdCache.Set(id, ret)
	return ret
}

// GetFeedVersionAgencyIDs returns the GTFS agency_ids of a feed version's agencies.
func (f *lookupCache) GetFeedVersionAgencyIDs(ctx context.Context, id int) ([]string, bool) {
	return getUnlessGone(ctx, f.fvAgencyCache, id)
}

func (f *lookupCache) queryFeedVersionAgencyIDs(ctx context.Context, id int) ([]string, error) {
	var ret []string
	if err := sqlx.SelectContext(ctx, f.db, &ret, feedVersionAgencyIdsQuery, id); err != nil {
		log.For(ctx).Error().Err(err).Int("feed_version_id", id).Msg("rtfinder: feed version agency lookup failed")
		return nil, err
	}
	return ret, nil
}

// GetFeedVersionRouteTypes returns the distinct route types of a feed version's routes.
func (f *lookupCache) GetFeedVersionRouteTypes(ctx context.Context, id int) ([]int, bool) {
	return getUnlessGone(ctx, f.fvRouteTypeCache, id)
}

func (f *lookupCache) queryFeedVersionRouteTypes(ctx context.Context, id int) ([]int, error) {
	var ret []int
	if err := sqlx.SelectContext(ctx, f.db, &ret, feedVersionRouteTypesQuery, id); err != nil {
		log.For(ctx).Error().Err(err).Int("feed_version_id", id).Msg("rtfinder: feed version route type lookup failed")
		return nil, err
	}
	return ret, nil
}

// gtfsRoute is a route's database id, GTFS route_id and route_type, and its
// agency's GTFS agency_id.
type gtfsRoute struct {
	ID        int    `db:"id"`
	RouteID   string `db:"route_id"`
	RouteType int    `db:"route_type"`
	AgencyID  string `db:"agency_id"`
}

// feedVersionRouteKey names a route by GTFS route_id within a feed version.
// Exported fields, as kvcache encodes keys as JSON.
type feedVersionRouteKey struct {
	FeedVersionID int
	RouteID       string
}

// GetFeedVersionRoute returns a route by its GTFS route_id within a feed version.
func (f *lookupCache) GetFeedVersionRoute(ctx context.Context, fvid int, routeId string) (gtfsRoute, bool) {
	return getUnlessGone(ctx, f.fvRouteCache, feedVersionRouteKey{FeedVersionID: fvid, RouteID: routeId})
}

func (f *lookupCache) queryFeedVersionRoute(ctx context.Context, key feedVersionRouteKey) (gtfsRoute, error) {
	var ret gtfsRoute
	err := sqlx.GetContext(ctx, f.db, &ret, feedVersionRouteQuery, key.FeedVersionID, key.RouteID)
	if errors.Is(err, sql.ErrNoRows) {
		return ret, kvcache.ErrNotFound
	} else if err != nil {
		log.For(ctx).Error().Err(err).Int("feed_version_id", key.FeedVersionID).Str("route_id", key.RouteID).Msg("rtfinder: route lookup failed")
	}
	return ret, err
}

// GetRoute returns a route by database id.
func (f *lookupCache) GetRoute(ctx context.Context, id int) (gtfsRoute, bool) {
	return getUnlessGone(ctx, f.routeCache, id)
}

func (f *lookupCache) queryRoute(ctx context.Context, id int) (gtfsRoute, error) {
	var ret gtfsRoute
	err := sqlx.GetContext(ctx, f.db, &ret, routeQuery, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ret, kvcache.ErrNotFound
	} else if err != nil {
		log.For(ctx).Error().Err(err).Int("route_id", id).Msg("rtfinder: route lookup failed")
	}
	return ret, err
}

// GetTripRouteIDs returns the GTFS route_ids, by trip_id, of the trips a
// realtime message names without a route_id, within a feed version.
func (f *lookupCache) GetTripRouteIDs(ctx context.Context, src *Source, fvid int) map[string]string {
	if len(src.unroutedTripIds) == 0 {
		return nil
	}
	src.tripRoutesOnce.Do(func() {
		src.tripRoutes = kvcache.NewRefreshCache(nil, "triproutes", func(ctx context.Context, fvid int) (map[string]string, error) {
			return f.queryTripRouteIDs(ctx, fvid, src.unroutedTripIds)
		})
		src.tripRoutes.RefreshTimeout = lookupTimeout
	})
	// A failed lookup is not cached: its trips match no route until a later
	// call succeeds.
	ret, _ := getUnlessGone(ctx, src.tripRoutes, fvid)
	return ret
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

func (f *lookupCache) queryTripRouteIDs(ctx context.Context, fvid int, tripIds []string) (map[string]string, error) {
	var rows []struct {
		TripID  string `db:"trip_id"`
		RouteID string `db:"route_id"`
	}
	if err := sqlx.SelectContext(ctx, f.db, &rows, tripRouteIdsQuery, fvid, tripIds); err != nil {
		log.For(ctx).Error().Err(err).Int("feed_version_id", fvid).Int("trip_ids", len(tripIds)).Msg("rtfinder: trip route id lookup failed")
		return nil, err
	}
	ret := make(map[string]string, len(rows))
	for _, row := range rows {
		ret[row.TripID] = row.RouteID
	}
	return ret, nil
}

// GetFeedVersionRTFeeds returns the onestop_ids of the feeds that share an
// operator with a feed version: the topics that may hold its realtime data.
func (f *lookupCache) GetFeedVersionRTFeeds(ctx context.Context, id int) ([]string, bool) {
	return f.rtFeedsCache.Get(ctx, id)
}

func (f *lookupCache) queryFeedVersionRTFeeds(ctx context.Context, id int) ([]string, error) {
	var eid []string
	if err := sqlx.SelectContext(ctx, f.db, &eid, feedVersionRTFeedsQuery, id); err != nil {
		log.For(ctx).Error().Err(err).Int("feed_version_id", id).Msg("rtfinder: rt feeds lookup failed")
		return nil, err
	}
	return eid, nil
}

// StopTimezone looks up the timezone for a stop
func (f *lookupCache) StopTimezone(ctx context.Context, id int, known string) (*time.Location, bool) {
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
	tz, ok := f.stopTzCache.Get(ctx, id)
	if !ok {
		return nil, false
	}
	return f.tzCache.Location(tz)
}

func (f *lookupCache) queryStopTimezone(ctx context.Context, id int) (string, error) {
	tz := ""
	if err := sqlx.GetContext(ctx, f.db, &tz, stopTimezoneQuery, id); err != nil {
		log.For(ctx).Error().Err(err).Int("stop_id", id).Msg("tz: lookup failed")
		return "", err
	}
	log.TraceCheck(func() {
		log.For(ctx).Trace().Int("stop_id", id).Str("tz", tz).Msg("tz: lookup successful")
	})
	return tz, nil
}

// FeedVersionTimezone looks up the timezone for a feed version using the first agency's timezone
func (f *lookupCache) FeedVersionTimezone(ctx context.Context, fvid int) (*time.Location, bool) {
	tz, ok := f.fvTzCache.Get(ctx, fvid)
	if !ok {
		return nil, false
	}
	return f.tzCache.Location(tz)
}

func (f *lookupCache) queryFeedVersionTimezone(ctx context.Context, fvid int) (string, error) {
	tz := ""
	if err := sqlx.GetContext(ctx, f.db, &tz, feedVersionTimezoneQuery, fvid); err != nil {
		log.For(ctx).Error().Err(err).Int("feed_version_id", fvid).Msg("tz: feed version timezone lookup failed")
		return "", err
	}
	return tz, nil
}

// Lookup time.Location by name
func (f *lookupCache) Location(tz string) (*time.Location, bool) {
	return f.tzCache.Location(tz)
}

type simpleCache[K comparable, V any] struct {
	lock   sync.Mutex
	values map[K]V
}

func (c *simpleCache[K, V]) Get(key K) (V, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	a, ok := c.values[key]
	return a, ok
}

func (c *simpleCache[K, V]) Set(key K, value V) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.values == nil {
		c.values = map[K]V{}
	}
	c.values[key] = value
}

func newSimpleCache[K comparable, V any]() *simpleCache[K, V] {
	return &simpleCache[K, V]{
		values: map[K]V{},
	}
}
