# GraphQL DataLoader Pattern Guide

This document describes how to add a new GTFS entity to the GraphQL API using the DataLoader pattern. The pattern involves defining database access methods, registering batch loaders, and implementing GraphQL resolvers.

## Why DataLoaders?

**DataLoaders solve the N+1 query problem** that commonly affects GraphQL APIs:

- **Without DataLoaders**: Resolving a list of 100 trips would make 1 query to fetch trips, then 100 separate queries to fetch each trip's route → 101 queries total
- **With DataLoaders**: Same scenario makes 1 query for trips, then 1 batched query fetching all 100 routes at once → 2 queries total

**Additional Benefits**:
- **Request-scoped caching**: Same entity requested multiple times in one request is fetched only once
- **Automatic batching**: Multiple loader calls are automatically grouped and executed together
- **Simplified resolver code**: Resolvers don't need to worry about batching or caching logic

## Architecture Overview

The DataLoader pattern involves four main components:

```
GraphQL Query → Resolver → Loader → Finder → Database
                   ↓         ↓         ↓
                 (gql/)  (loaders.go) (dbfinder/)
```

### Component Responsibilities

1. **Finder Interface** (`server/model/finders.go`)
   - Defines methods for batch loading entities
   - Contract that all database implementations must satisfy
   - Three interface types: `EntityFinder`, `EntityLoader`, `EntityMutator`

2. **Database Finder** (`server/finders/dbfinder/`)
   - Implements Finder interface methods
   - Executes SQL queries using Squirrel query builder
   - Returns entities grouped by parent IDs or arranged by requested IDs

3. **Loaders** (`server/gql/loaders.go`)
   - Wraps Finder methods with DataLoader batching/caching
   - Configures batch timing (wait time) and size limits
   - Provides request-scoped loader instances via middleware

4. **Resolvers** (`server/gql/*_resolver.go`)
   - GraphQL field resolvers that use loaders to fetch data
   - Call `LoaderFor(ctx)` to get request-scoped loaders
   - Handle nullable fields and type conversions

## Step-by-Step: Adding a New Entity

This guide uses `Stop` as the example entity, demonstrating relationships with `Trip` and `StopTime`. Follow these steps to add any new GTFS entity to the GraphQL API.

### Step 1: Define the Finder Interface Methods

**File**: `server/model/finders.go`

Add two methods to the `EntityLoader` interface:

```go
type EntityLoader interface {
    // ... existing methods ...
    
    // Fetch multiple Stops by their IDs (primary key lookup)
    StopsByIDs(context.Context, []int) ([]*Stop, []error)
    
    // Fetch all Stops grouped by FeedVersion (one-to-many)
    StopsByFeedVersionIDs(context.Context, *int, []int) ([][]*Stop, error)
}
```

**Method Signatures**:
- **ByIDs**: Returns entities in same order as input IDs, with one error per entity
- **ByParentIDs**: Returns slice of slices, one sub-slice per parent ID

### Step 2: Implement Database Finder Methods

**File**: `server/finders/dbfinder/stop.go` (create new file)

```go
package dbfinder

import (
    "context"
    
    "github.com/interline-io/transitland-lib/server/dbutil"
    "github.com/interline-io/transitland-lib/server/model"
    sq "github.com/irees/squirrel"
)

// StopsByFeedVersionIDs loads Stops grouped by feed_version_id
func (f *Finder) StopsByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.Stop, error) {
    var ents []*model.Stop
    q := stopSelect(limit, nil, nil).Where(In("gtfs_stops.feed_version_id", keys))
    err := dbutil.Select(ctx, f.db, q, &ents)
    return arrangeGroup(keys, ents, func(ent *model.Stop) int { return ent.FeedVersionID }), err
}

// StopsByIDs loads specific Stops by their primary keys
func (f *Finder) StopsByIDs(ctx context.Context, ids []int) ([]*model.Stop, []error) {
    var ents []*model.Stop
    q := stopSelect(nil, nil, ids)
    if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
        return nil, []error{err}
    }
    return arrangeBy(ids, ents, func(ent *model.Stop) int { return ent.ID }), nil
}

// stopSelect builds the base SELECT query with explicit field enumeration
func stopSelect(limit *int, after *model.Cursor, ids []int) sq.SelectBuilder {
    q := sq.StatementBuilder.Select(
        // IMPORTANT: Enumerate all fields explicitly (security policy)
        "gtfs_stops.id",
        "gtfs_stops.feed_version_id",
        "gtfs_stops.created_at",
        "gtfs_stops.updated_at",
        "gtfs_stops.stop_id",
        "gtfs_stops.stop_name",
        "gtfs_stops.stop_lat",
        "gtfs_stops.stop_lon",
        "gtfs_stops.stop_code",
        "gtfs_stops.stop_desc",
        "gtfs_stops.zone_id",
        "gtfs_stops.stop_url",
        "gtfs_stops.location_type",
        "gtfs_stops.parent_station",
        "gtfs_stops.stop_timezone",
        "gtfs_stops.wheelchair_boarding",
        "gtfs_stops.platform_code",
        // ... all other fields ...
        "feed_versions.sha1 AS feed_version_sha1",
        "current_feeds.onestop_id AS feed_onestop_id",
    ).From("gtfs_stops").
        Join("feed_versions ON feed_versions.id = gtfs_stops.feed_version_id").
        Join("current_feeds ON current_feeds.id = feed_versions.feed_id")

    if len(ids) > 0 {
        q = q.Where(In("gtfs_stops.id", ids))
    }
    q = q.Limit(finderCheckLimit(limit))
    return q
}
```

