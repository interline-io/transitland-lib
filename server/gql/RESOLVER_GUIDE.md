# GraphQL Resolver Guide

This guide covers the process of adding and maintaining resolvers in the transitland-lib GraphQL API, reflecting the patterns used in core resolvers like `Stop`, `Route`, `Agency`, and `FeedVersion`.

## Overview: Adding a New GTFS Entity

When adding a new GTFS entity type (e.g., `BookingRule`, `Location`), you'll need to modify multiple files across the codebase:

1. **GraphQL Schema** (`schema/graphql/schema.graphqls`)
2. **Generated Code** (run `go generate`)
3. **Model Types** (`server/model/models.go`)
4. **Filter Types** (generated in `server/model/generated_models.go`)
5. **Finder Interface** (`server/model/finders.go`)
6. **Database Finder** (`server/finders/dbfinder/<entity>.go`)
7. **Loader Params** (`server/gql/loader_params.go`)
8. **Loaders** (`server/gql/loaders.go`)
9. **Resolver** (`server/gql/<entity>_resolver.go`)
10. **Register Resolver** (`server/gql/resolver.go`)
11. **Tests** (`server/gql/gqltest/<entity>_cases.go`, plus the shared permission and cursor tests)

## 1. Adding New Resolvers

### Step 1: Define the Schema

The GraphQL schema definitions are located in `schema/graphql/`.

*   **New Entities**: Add new types to `schema/graphql/schema.graphqls` (or other relevant files). **Do not create new .graphql files.**
*   **Existing Entities**: Add fields to existing types in `schema/graphql/schema.graphqls`.
*   **Filter Types**: Add input types for filtering (e.g., `BookingRuleFilter`).

Example: Adding `Pathway` to `schema/graphql/schema.graphqls`:

```graphql
"""Record from a static GTFS [pathways.txt](https://gtfs.org/reference/static/#pathwaysstxt)."""
type Pathway {
  "Internal integer ID"
  id: Int!
  "GTFS pathways.pathway_id"
  pathway_id: String!
  # ... other fields
  "Pathway begins at this stop"
  from_stop: Stop!
  "Pathway ends at this stop"
  to_stop: Stop!
  "Feed version SHA1 associated with this entity"
  feed_version_sha1: String!
  "Feed Onestop ID associated with this entity"
  feed_onestop_id: String!
  "Source feed version for this entity"
  feed_version: FeedVersion!
}
```

Also add any filter types for querying:

```graphql
"""Search options for pathways"""
input PathwayFilter {
  "Restrict to specific ids"
  ids: [Int!]
  "Search for pathways with this pathway_id"
  pathway_id: String
}
```

Add the list field to the parent type. A `FeedVersion` list takes `limit`, `after` and `where`:

```graphql
type FeedVersion {
  # ... existing fields ...
  "Records from this feed version's `pathways.txt`, if imported"
  pathways(limit: Int, after: Int, where: PathwayFilter): [Pathway!]!
}
```

Give each new type the `feed_version`, `feed_version_sha1` and `feed_onestop_id` fields, and give each filter at least `ids`.

### Step 2: Generate Code

After modifying the schema, you must regenerate the Go code.

Run `go generate` in the `internal/generated/gqlout` directory.

```bash
(cd internal/generated/gqlout && go generate)
```

This generates:
- Resolver interfaces in `internal/generated/gqlout/`
- Filter types in `server/model/generated_models.go`

### Step 3: Define the Model Type

Add a model type to `server/model/models.go` that wraps the GTFS entity:

```go
type Pathway struct {
    FeedOnestopID   string
    FeedVersionSHA1 string
    gtfs.Pathway
}
```

The `FeedOnestopID` and `FeedVersionSHA1` fields are common metadata populated by the database finder joins. These fields are used by the GraphQL resolvers for `feed_onestop_id` and `feed_version_sha1` fields.

### Step 4: Add Finder Interface Methods

Add methods to the `EntityLoader` interface in `server/model/finders.go`:

