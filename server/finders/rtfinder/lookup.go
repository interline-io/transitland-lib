package rtfinder

import (
	"context"
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

type lookupCache struct {
	db                     tldb.Ext
	rtFeedsCache           *kvcache.Cache[int, []string] // by feed version id
	stopTzCache            *kvcache.Cache[int, string]   // timezone names by stop id
	fvTzCache              *kvcache.Cache[int, string]   // timezone names by feed version id
	fvidFeedCache          *simpleCache[int, string]
	fvidAgencyCountCache   *simpleCache[int, int]
	feedOperatorCountCache *simpleCache[string, int]
	agencyRouteIdCache     *simpleCache[int, set.Set[string]]
	gtfsStopIdCache        *simpleCache[int, string]
	gtfsAgencyIdCache      *simpleCache[int, string]
	routeIdCache           *simpleCache[skey, int]
	tzCache                *tzcache.Cache[int]
}

func newLookupCache(db tldb.Ext) *lookupCache {
	f := &lookupCache{
		db:                     db,
		tzCache:                tzcache.NewCache[int](),
		fvidFeedCache:          newSimpleCache[int, string](),
		fvidAgencyCountCache:   newSimpleCache[int, int](),
		feedOperatorCountCache: newSimpleCache[string, int](),
		agencyRouteIdCache:     newSimpleCache[int, set.Set[string]](),
		gtfsStopIdCache:        newSimpleCache[int, string](),
		gtfsAgencyIdCache:      newSimpleCache[int, string](),
		routeIdCache:           newSimpleCache[skey, int](),
	}
	f.rtFeedsCache = kvcache.NewRefreshCache(nil, "rtfeeds", f.queryFeedVersionRTFeeds)
	f.rtFeedsCache.RefreshTimeout = lookupTimeout
	f.stopTzCache = kvcache.NewRefreshCache(nil, "stoptz", f.queryStopTimezone)
	f.stopTzCache.RefreshTimeout = lookupTimeout
	f.fvTzCache = kvcache.NewRefreshCache(nil, "fvtz", f.queryFeedVersionTimezone)
	f.fvTzCache.RefreshTimeout = lookupTimeout
	return f
}

func (f *lookupCache) GetRouteID(fvid int, tid string) (int, bool) {
	sk := skey{fvid, tid}
	if a, ok := f.routeIdCache.Get(sk); ok {
		return a, ok
	}
	eid := 0
	err := sqlx.Get(f.db, &eid, "select id from gtfs_routes where feed_version_id = $1 and route_id = $2", fvid, tid)
	f.routeIdCache.Set(sk, eid)
	return eid, err == nil
}

func (f *lookupCache) GetGtfsStopID(id int) (string, bool) {
	if a, ok := f.gtfsStopIdCache.Get(id); ok {
		return a, ok
	}
	q := `select stop_id from gtfs_stops where id = $1 limit 1`
	eid := ""
	err := sqlx.Get(f.db, &eid, q, id)
	f.gtfsStopIdCache.Set(id, eid)
	return eid, err == nil
}

// GetGtfsAgencyID returns the GTFS agency_id of an agency by database id.
func (f *lookupCache) GetGtfsAgencyID(id int) (string, bool) {
	if a, ok := f.gtfsAgencyIdCache.Get(id); ok {
		return a, ok
	}
	q := `select agency_id from gtfs_agencies where id = $1 limit 1`
	eid := ""
	if err := sqlx.Get(f.db, &eid, q, id); err != nil {
		return "", false
	}
	f.gtfsAgencyIdCache.Set(id, eid)
	return eid, true
}

// GetFeedVersionAgencyCount returns the number of agencies in a feed version.
func (f *lookupCache) GetFeedVersionAgencyCount(ctx context.Context, id int) (int, bool) {
	if a, ok := f.fvidAgencyCountCache.Get(id); ok {
		return a, true
	}
	count := 0
	q := `select count(*) from gtfs_agencies where feed_version_id = $1`
	if err := sqlx.Get(f.db, &count, q, id); err != nil {
		log.For(ctx).Error().Err(err).Int("feed_version_id", id).Msg("rtfinder: agency count lookup failed")
		return 0, false
	}
	f.fvidAgencyCountCache.Set(id, count)
	return count, true
}

// GetFeedOperatorCount returns the number of operators a feed is associated
// with. A feed serving more than one operator is a shared feed, whose contents
// cannot be attributed to any single one of them.
func (f *lookupCache) GetFeedOperatorCount(ctx context.Context, onestopId string) (int, bool) {
	if a, ok := f.feedOperatorCountCache.Get(onestopId); ok {
		return a, true
	}
	count := 0
	q := `
	select count(distinct coif.resolved_onestop_id)
	from current_feeds cf
	join current_operators_in_feed coif on coif.feed_id = cf.id
	where cf.onestop_id = $1`
	if err := sqlx.Get(f.db, &count, q, onestopId); err != nil {
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
	q := `select route_id from gtfs_routes where agency_id = $1`
	if err := sqlx.Select(f.db, &routeIds, q, id); err != nil {
		log.For(ctx).Error().Err(err).Int("agency_id", id).Msg("rtfinder: agency route id lookup failed")
		return nil
	}
	ret := set.New(routeIds...)
	f.agencyRouteIdCache.Set(id, ret)
	return ret
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
	ret, _ := src.tripRoutes.Get(ctx, fvid)
	return ret
}

func (f *lookupCache) queryTripRouteIDs(ctx context.Context, fvid int, tripIds []string) (map[string]string, error) {
	q := `
	select gtfs_trips.trip_id, gtfs_routes.route_id
	from gtfs_trips
	join gtfs_routes on gtfs_routes.id = gtfs_trips.route_id
	where gtfs_trips.feed_version_id = $1 and gtfs_trips.trip_id = any($2)`
	var rows []struct {
		TripID  string `db:"trip_id"`
		RouteID string `db:"route_id"`
	}
	if err := sqlx.SelectContext(ctx, f.db, &rows, q, fvid, tripIds); err != nil {
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
	q := `
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
	var eid []string
	if err := sqlx.SelectContext(ctx, f.db, &eid, q, id); err != nil {
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
	q := `
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
	tz := ""
	if err := sqlx.GetContext(ctx, f.db, &tz, q, id); err != nil {
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
	q := `SELECT agency_timezone FROM gtfs_agencies WHERE feed_version_id = $1 LIMIT 1`
	tz := ""
	if err := sqlx.GetContext(ctx, f.db, &tz, q, fvid); err != nil {
		log.For(ctx).Error().Err(err).Int("feed_version_id", fvid).Msg("tz: feed version timezone lookup failed")
		return "", err
	}
	return tz, nil
}

// Lookup time.Location by name
func (f *lookupCache) Location(tz string) (*time.Location, bool) {
	return f.tzCache.Location(tz)
}

/////

type skey struct {
	fvid int
	eid  string
}

///

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