**Important Notes**:
- **Security**: Never use `SELECT *` — enumerate all fields explicitly
- **Helper Functions**:
  - `arrangeGroup()`: Groups results by parent ID, preserving input order
  - `arrangeBy()`: Orders results by requested IDs, filling nil for missing entities
- **Joins**: Include `feed_versions` and `current_feeds` for common metadata fields

### Step 3: Register Loaders

**File**: `server/gql/loaders.go`

#### 3a. Add Loader Fields to the Struct

```go
type Loaders struct {
    // ... existing loaders ...
    
    StopsByIDs            *dataloader.Loader[int, *model.Stop]
    StopsByFeedVersionIDs *dataloader.Loader[stopLoaderParam, []*model.Stop]
}

// Define the loader param struct for group loaders
type stopLoaderParam struct {
    FeedVersionID int
    Limit         *int
    Where         *model.StopFilter
    // Add other ID fields as needed (e.g., RouteID, LevelID)
}
```

#### 3b. Initialize Loaders in NewLoaders()

```go
func NewLoaders(dbf model.Finder, batchSize int, stopTimeBatchSize int) *Loaders {
    // batchSize defaults to maxBatch (100) if 0
    // stopTimeBatchSize defaults to maxBatch (100) if 0
    return &Loaders{
        // ... existing loaders ...
        
        StopsByIDs: withWaitAndCapacity(waitTime, batchSize, dbf.StopsByIDs),
        
        StopsByFeedVersionIDs: withWaitAndCapacityGroup(waitTime, batchSize, 
            dbf.StopsByFeedVersionIDs,
            func(p stopLoaderParam) (int, *model.StopFilter, *int) {
                return p.FeedVersionID, p.Where, p.Limit
            },
        ),
    }
}
```

**Loader Configuration**:
- **Constants defined in loaders.go**:
  - `waitTime = 2 * time.Millisecond`
  - `maxBatch = 100`
  - `stopTimeBatchWaitTime = 10 * time.Millisecond`
- `withWaitAndCapacity(waitTime, batchSize, batchFn)`: For byID loaders
- `withWaitAndCapacityGroup(waitTime, batchSize, queryFn, paramFn)`: For byParentID loaders
  - `queryFn`: The finder method signature
  - `paramFn`: Extracts (key, where, limit) from the loader param struct

### Step 4: Create GraphQL Resolver

**File**: `server/gql/stop_resolver.go` (create new file)

```go
package gql

import (
    "context"
    
    "github.com/interline-io/transitland-lib/server/model"
)

type stopResolver struct{ *Resolver }

// FeedVersion returns the parent FeedVersion for this Stop
func (r *stopResolver) FeedVersion(ctx context.Context, obj *model.Stop) (*model.FeedVersion, error) {
    return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

// ParentStation returns the parent stop (for stations)
func (r *stopResolver) ParentStation(ctx context.Context, obj *model.Stop) (*model.Stop, error) {
    if !obj.ParentStation.Valid {
        return nil, nil
    }
    return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.ParentStation.Int())()
}

// StopTimes returns all StopTimes for this Stop
func (r *stopResolver) StopTimes(ctx context.Context, obj *model.Stop, limit *int) ([]*model.StopTime, error) {
    return LoaderFor(ctx).StopTimesByStopIDs.Load(ctx, stopTimeLoaderParam{
        FeedVersionID: obj.FeedVersionID,
        StopID:        obj.ID,
        Limit:         limit,
    })()
}
```