```go
type EntityLoader interface {
    // ... existing methods ...

    PathwaysByIDs(context.Context, []int) ([]*Pathway, []error)
    PathwaysByFeedVersionIDs(context.Context, *int, *Cursor, *PathwayFilter, []int) ([][]*Pathway, error)
    PathwaysByFromStopIDs(context.Context, *int, *PathwayFilter, []int) ([][]*Pathway, error)
}
```

`UnimplementedFinder` in `server/model/unimplemented_finder.go` implements the full interface, so add a stub there for each new method.

**Method Signature Patterns**:
- **ByIDs**: `(ctx, ids []int) ([]*Entity, []error)` - Returns entities in same order as input IDs, with one error per entity
- **ByParentIDs**: `(ctx, limit *int, filter *FilterType, keys []int) ([][]*Entity, error)` - Returns slice of slices, one sub-slice per parent ID
- **ByFeedVersionIDs**: `(ctx, limit *int, after *Cursor, filter *FilterType, keys []int) ([][]*Entity, error)` - Like ByParentIDs, with an `after` cursor for paging through a large table

### Step 5: Implement Database Finder Methods

Create a new file `server/finders/dbfinder/pathway.go`. One select function builds the query for every finder of the table:

```go
package dbfinder

import (
    "context"

    "github.com/interline-io/transitland-lib/server/dbutil"
    "github.com/interline-io/transitland-lib/server/model"
    sq "github.com/irees/squirrel"
)

// PathwaysByFeedVersionIDs returns the pathways of each feed version.
func (f *Finder) PathwaysByFeedVersionIDs(ctx context.Context, limit *int, after *model.Cursor, where *model.PathwayFilter, keys []int) ([][]*model.Pathway, error) {
    var ents []*model.Pathway
    q := lateralWrap(pathwaySelect(limit, after, nil, f.PermFilter(ctx), where), "feed_versions", "id", "gtfs_pathways", "feed_version_id", keys)
    err := dbutil.Select(ctx, f.db, q, &ents)
    return arrangeGroup(keys, ents, func(ent *model.Pathway) int { return ent.FeedVersionID }), queryErr(ctx, err)
}

func (f *Finder) PathwaysByFromStopIDs(ctx context.Context, limit *int, where *model.PathwayFilter, keys []int) ([][]*model.Pathway, error) {
    var ents []*model.Pathway
    q := lateralWrap(pathwaySelect(limit, nil, nil, f.PermFilter(ctx), where), "gtfs_stops", "id", "gtfs_pathways", "from_stop_id", keys)
    err := dbutil.Select(ctx, f.db, q, &ents)
    return arrangeGroup(keys, ents, func(ent *model.Pathway) int { return ent.FromStopID.Int() }), queryErr(ctx, err)
}

func (f *Finder) PathwaysByIDs(ctx context.Context, ids []int) ([]*model.Pathway, []error) {
    var ents []*model.Pathway
    q := pathwaySelect(nil, nil, ids, f.PermFilter(ctx), nil)
    if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
        return nil, queryErrs(ctx, len(ids), err)
    }
    return arrangeBy(ids, ents, func(ent *model.Pathway) int { return ent.ID }), nil
}

func pathwaySelect(limit *int, after *model.Cursor, ids []int, permFilter *model.PermFilter, where *model.PathwayFilter) sq.SelectBuilder {
    // IMPORTANT: Enumerate all fields explicitly - never use SELECT *
    q := sq.StatementBuilder.Select(
        "gtfs_pathways.id",
        "gtfs_pathways.feed_version_id",
        "gtfs_pathways.pathway_id",
        "gtfs_pathways.from_stop_id",
        "gtfs_pathways.to_stop_id",
        // ... other fields ...
        "feed_versions.sha1 AS feed_version_sha1",
        "current_feeds.onestop_id AS feed_onestop_id",
    ).From("gtfs_pathways").
        Join("feed_versions ON feed_versions.id = gtfs_pathways.feed_version_id").
        Join("current_feeds ON current_feeds.id = feed_versions.feed_id")

    if len(ids) > 0 {
        q = q.Where(In("gtfs_pathways.id", ids))
    }
    if after != nil && after.Valid && after.ID > 0 {
        q = q.Where(sq.Gt{"gtfs_pathways.id": after.ID})
    }
    if where != nil {
        if len(where.Ids) > 0 {
            q = q.Where(In("gtfs_pathways.id", where.Ids))
        }
        if where.PathwayID != nil {
            q = q.Where(sq.Eq{"gtfs_pathways.pathway_id": *where.PathwayID})
        }
    }
    q = joinImported(q)
    q = pfJoinCheckFv(q, permFilter)
    return q.OrderBy("gtfs_pathways.id ASC").Limit(finderCheckLimit(limit))
}
```

