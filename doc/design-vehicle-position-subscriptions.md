# Vehicle Position Subscriptions

Design document for real-time vehicle position streaming via GraphQL subscriptions.

Branch: `vp-2`

## Goal

Allow clients to subscribe to live GTFS-RT vehicle position updates over WebSocket, with filtering by geography and feed. Positions are pushed as full snapshots on each RT update cycle.

## Architecture

```
RT Feed Sources
    |
    v
Cache (Local or Redis)
    |  pub/sub notification on update
    v
Subscription Resolver
    |  collect + filter positions from all cached feeds
    v
WebSocket (graphql-ws protocol)
    |
    v
Client
```

Key design choice: the subscription resolver reads only from the RT cache, not the database. This keeps the hot path fast and avoids DB load per push, but limits filtering to data available in the GTFS-RT messages themselves.

## GraphQL Schema

```graphql
type Subscription {
  vehicle_positions(where: VehiclePositionFilter): [VehiclePosition!]!
}

input VehiclePositionFilter {
  bbox: BoundingBox
  feed_onestop_ids: [String!]
  limit: Int
}
```

Extended `VehiclePosition` type fields: `trip`, `feed_onestop_id`, `bearing`, `speed`, `stop_id` (as String, no DB lookup).

## Work Items

### Infrastructure

- [x] WebSocket transport in gqlgen server (gorilla/websocket, keepalive pings)
- [x] `wsAwareTimeout` middleware bypasses `http.TimeoutHandler` for WebSocket upgrades (single router, no middleware duplication)
- [x] Hijack() support on responseWriterWrapper for meters middleware
- [x] GraphiQL v3 upgrade with native subscription support

### Cache Pub/Sub

- [x] `Subscribe()` and `GetSourceKeys()` on Cache interface
- [x] LocalCache in-memory subscriber notification
- [x] RedisCache PSUBSCRIBE-based notification
- [x] Clean up unused `subscribers` map and `notifySubscribers` on RedisCache (dead code removed)
- [ ] Consider debounce/coalesce: multiple RT feed updates within a short window should produce one push, not N

### Vehicle Position Parsing

- [x] Parse `ent.Vehicle` in `Source.processMessage` (was a TODO)
- [x] `GetVehiclePositions()` on Source, Finder, and RTFinder interface
- [x] `GetCachedFeedIDs()` to discover feeds with VP data from cache keys
- [ ] **Thread safety (near-term):** Add `sync.RWMutex` to `Source` struct. `processMessage` is called from background goroutines (LocalCache.AddData, Redis listener) while `GetVehiclePositions`, `GetTrip`, and `GetTimestamp` are called concurrently from request/subscription goroutines. The fields (`entityByTrip`, `alerts`, `vehiclePositions`) are replaced non-atomically — readers can see inconsistent state across fields. This is a pre-existing race for trip updates and alerts; adding vehicle positions extends the surface. Fix: write lock in `processMessage`, read lock in all getters.

### Subscription Resolver

- [x] `subscriptionResolver.VehiclePositions` — subscribe, initial snapshot, ongoing pushes
- [x] `convertVehiclePosition` — full protobuf-to-model mapping (position, bearing, speed, vehicle, trip, stop info, timestamp)
- [x] `matchesFilter` — bbox filtering
- [x] `feed_onestop_ids` filtering (at feed level before collecting)

### Filters — Remaining

- [x] `agency_ids`: Removed from schema — requires DB lookup which conflicts with cache-only design. May revisit.
- [x] `route_ids`: Removed from schema — depends on optional RT field, keeping surface simple. May revisit.
- [x] `stop_id`: Changed from `Stop` (object, DB lookup) to `String` (raw GTFS-RT value) to keep subscription path DB-free.
- [x] `limit` on VehiclePositionFilter — defaults to 1000 (RESOLVER_MAXLIMIT), capped at 1000

### Performance

- [ ] Scope notifications: currently any cache update (trip updates, alerts, etc.) triggers a full vehicle position collection for all subscribers. Filter to only `realtime_vehicle_positions` topics before collecting.
- [ ] Debounce: coalesce rapid updates within a window (e.g., 500ms-1s) before pushing
- [ ] RedisCache `GetSourceKeys` does a SCAN on every notification per subscriber. Cache key list in memory with short TTL, or maintain incrementally.

### Observability / Security

- [x] WebSocket requests now go through the same middleware stack as HTTP (single router)
- [x] WebSocket upgrader `CheckOrigin` allows all origins — acceptable because all connections require explicit API key/auth header (no cookie/session auth)
- [ ] Add subscription connection count metrics / logging for monitoring

### Testing

- [x] Integration tests via gqlgen WebSocket test client (`subscription_resolver_test.go`):
  - Full field mapping (feed_onestop_id, bearing, speed, position, vehicle, trip)
  - Feed filter (two feeds, filter to one)
  - Bbox filter (matching and non-matching)
  - Live update (empty initial snapshot, push data, verify update arrives)
- [ ] Unit tests for `convertVehiclePosition` (pure function, easy to test)
- [ ] Unit tests for `GetCachedFeedIDs` key parsing
- [ ] Verify existing RT resolver tests still pass with Cache interface changes

### Dev Tooling

- [x] `cmd/rt-test-server` — synthetic VP generator + optional real RT feed fetcher
- [x] `testdata/dmfr/rt-test.dmfr.json` fixture

## Design Decisions

- **Cache-only, no DB in hot path.** The subscription resolver reads exclusively from the RT cache. This avoids DB load per push and keeps latency low. It constrains what filtering and field resolution is possible — filters that require static GTFS data (agency_ids, route_ids) and fields that resolve to DB objects (stop_id as Stop) were removed or simplified to match.
- **`stop_id` is a raw String, not a resolved Stop object.** The rest of the GraphQL API resolves stop_id to a Stop entity via DB lookup. The subscription intentionally returns the raw GTFS-RT string to avoid DB dependencies. A resolved `stop` field could be added later as an opt-in.
- **`agency_ids` and `route_ids` removed from filter.** Both require joining against static GTFS data. Keeping the filter surface to what the cache can deliver (bbox, feed_onestop_ids, limit).
- **Single router with `wsAwareTimeout`.** Standard Go pattern — `http.TimeoutHandler` is incompatible with WebSocket (no Hijack support, kills long-lived connections). A single middleware skips the timeout for Upgrade requests. All other middleware (auth, CORS, meters, logging, permissions) is shared.
- **`CheckOrigin` allows all origins.** Safe because auth is API-key-based (explicit header), not cookie/session-based. No ambient browser credentials to exploit via CSRF.
- **Full snapshot per push, not deltas.** Simpler for consumers (replace the whole list, no client-side state management). Trade-off is bandwidth for large fleets.
- **Default limit of 1000 per push.** Uses existing `RESOLVER_MAXLIMIT`. Prevents unbounded responses from unfiltered subscriptions.
- **gorilla/websocket.** Archived/unmaintained, but it's gqlgen's transitive dependency. No alternative without switching gqlgen's transport.

## Open Questions

1. **Snapshot vs. delta**: Currently sends full snapshot of all matching positions on every update. Should this eventually support deltas (added/changed/removed)?
2. **Rate limiting**: Should there be a max subscription count per client or global?
3. **gorilla/websocket**: Archived/unmaintained, but it's a transitive dependency via gqlgen's `transport.Websocket`. No action needed unless gqlgen migrates (likely to `coder/websocket`).