**Resolver Patterns**:

1. **Simple Foreign Key** (e.g., `FeedVersion`):
   ```go
   return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
   ```

2. **Nullable Fields** (e.g., `ParentStation`):
   ```go
   if !obj.ParentStation.Valid {
       return nil, nil  // Handle null case
   }
   return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.ParentStation.Int())()
   ```

3. **One-to-Many** (e.g., `StopTimes` for a Stop):
   ```go
   return LoaderFor(ctx).StopTimesByStopIDs.Load(ctx, stopTimeLoaderParam{
       FeedVersionID: obj.FeedVersionID,
       StopID:        obj.ID,
       Where:         where,
       Limit:         limit,
   })()
   ```

4. **One-to-Many on FeedVersion** (e.g., all Stops):
   ```go
   return LoaderFor(ctx).StopsByFeedVersionIDs.Load(ctx, stopLoaderParam{
       FeedVersionID: obj.ID,
       Where:         where,
       Limit:         limit,
   })()
   ```

### Step 5: Register Resolver with GraphQL

**File**: `server/gql/resolver.go`

Add a method to return your resolver:

```go
func (r *Resolver) Stop() gqlout.StopResolver { 
    return &stopResolver{r} 
}
```

This connects the GraphQL type to your resolver implementation.

### Step 6: Add GraphQL Query Resolvers (Optional)

If you want top-level queries (e.g., `query { stops { ... } }`), add to the Query resolver:

**File**: `server/gql/query_resolver.go`

```go
func (r *queryResolver) Stops(ctx context.Context, limit *int, after *model.Cursor, ids []int, where *model.StopFilter) ([]*model.Stop, error) {
    // Use FindStops from EntityFinder interface
    return r.finder.FindStops(ctx, limit, after, ids, where)
}
```

**Note**: This requires implementing `FindStops()` in the `EntityFinder` interface and dbfinder.

## Loader Patterns Reference

### Pattern 1: Simple ByID Lookup

**Use Case**: Fetching single entities by primary key

```go
type Loaders struct {
    AgenciesByIDs *dataloader.Loader[int, *model.Agency]
}

// In NewLoaders():
AgenciesByIDs: withWaitAndCapacity(waitTime, batchSize, dbf.AgenciesByIDs),

// In resolver:
return LoaderFor(ctx).AgenciesByIDs.Load(ctx, obj.AgencyID)()
```

### Pattern 2: Group By Parent ID

**Use Case**: One-to-many relationships (e.g., all routes for an agency)

```go
type Loaders struct {
    RoutesByAgencyIDs *dataloader.Loader[routeLoaderParam, []*model.Route]
}

// In NewLoaders():
RoutesByAgencyIDs: withWaitAndCapacityGroup(waitTime, batchSize, 
    dbf.RoutesByAgencyIDs,
    func(p routeLoaderParam) (int, *model.RouteFilter, *int) {
        return p.AgencyID, p.Where, p.Limit
    },
),

// In resolver:
return LoaderFor(ctx).RoutesByAgencyIDs.Load(ctx, routeLoaderParam{
    AgencyID: obj.ID,
    Limit:    limit,
    Where:    where,
})()
```

### Pattern 3: Composite Key Lookup

**Use Case**: Looking up by multiple fields (e.g., feed_version_id + stop_id)

```go
// Define key type for stop times within a specific feed version
type stopTimeLoaderParam struct {
    FeedVersionID int
    StopID        int
    Limit         *int
    Where         *model.StopTimeFilter
}

type Loaders struct {
    StopTimesByStopIDs *dataloader.Loader[stopTimeLoaderParam, []*model.StopTime]
}

// In NewLoaders():
// Note: stopTimeBatchWaitTime (10ms) is used for large entity batches
StopTimesByStopIDs: withWaitAndCapacityGroup(stopTimeBatchWaitTime, stopTimeBatchSize, 
    dbf.StopTimesByStopIDs,
    func(p stopTimeLoaderParam) (model.FVPair, *model.StopTimeFilter, *int) {
        return model.FVPair{FeedVersionID: p.FeedVersionID, EntityID: p.StopID}, p.Where, p.Limit
    },
),

// In resolver:
return LoaderFor(ctx).StopTimesByStopIDs.Load(ctx, stopTimeLoaderParam{
    FeedVersionID: obj.FeedVersionID,
    StopID:        obj.ID,
    Where:         where,
    Limit:         limit,
})()
```