Each select follows the same rules:
- **One select per table.** Every finder of the table calls it with its own arguments, so the columns, gates and filters stay the same across finders.
- **`lateralWrap` for lists under a parent.** It applies `limit` and `after` to each parent key, so a query across five feed versions returns up to `limit` records from each one, not `limit` in all.
- **`joinImported`, the import gate.** It hides a feed version's records unless its import succeeded and no import or unimport is running.
- **`pfJoinCheckFv`, the permission filter.** It hides the records of deleted feeds and feed versions, and of feeds the caller can't see. Apply it even under a parent that passed the check, because the `feed_version_fetch` mutation returns a feed version without checking permissions.
- **Ascending ids.** `after` takes the last id of the previous page, so the select orders by id.

**Helper Functions**:
- `arrangeGroup()`: Groups results by parent ID, preserving input order
- `arrangeBy()`: Orders results by requested IDs, filling nil for missing entities
- `queryErr()`, `queryErrs()`: Wrap a failed query's error for the finder to return, keeping the database's error text out of API responses

### Step 6: Implement the Resolver

If you added a new top-level type or query, `gqlgen` might have added a dummy implementation to `server/gql/schema.resolvers.go`. You should move this logic to a dedicated resolver file (e.g., `server/gql/pathway_resolver.go`) to keep the codebase organized.

1.  **Create the Resolver File**: `server/gql/pathway_resolver.go`
2.  **Define the Resolver Struct**:
    ```go
    package gql

    import (
        "context"
        "github.com/interline-io/transitland-lib/server/model"
    )

    type pathwayResolver struct{ *Resolver }

    func (r *pathwayResolver) FeedVersion(ctx context.Context, obj *model.Pathway) (*model.FeedVersion, error) {
        return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
    }

    func (r *pathwayResolver) FromStop(ctx context.Context, obj *model.Pathway) (*model.Stop, error) {
        return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.FromStopID.Int())()
    }

    func (r *pathwayResolver) ToStop(ctx context.Context, obj *model.Pathway) (*model.Stop, error) {
        return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.ToStopID.Int())()
    }
    ```
3.  **Register the Resolver**: In `server/gql/resolver.go`, add a method to return your resolver:
    ```go
    func (r *Resolver) Pathway() gqlout.PathwayResolver {
        return &pathwayResolver{r}
    }
    ```

### Step 7: Wire up Data Access (Loaders)

To make `LoaderFor(ctx)` work, you need to configure the data loaders.

1.  **`FeedVersion` Lists**: No new parameter type is needed. The generic `feedVersionTableParam[W]` in `server/gql/loader_params.go` carries the feed version ID, limit, `after` cursor and a filter of type `W`. Declare the loader in the `Loaders` struct in `server/gql/loaders.go`, and initialize it in `NewLoaders` with `withFeedVersionTable`:
    ```go
    type Loaders struct {
        // ...
        PathwaysByFeedVersionIDs *dataloader.Loader[feedVersionTableParam[*model.PathwayFilter], []*model.Pathway]
    }

    func NewLoaders(dbf model.Finder, batchSize int, stopTimeBatchSize int) *Loaders {
        loaders := &Loaders{
            // ...
            PathwaysByFeedVersionIDs: withFeedVersionTable(waitTime, batchSize, dbf.PathwaysByFeedVersionIDs),
        }
        return loaders
    }
    ```
    A finder that takes no filter goes through `cursorOnlyAdapter`, which uses `bool` for the unused filter: `withFeedVersionTable(waitTime, batchSize, cursorOnlyAdapter(dbf.FareProductsByFeedVersionIDs))`.

