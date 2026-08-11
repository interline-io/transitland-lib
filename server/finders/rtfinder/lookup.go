package rtfinder

import (
	"context"
	"sync"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/set"
	"github.com/interline-io/transitland-lib/server/caches/tzcache"
	"github.com/jmoiron/sqlx"
)

type lookupCache struct {
	db                     sqlx.Ext
	fvidSourceCache        *simpleCache[int, []string]
	fvidFeedCache          *simpleCache[int, string]
	fvidAgencyCountCache   *simpleCache[int, int]
	feedOperatorCountCache *simpleCache[string, int]
	agencyRouteIdCache     *simpleCache[int, set.Set[string]]
	routeTripIdCache       *simpleCache[int, set.Set[string]]
	gtfsTripIdCache        *simpleCache[int, string]
	gtfsStopIdCache        *simpleCache[int, string]
	routeIdCache           *simpleCache[skey, int]
	tzCache                *tzcache.Cache[int]
	rtLookupLock           sync.Mutex
}

func newLookupCache(db sqlx.Ext) *lookupCache {
	return &lookupCache{
		db:                     db,
		tzCache:                tzcache.NewCache[int](),
		fvidSourceCache:        newSimpleCache[int, []string](),
		fvidFeedCache:          newSimpleCache[int, string](),
		fvidAgencyCountCache:   newSimpleCache[int, int](),
		feedOperatorCountCache: newSimpleCache[string, int](),
		agencyRouteIdCache:     newSimpleCache[int, set.Set[string]](),
		routeTripIdCache:       newSimpleCache[int, set.Set[string]](),
		gtfsTripIdCache:        newSimpleCache[int, string](),
		gtfsStopIdCache:        newSimpleCache[int, string](),
		routeIdCache:           newSimpleCache[skey, int](),
	}
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

func (f *lookupCache) GetGtfsTripID(id int) (string, bool) {
	if a, ok := f.gtfsTripIdCache.Get(id); ok {
		return a, ok
	}
	q := `select trip_id from gtfs_trips where id = $1 limit 1`
	eid := ""
	err := sqlx.Get(f.db, &eid, q, id)
	f.gtfsTripIdCache.Set(id, eid)
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

// GetRouteTripIDs returns the set of GTFS trip_ids belonging to a route.
func (f *lookupCache) GetRouteTripIDs(ctx context.Context, id int) set.Set[string] {
	if a, ok := f.routeTripIdCache.Get(id); ok {
		return a
	}
	var tripIds []string
	q := `select trip_id from gtfs_trips where route_id = $1`
	if err := sqlx.Select(f.db, &tripIds, q, id); err != nil {
		log.For(ctx).Error().Err(err).Int("route_id", id).Msg("rtfinder: route trip id lookup failed")
		return nil
	}
	ret := set.New(tripIds...)
	f.routeTripIdCache.Set(id, ret)
	return ret
}

func (f *lookupCache) GetFeedVersionRTFeeds(id int) ([]string, bool) {
	f.rtLookupLock.Lock()
	defer f.rtLookupLock.Unlock()
	if a, ok := f.fvidSourceCache.Get(id); ok {
		return a, ok
	}
	// Only feeds that actually publish a realtime URL. The operator join reaches
	// every feed the operator has, static ones included, and a topic is keyed on
	// whichever feed owns the URL that was fetched — so without this every
	// lookup also probes feeds that can never hold RT data. Filtering on the URL
	// rather than on spec is deliberate: a gtfs feed can carry realtime URLs of
	// its own. Empty strings count as absent; feeds publish those.
	q := `
	select 
		distinct on(cf.onestop_id)
		cf.onestop_id 
	from feed_versions fv 
	join current_operators_in_feed coif on coif.feed_id = fv.feed_id 
	join current_operators_in_feed coif2 on coif2.resolved_onestop_id = coif.resolved_onestop_id 
	join current_feeds cf on coif2.feed_id = cf.id
	where fv.id = $1 
	and (
		coalesce(cf.urls ->> 'realtime_alerts', '') <> ''
		or coalesce(cf.urls ->> 'realtime_trip_updates', '') <> ''
		or coalesce(cf.urls ->> 'realtime_vehicle_positions', '') <> ''
	)
	order by cf.onestop_id
	`
	var eid []string
	err := sqlx.Select(
		f.db,
		&eid,
		q,
		id,
	)
	f.fvidSourceCache.Set(id, eid) // set before return
	if err != nil {
		return nil, false
	}
	return eid, true
}

// StopTimezone looks up the timezone for a stop
func (f *lookupCache) StopTimezone(ctx context.Context, id int, known string) (*time.Location, bool) {
	// Need to lock while looking up or setting.
	f.rtLookupLock.Lock()
	defer f.rtLookupLock.Unlock()

	// If a timezone is provided, save it and return immediately
	if known != "" {
		log.TraceCheck(func() {
			log.For(ctx).Trace().Int("stop_id", id).Str("known", known).Msg("tz: using known timezone")
		})
		return f.tzCache.Add(id, known)
	}

	// Check the cache
	if loc, ok := f.tzCache.Get(id); ok {
		log.TraceCheck(func() {
			log.For(ctx).Trace().Int("stop_id", id).Str("known", known).Str("loc", loc.String()).Msg("tz: using cached timezone")
		})
		return loc, ok
	} else {
		log.TraceCheck(func() {
			log.For(ctx).Trace().Int("stop_id", id).Str("known", known).Str("loc", loc.String()).Msg("tz: timezone not in cache")
		})
	}
	if id == 0 {
		log.TraceCheck(func() {
			log.For(ctx).Trace().Int("stop_id", id).Msg("tz: lookup failed, cant find timezone for stops with id=0 unless speciifed explicitly")
		})
		return nil, false
	}
	// Otherwise lookup the timezone
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
	if err := sqlx.Get(f.db, &tz, q, id); err != nil {
		log.For(ctx).Error().Err(err).Int("stop_id", id).Str("known", known).Msg("tz: lookup failed")
		return nil, false
	}
	loc, ok := f.tzCache.Add(id, tz)
	log.TraceCheck(func() {
		log.For(ctx).Trace().Int("stop_id", id).Str("known", known).Str("loc", loc.String()).Msg("tz: lookup successful")
	})
	return loc, ok
}

// FeedVersionTimezone looks up the timezone for a feed version using the first agency's timezone
func (f *lookupCache) FeedVersionTimezone(ctx context.Context, fvid int) (*time.Location, bool) {
	// Use a special key for feed version timezones (negative to avoid collision with stop IDs)
	cacheKey := -fvid
	if loc, ok := f.tzCache.Get(cacheKey); ok {
		return loc, ok
	}
	q := `SELECT agency_timezone FROM gtfs_agencies WHERE feed_version_id = $1 LIMIT 1`
	tz := ""
	if err := sqlx.Get(f.db, &tz, q, fvid); err != nil {
		log.For(ctx).Error().Err(err).Int("feed_version_id", fvid).Msg("tz: feed version timezone lookup failed")
		return nil, false
	}
	loc, ok := f.tzCache.Add(cacheKey, tz)
	return loc, ok
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