### Pattern 4: Trip → StopTimes → Stops Relationship

**Use Case**: The classic GTFS pattern - getting stops for a trip via stop_times

```go
// StopTime resolver - get the Stop for a StopTime
func (r *stopTimeResolver) Stop(ctx context.Context, obj *model.StopTime) (*model.Stop, error) {
    return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.StopID)()
}

// Trip resolver - get all StopTimes for a Trip
func (r *tripResolver) StopTimes(ctx context.Context, obj *model.Trip, limit *int) ([]*model.StopTime, error) {
    return LoaderFor(ctx).StopTimesByTripIDs.Load(ctx, stopTimeLoaderParam{
        FeedVersionID: obj.FeedVersionID,
        TripID:        obj.ID,
        Limit:         limit,
    })()
}

// Stop resolver - get all StopTimes at this Stop
func (r *stopResolver) StopTimes(ctx context.Context, obj *model.Stop, limit *int) ([]*model.StopTime, error) {
    return LoaderFor(ctx).StopTimesByStopIDs.Load(ctx, stopTimeLoaderParam{
        FeedVersionID: obj.FeedVersionID,
        StopID:        obj.ID,
        Limit:         limit,
    })()
}
```

### Pattern 5: Parameterized Group Queries

**Use Case**: One-to-many with filters (e.g., stop_times with limit)

```go
type stopTimeLoaderParam struct {
    FeedVersionID int
    TripID        int
    Limit         *int
    Where         *model.TripStopTimeFilter
}

type Loaders struct {
    StopTimesByTripIDs *dataloader.Loader[tripStopTimeLoaderParam, []*model.StopTime]
}

// In NewLoaders():
// The withWaitAndCapacityGroup helper handles all the batching/grouping complexity
StopTimesByTripIDs: withWaitAndCapacityGroup(waitTime, batchSize, 
    dbf.StopTimesByTripIDs,
    func(p tripStopTimeLoaderParam) (model.FVPair, *model.TripStopTimeFilter, *int) {
        return model.FVPair{FeedVersionID: p.FeedVersionID, EntityID: p.TripID}, p.Where, p.Limit
    },
),

// In resolver:
return LoaderFor(ctx).StopTimesByTripIDs.Load(ctx, tripStopTimeLoaderParam{
    FeedVersionID: obj.FeedVersionID,
    TripID:        obj.ID,
    Where:         where,
    Limit:         limit,
})()
```

## Common Configurations

### Batch Timing

- **Default**: 2ms wait time (`waitTime` constant)
  - Good for most entities (routes, agencies, calendars)

- **Large entities**: 10ms wait time (`stopTimeBatchWaitTime` constant)
  - Use for stop_times, shapes (entities with many records per parent)

### Batch Size

- **Default**: 100 entities
  - Suitable for most use cases
- **Large batches**: 500+ entities
  - Use when queries are simple and database can handle larger batches
  - Example: stop_times often configured with 500+ batch size

### When to Customize

```go
// Default configuration (most cases)
withWaitAndCapacity(waitTime, batchSize, dbf.EntityByIDs)

// Custom timing and size (large entities like stop_times)
withWaitAndCapacityGroup(stopTimeBatchWaitTime, stopTimeBatchSize, queryFn, paramFn)

// Full manual control
dataloader.NewBatchedLoader(
    unwrapResult(batchFn),
    dataloader.WithWait[ParamT, T](5*time.Millisecond),
    dataloader.WithBatchCapacity[ParamT, T](1000),
)
```

## Testing

### Unit Tests

Create resolver tests in `server/gql/booking_rule_resolver_test.go`:

```go
package gql

import (
    "context"
    "testing"
)

func TestBookingRuleResolver_FeedVersion(t *testing.T) {
    cfg := testcfg()
    resolver := cfg.Resolver
    ctx := cfg.Context()
    
    // Query a BookingRule
    q := `query($ids: [Int!]!) { 
        booking_rules(ids: $ids) { 
            booking_rule_id 
            feed_version { sha1 }
        } 
    }`
    
    vars := hw{"ids": hw{"1"}}
    
    // Execute query
    got := cfg.QueryTestcase(t, q, vars)
    
    // Verify results
    // ... assertions ...
}
```

### Integration Tests

Loaders are automatically tested through GraphQL queries. If your resolver works, your loader works.

## Troubleshooting

### Problem: "Interface method not implemented"

**Cause**: Forgot to add method to Finder interface or didn't implement in dbfinder

**Solution**:
1. Check `server/model/finders.go` has method in `EntityLoader` interface
2. Check `server/finders/dbfinder/<entity>.go` implements method on `*Finder`

### Problem: "Results returned in wrong order"

**Cause**: Not using `arrangeBy()` or `arrangeGroup()` helper functions

**Solution**: These functions ensure results match the order of input IDs:
```go
return arrangeBy(ids, ents, func(ent *model.Entity) int { return ent.ID }), nil
```

### Problem: "Loader not found in context"

**Cause**: Forgot to register loader in `NewLoaders()`

**Solution**: Add loader initialization in `server/gql/loaders.go`:
```go
MyEntityByIDs: withWaitAndCapacity(waitTime, batchSize, dbf.MyEntityByIDs),
```

### Problem: "SELECT * security violation"

**Cause**: Using wildcard SELECT in query builder

**Solution**: Enumerate all fields explicitly:
```go
q := sq.StatementBuilder.Select(
    "table.id",
    "table.field1",
    "table.field2",
    // ... all other fields
)
```

### Problem: "Loader returns nil for valid ID"

**Cause**: Missing entity in database, or wrong JOIN causing row exclusion

**Solution**:
- Check if entity exists: `SELECT * FROM table WHERE id = ?`
- Verify JOINs are correct (use LEFT JOIN if parent might be missing)
- Check `arrangeBy()` is filling nil for missing entities correctly

## Advanced Topics

### Custom Loader Keys

For complex lookups beyond simple IDs, define custom parameter types:

```go
type customLoaderParam struct {
    ID        int
    Filter    string
    Limit     *int
}

// Use as loader key type
MyCustomLoader *dataloader.Loader[customLoaderParam, *model.Entity]
```

### Loader Middleware

Loaders are request-scoped via middleware in `server/gql/loaders.go`:

```go
func loaderMiddleware(finder model.Finder, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := context.WithValue(r.Context(), loadersKey, NewLoaders(r.Context(), finder))
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

This ensures:
- New loader instances per request (no cross-request caching)
- Request context passed through entire chain
- Loaders properly garbage collected after request

### Bypassing Loaders

For rare cases where batching doesn't help (e.g., single entity in entire request), you can call finder directly:

```go
// In resolver - bypass loader for one-off query
entity, err := r.finder.FindEntityByID(ctx, id)
```

**Warning**: Only do this when you're certain the entity won't be requested multiple times.

## Summary Checklist

When adding a new entity to GraphQL:

- [ ] Add methods to `EntityLoader` interface in `server/model/finders.go`
- [ ] Implement methods in `server/finders/dbfinder/<entity>.go`
  - [ ] Use explicit field enumeration (no `SELECT *`)
  - [ ] Use `arrangeBy()` or `arrangeGroup()` for result ordering
- [ ] Register loaders in `server/gql/loaders.go`
  - [ ] Add fields to `Loaders` struct
  - [ ] Initialize in `NewLoaders()` function
- [ ] Create resolver in `server/gql/<entity>_resolver.go`
  - [ ] Handle nullable fields
  - [ ] Use `LoaderFor(ctx)` to get loaders
- [ ] Register resolver in `server/gql/resolver.go`
- [ ] Write tests in `server/gql/<entity>_resolver_test.go`
- [ ] (Optional) Add top-level query in `server/gql/query_resolver.go`

## Additional Resources

- **GraphQL Schema**: See `server/gql/schema/*.graphql` for entity relationship design
- **DataLoader Library**: https://github.com/graph-gophers/dataloader
- **Squirrel Query Builder**: https://github.com/Masterminds/squirrel (note: using irees fork)

---

*This guide is a living document. If you find steps unclear or discover patterns not covered here, please update this file.*