2.  **Other Parents**: Add a struct to `server/gql/loader_params.go` for a list under any other parent, such as a stop's pathways:
    ```go
    type pathwayLoaderParam struct {
        FromStopID int
        ToStopID   int
        Limit      *int
        Where      *model.PathwayFilter
    }
    ```
    Declare its loaders in the `Loaders` struct, and initialize them in `NewLoaders` with `withWaitAndCapacityGroup`, which maps the struct to the parent key, filter and limit:
    ```go
    type Loaders struct {
        // ...
        PathwaysByIDs         *dataloader.Loader[int, *model.Pathway]
        PathwaysByFromStopIDs *dataloader.Loader[pathwayLoaderParam, []*model.Pathway]
    }

    func NewLoaders(dbf model.Finder, batchSize int, stopTimeBatchSize int) *Loaders {
        loaders := &Loaders{
            // ...
            PathwaysByIDs: withWaitAndCapacity(waitTime, batchSize, dbf.PathwaysByIDs),
            PathwaysByFromStopIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.PathwaysByFromStopIDs,
                func(p pathwayLoaderParam) (int, *model.PathwayFilter, *int) {
                    return p.FromStopID, p.Where, p.Limit
                },
            ),
        }
        return loaders
    }
    ```

**Loader Helper Functions**:
- `withWaitAndCapacity(waitTime, batchSize, batchFn)`: For simple ByID loaders
- `withWaitAndCapacityGroup(waitTime, batchSize, queryFn, paramFn)`: For ByParentID loaders; requests that share a filter and limit run as one query
- `withFeedVersionTable(waitTime, batchSize, queryFn)`: For `FeedVersion` lists; requests that share a cursor, filter and limit run as one query
- `paramGroupAdapter`, `cursorGroupAdapter` and `cursorOnlyAdapter`: Adapt a finder that lacks a filter, or takes a cursor, to the grouped loaders

### Step 8: Add Tests

Add cases to `server/gql/gqltest/pathway_cases.go`, so every finder runs them, and register the builder in `caseGroups` in `gqltest.go`. Find records by stable keys, such as a feed version's sha1 or a GTFS ID, and look them up through `cfg.Finder` rather than SQL. Internal IDs depend on the order the fixtures load, so never hard-code them.

```go
package gqltest

import (
    "testing"

    "github.com/interline-io/transitland-lib/server/model"
)

func pathwayResolverTestcases(t testing.TB, cfg model.Config) []testcase {
    testcases := []testcase{
        {
            Name: "basic pathway query",
            Query: `query($sha1: String!) {
                feed_versions(where: {sha1: $sha1}) {
                    pathways(limit: 1) {
                        pathway_id
                        from_stop { stop_id }
                        to_stop { stop_id }
                    }
                }
            }`,
            Vars:              hw{"sha1": "..."},
            Selector:          "feed_versions.0.pathways.#.pathway_id",
            SelectExpectCount: 1,
        },
    }
    return testcases
}
```

Two shared tests cover every list:
- **`TestPermFilter_Negative`** (`server/gql/permfilter_negative_test.go`): Add each new finder to its `checks` table. The test requires each finder to return rows while the feeds are public, then expects none once every feed is private, so it fails for a select that skips `pfJoinCheckFv`.
- **`feedVersionResolverTableCursorsTest`** (`server/gql/gqltest/feed_version_cases.go`): Add each new `FeedVersion` list, with a feed version that has at least three records. The test checks ascending IDs, `after`, `ids`, and `after` with `ids`.

When a test checks that something gets filtered out, first assert that the unfiltered query returns rows, so the test can't pass without exercising the filter. `RESOLVER_TEST_GUIDE.md` documents the test harness.

## 2. Implementation Patterns

The core resolvers (`stop`, `route`, `agency`, etc.) follow specific patterns for data access, pagination, and filtering.

### Common Resolver Methods

Almost every entity resolver includes these common methods:

```go
// FeedVersion - returns the parent feed version
func (r *entityResolver) FeedVersion(ctx context.Context, obj *model.Entity) (*model.FeedVersion, error) {
    return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}
```

The `feed_version_sha1` and `feed_onestop_id` fields are typically populated directly from the model (set by the database finder joins) and don't need resolvers.

### Data Access: Loaders

Use `LoaderFor(ctx)` to access data loaders. This ensures efficient batching and caching.

Example from `server/gql/pathway_resolver.go`:

```go
func (r *pathwayResolver) FromStop(ctx context.Context, obj *model.Pathway) (*model.Stop, error) {
    return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.FromStopID.Int())()
}

func (r *pathwayResolver) ToStop(ctx context.Context, obj *model.Pathway) (*model.Stop, error) {
    return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.ToStopID.Int())()
}
```

For lists, use the appropriate loader method (often named `...By...IDs`) and pass a loader parameter struct.

```go
func (r *stopResolver) PathwaysFromStop(ctx context.Context, obj *model.Stop, limit *int) ([]*model.Pathway, error) {
    return LoaderFor(ctx).PathwaysByFromStopIDs.Load(ctx, pathwayLoaderParam{
        FromStopID: obj.ID, 
        Limit:      resolverCheckLimit(limit),
    })()
}
```

### Pagination (`limit` and `after`)

*   **Limit**: Use `resolverCheckLimit(limit)` to sanitize the limit argument (default 100, maximum 1,000).
*   **Higher Limits**: A `FeedVersion` list of GTFS records uses `resolverCheckLimitMax(limit, RESOLVER_GTFS_TABLE_MAXLIMIT)`, which allows up to 100,000 records per feed version, rather than a constant of its own. Other lists that need a higher limit define a constant in `resolver.go`.
*   **After**: The `after` argument is typically an `Int` representing the ID of the last item from the previous page (keyset pagination).

Top-level queries and `FeedVersion` lists use `checkCursor(after)` to convert the integer to a cursor model.

```go
// Top-level query example
func (r *queryResolver) Agencies(ctx context.Context, limit *int, after *int, ids []int, where *model.AgencyFilter) ([]*model.Agency, error) {
    cfg := model.ForContext(ctx)
    return cfg.Finder.FindAgencies(ctx, resolverCheckLimit(limit), checkCursor(after), ids, where)
}
```

### Filtering (`where`)

Use a dedicated input type (e.g., `AgencyFilter`) for filtering results. Pass this filter to the loader or finder.

```go
func (r *agencyResolver) Routes(ctx context.Context, obj *model.Agency, limit *int, where *model.RouteFilter) ([]*model.Route, error) {
    return LoaderFor(ctx).RoutesByAgencyIDs.Load(ctx, routeLoaderParam{
        AgencyID: obj.ID, 
        Limit:    resolverCheckLimit(limit), 
        Where:    where,
    })()
}
```

### Context and Finders

For operations that don't fit the loader pattern (e.g., complex searches or RT data), use `model.ForContext(ctx)` to access the `Finder` or `RTFinder`.

```go
func (r *routeResolver) Alerts(ctx context.Context, obj *model.Route, active *bool, limit *int, includeModes *bool, includeTrips *bool) ([]*model.Alert, error) {
    return model.ForContext(ctx).RTFinder.FindAlertsForRoute(ctx, obj, resolverCheckLimit(limit), active, includeModes != nil && *includeModes, includeTrips != nil && *includeTrips), nil
}
```

## 3. Core Resolver Examples

### Stop Resolver (`server/gql/stop_resolver.go`)

The `Stop` resolver demonstrates relationships to other entities (`FeedVersion`, `Level`, `Parent`) and lists of children (`Children`, `RouteStops`).

```go
func (r *stopResolver) FeedVersion(ctx context.Context, obj *model.Stop) (*model.FeedVersion, error) {
    return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

func (r *stopResolver) Children(ctx context.Context, obj *model.Stop, limit *int) ([]*model.Stop, error) {
    return LoaderFor(ctx).StopsByParentStopIDs.Load(ctx, stopLoaderParam{ParentStopID: obj.ID, Limit: resolverCheckLimit(limit)})()
}
```

### Route Resolver (`server/gql/route_resolver.go`)

The `Route` resolver shows how to handle geometry and filtered lists (`Trips`, `Stops`).

```go
func (r *routeResolver) Geometry(ctx context.Context, obj *model.Route) (*tt.Geometry, error) {
    if obj.Geometry.Valid {
        return &obj.Geometry, nil
    }
    // Defer geometry loading
    geoms, err := LoaderFor(ctx).RouteGeometriesByRouteIDs.Load(ctx, routeGeometryLoaderParam{RouteID: obj.ID})()
    // ...
}

func (r *routeResolver) Trips(ctx context.Context, obj *model.Route, limit *int, where *model.TripFilter) ([]*model.Trip, error) {
    return LoaderFor(ctx).TripsByRouteIDs.Load(ctx, tripLoaderParam{
        RouteID:       obj.ID,
        FeedVersionID: obj.FeedVersionID,
        Limit:         resolverCheckLimit(limit),
        Where:         where,
    })()
}
```

## 4. Real-World Example: Transfer

This section shows each change that added `FeedVersion.transfers`.

### Schema Definition (`schema/graphql/schema.graphqls`)

When adding a new entity, you need to:
1. Define the type itself
2. Add filter input type
3. Add fields to parent types that reference this entity

```graphql
"""
Record from a static GTFS [transfers.txt](https://gtfs.org/schedule/reference/#transferstxt) file.
"""
type Transfer {
  "Internal integer ID"
  id: Int!

  "GTFS `transfers.from_stop_id`"
  from_stop: Stop

  # ... to_stop, from_route, to_route, from_trip and to_trip

  "GTFS `transfers.transfer_type` [0=recommended transfer point, 1=timed transfer point, 2=requires `min_transfer_time`, 3=not possible, 4=in-seat transfer, 5=in-seat transfer not allowed]"
  transfer_type: Int!

  "GTFS `transfers.min_transfer_time`; seconds a rider needs to make the transfer"
  min_transfer_time: Int

  "Feed version SHA1 associated with this entity"
  feed_version_sha1: String!

  "Feed Onestop ID associated with this entity"
  feed_onestop_id: String!

  "Source feed version for this entity"
  feed_version: FeedVersion!
}

"""Search options for transfers"""
input TransferFilter {
  "Restrict to specific ids"
  ids: [Int!]
  "Search for transfers with this GTFS transfer_type"
  transfer_type: Int
}

# Also add to FeedVersion type:
type FeedVersion {
  # ... existing fields ...
  "Records from this feed version's `transfers.txt`, if imported"
  transfers(limit: Int, after: Int, where: TransferFilter): [Transfer!]!
}
```

A field is nullable wherever its database column allows NULL, as `from_stop` is here.

### Model Type (`server/model/models.go`)

```go
// Transfer is a transfers.txt record.
type Transfer struct {
    FeedOnestopID   string
    FeedVersionSHA1 string
    gtfs.Transfer
}
```

**Type Aliases**: For entities that share the same underlying data but have different GraphQL resolvers, use Go type aliases:

```go
// FlexStopTime is an alias for StopTime.
// Both types represent stop_times records and share the same underlying model.
// The separation exists only for GraphQL schema purposes where they have different resolvers.
type FlexStopTime = StopTime
```

**Field Names**: gqlgen binds each schema field to the Go field or method whose name matches. When the names differ, map the field in `gqlgen.yml` with `fieldName`, as `Translation.table_name` does for `gtfs.Translation`'s `TableNameValue`.

### Finder Interface (`server/model/finders.go`)

```go
type EntityLoader interface {
    // ...
    TransfersByFeedVersionIDs(context.Context, *int, *Cursor, *TransferFilter, []int) ([][]*Transfer, error)
}
```

The matching stub in `server/model/unimplemented_finder.go`:

```go
func (UnimplementedFinder) TransfersByFeedVersionIDs(context.Context, *int, *Cursor, *TransferFilter, []int) ([][]*Transfer, error) {
    return nil, notImplErr()
}
```

### Database Finder (`server/finders/dbfinder/transfer.go`)

```go
// TransfersByFeedVersionIDs returns the transfers of each feed version.
func (f *Finder) TransfersByFeedVersionIDs(ctx context.Context, limit *int, after *model.Cursor, where *model.TransferFilter, keys []int) ([][]*model.Transfer, error) {
    var ents []*model.Transfer
    q := lateralWrap(transferSelect(limit, after, f.PermFilter(ctx), where), "feed_versions", "id", "gtfs_transfers", "feed_version_id", keys)
    err := dbutil.Select(ctx, f.db, q, &ents)
    return arrangeGroup(keys, ents, func(ent *model.Transfer) int { return ent.FeedVersionID }), queryErr(ctx, err)
}

func transferSelect(limit *int, after *model.Cursor, permFilter *model.PermFilter, where *model.TransferFilter) sq.SelectBuilder {
    q := sq.StatementBuilder.Select(
        "gtfs_transfers.id",
        "gtfs_transfers.feed_version_id",
        "gtfs_transfers.from_stop_id",
        "gtfs_transfers.to_stop_id",
        "gtfs_transfers.from_route_id",
        "gtfs_transfers.to_route_id",
        "gtfs_transfers.from_trip_id",
        "gtfs_transfers.to_trip_id",
        "gtfs_transfers.transfer_type",
        "gtfs_transfers.min_transfer_time",
        "feed_versions.sha1 AS feed_version_sha1",
        "current_feeds.onestop_id AS feed_onestop_id",
    ).From("gtfs_transfers").
        Join("feed_versions ON feed_versions.id = gtfs_transfers.feed_version_id").
        Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
    if after != nil && after.Valid && after.ID > 0 {
        q = q.Where(sq.Gt{"gtfs_transfers.id": after.ID})
    }
    if where != nil {
        if len(where.Ids) > 0 {
            q = q.Where(In("gtfs_transfers.id", where.Ids))
        }
        if where.TransferType != nil {
            q = q.Where(sq.Eq{"gtfs_transfers.transfer_type": *where.TransferType})
        }
    }
    q = joinImported(q)
    q = pfJoinCheckFv(q, permFilter)
    return q.OrderBy("gtfs_transfers.id ASC").Limit(finderCheckLimit(limit))
}
```

### Loaders (`server/gql/loaders.go`)

A `FeedVersion` list needs no loader param of its own:

```go
// In Loaders struct:
TransfersByFeedVersionIDs *dataloader.Loader[feedVersionTableParam[*model.TransferFilter], []*model.Transfer]

// In NewLoaders():
TransfersByFeedVersionIDs: withFeedVersionTable(waitTime, batchSize, dbf.TransfersByFeedVersionIDs),
```

### Resolver (`server/gql/transfer_resolver.go`)

```go
package gql

import (
    "context"

    "github.com/interline-io/transitland-lib/server/model"
)

type transferResolver struct{ *Resolver }

func (r *transferResolver) FeedVersion(ctx context.Context, obj *model.Transfer) (*model.FeedVersion, error) {
    return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

func (r *transferResolver) FromStop(ctx context.Context, obj *model.Transfer) (*model.Stop, error) {
    if !obj.FromStopID.Valid {
        return nil, nil // Handle nullable field
    }
    return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.FromStopID.Int())()
}

// ToStop, FromRoute, ToRoute, FromTrip and ToTrip follow the same pattern.
```

### Register Resolver (`server/gql/resolver.go`)

```go
func (r *Resolver) Transfer() gqlout.TransferResolver {
    return &transferResolver{r}
}
```

### Add Field to Parent Resolver (`server/gql/feed_version_resolver.go`)

When you add a field to a parent type (like `transfers` on `FeedVersion`), you need to add the resolver method:

```go
func (r *feedVersionResolver) Transfers(ctx context.Context, obj *model.FeedVersion, limit *int, after *int, where *model.TransferFilter) ([]*model.Transfer, error) {
    return LoaderFor(ctx).TransfersByFeedVersionIDs.Load(ctx, feedVersionTableParam[*model.TransferFilter]{
        FeedVersionID: obj.ID,
        Limit:         resolverCheckLimitMax(limit, RESOLVER_GTFS_TABLE_MAXLIMIT),
        After:         checkCursor(after),
        Where:         where,
    })()
}
```

### Tests

`feedVersionResolverTransfersTestcases` in `server/gql/gqltest/transfer_cases.go` queries each field and the `transfer_type` filter on feed versions it finds by sha1. The finder also has an entry in each shared test:

```go
// TestPermFilter_Negative, in server/gql/permfilter_negative_test.go:
{"TransfersByFeedVersionIDs", func() (int, error) { return countGroups(f.TransfersByFeedVersionIDs(ctx, nil, nil, nil, []int{baFv})) }},

// feedVersionResolverTableCursorsTest, in server/gql/gqltest/feed_version_cases.go:
{"transfers", "e535eb2b3b9ac3ef15d82c56575e914575e732e0"},
```

## 5. Summary Checklist

When adding a new entity to GraphQL, ensure you complete all these steps:

- [ ] **Schema** (`schema/graphql/schema.graphqls`)
  - [ ] Add type definition with all fields, including `feed_version`, `feed_version_sha1` and `feed_onestop_id`
  - [ ] Add a filter input type that takes at least `ids`
  - [ ] Add fields to parent types (e.g., `FeedVersion.transfers`)

- [ ] **Generate Code**
  - [ ] Run `(cd internal/generated/gqlout && go generate)`

- [ ] **Model** (`server/model/models.go`)
  - [ ] Add model struct with `FeedOnestopID`, `FeedVersionSHA1` fields
  - [ ] Embed the underlying GTFS type
  - [ ] (Optional) Use type alias if sharing data with another GraphQL type

- [ ] **Finder Interface** (`server/model/finders.go`)
  - [ ] Add `EntityByIDs` method
  - [ ] Add `EntityByParentIDs` method(s), with an `after` cursor for `FeedVersion` lists
  - [ ] Add a stub for each new method to `server/model/unimplemented_finder.go`

- [ ] **Database Finder** (`server/finders/dbfinder/<entity>.go`)
  - [ ] Implement finder methods
  - [ ] Create one select function for the table, with explicit field enumeration (no `SELECT *`), and use it in every finder
  - [ ] Apply `joinImported` and `pfJoinCheckFv` in the select, and order by id
  - [ ] Wrap lists under a parent in `lateralWrap`
  - [ ] Use `arrangeBy()` and `arrangeGroup()` helpers

- [ ] **Loader Params** (`server/gql/loader_params.go`)
  - [ ] Use `feedVersionTableParam` for `FeedVersion` lists; add a param struct only for other parents

- [ ] **Loaders** (`server/gql/loaders.go`)
  - [ ] Add loader fields to `Loaders` struct
  - [ ] Initialize loaders in `NewLoaders()`, with `withFeedVersionTable` for `FeedVersion` lists

- [ ] **Resolver** (`server/gql/<entity>_resolver.go`)
  - [ ] Create resolver struct
  - [ ] Implement all field resolvers (especially `FeedVersion`)
  - [ ] Handle nullable fields with `.Valid` checks

- [ ] **Register Resolver** (`server/gql/resolver.go`)
  - [ ] Add resolver method

- [ ] **Parent Resolver** (e.g., `server/gql/feed_version_resolver.go`)
  - [ ] Add resolver method for the new field on the parent type
  - [ ] For a `FeedVersion` list, pass `resolverCheckLimitMax(limit, RESOLVER_GTFS_TABLE_MAXLIMIT)` and `checkCursor(after)`

- [ ] **Tests** (`server/gql/gqltest/<entity>_cases.go`)
  - [ ] Add test cases for basic queries
  - [ ] Test filters and relationships
  - [ ] Test accessing entity from parent (e.g., `feed_versions { transfers { ... } }`)
  - [ ] Find records by stable keys, never by hard-coded internal IDs
  - [ ] Add the finder to `TestPermFilter_Negative`
  - [ ] Add each `FeedVersion` list to `feedVersionResolverTableCursorsTest`
