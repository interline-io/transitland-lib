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
11. **Tests** (`server/gql/<entity>_resolver_test.go`)

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
    PathwaysByFromStopIDs(context.Context, *int, *PathwayFilter, []int) ([][]*Pathway, error)
}
```

**Method Signature Patterns**:
- **ByIDs**: `(ctx, ids []int) ([]*Entity, []error)` - Returns entities in same order as input IDs, with one error per entity
- **ByParentIDs**: `(ctx, limit *int, filter *FilterType, keys []int) ([][]*Entity, error)` - Returns slice of slices, one sub-slice per parent ID

### Step 5: Implement Database Finder Methods

Create a new file `server/finders/dbfinder/pathway.go`:

```go
package dbfinder

import (
    "context"
    
    "github.com/interline-io/transitland-lib/server/dbutil"
    "github.com/interline-io/transitland-lib/server/model"
    sq "github.com/irees/squirrel"
)

func (f *Finder) PathwaysByFromStopIDs(ctx context.Context, limit *int, where *model.PathwayFilter, keys []int) ([][]*model.Pathway, error) {
    var ents []*model.Pathway
    q := pathwaySelect(limit, nil, nil, where).Where(In("gtfs_pathways.from_stop_id", keys))
    err := dbutil.Select(ctx, f.db, q, &ents)
    return arrangeGroup(keys, ents, func(ent *model.Pathway) int { return ent.FromStopID.Int() }), err
}

func (f *Finder) PathwaysByIDs(ctx context.Context, ids []int) ([]*model.Pathway, []error) {
    var ents []*model.Pathway
    q := pathwaySelect(nil, nil, ids, nil)
    if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
        return nil, []error{err}
    }
    return arrangeBy(ids, ents, func(ent *model.Pathway) int { return ent.ID }), nil
}

func pathwaySelect(limit *int, _ *model.Cursor, ids []int, where *model.PathwayFilter) sq.SelectBuilder {
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
    if where != nil {
        if where.PathwayID != nil && *where.PathwayID != "" {
            q = q.Where(sq.Eq{"gtfs_pathways.pathway_id": *where.PathwayID})
        }
    }
    q = q.OrderBy("gtfs_pathways.id ASC")
    q = q.Limit(finderCheckLimit(limit))
    return q
}
```

**Helper Functions**:
- `arrangeGroup()`: Groups results by parent ID, preserving input order
- `arrangeBy()`: Orders results by requested IDs, filling nil for missing entities

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

1.  **Define Loader Parameters**: Add a struct to `server/gql/loader_params.go` for group loaders that need filtering or arguments:
    ```go
    type pathwayLoaderParam struct {
        FeedVersionID int
        FromStopID    int
        Limit         *int
        Where         *model.PathwayFilter
    }
    ```

2.  **Add to Loaders Struct**: Add the loader definition to the `Loaders` struct in `server/gql/loaders.go`:
    ```go
    type Loaders struct {
        // ...
        PathwaysByIDs            *dataloader.Loader[int, *model.Pathway]
        PathwaysByFromStopIDs    *dataloader.Loader[pathwayLoaderParam, []*model.Pathway]
    }
    ```

3.  **Initialize Loader**: Initialize the loader in `NewLoaders` in `server/gql/loaders.go`. This connects the loader to the underlying `Finder` method:
    ```go
    func NewLoaders(dbf model.Finder, batchSize int, stopTimeBatchSize int) *Loaders {
        loaders := &Loaders{
            // ...
            PathwaysByIDs: withWaitAndCapacity(waitTime, batchSize, dbf.PathwaysByIDs),
            PathwaysByFromStopIDs: withWaitAndCapacityGroup(waitTime, batchSize, 
                dbf.PathwaysByFromStopIDs,
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
- `withWaitAndCapacityGroup(waitTime, batchSize, queryFn, paramFn)`: For ByParentID loaders with grouping

### Step 8: Add Tests

Create tests in `server/gql/pathway_resolver_test.go`:

```go
package gql

import (
    "testing"
)

func TestPathwayResolver(t *testing.T) {
    testcases := []testcase{
        {
            name: "basic pathway query",
            query: `query {
                feed_versions(where: {sha1: "..."}) {
                    pathways(limit: 1) {
                        pathway_id
                        from_stop { stop_id }
                        to_stop { stop_id }
                    }
                }
            }`,
            selector: "feed_versions.0.pathways.#.pathway_id",
            selectExpectCount: 1,
        },
    }
    c, _ := newTestClient(t)
    for _, tc := range testcases {
        t.Run(tc.name, func(t *testing.T) {
            queryTestcase(t, c, tc)
        })
    }
}
```

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

*   **Limit**: Use `resolverCheckLimit(limit)` to sanitize the limit argument (default max: 100).
*   **Higher Limits**: For entities that may have many records, use `resolverCheckLimitMax(limit, RESOLVER_<ENTITY>_MAXLIMIT)` with a custom max constant defined in `resolver.go`.
*   **After**: The `after` argument is typically an `Int` representing the ID of the last item from the previous page (keyset pagination).

Top-level queries often use `checkCursor(after)` to convert the integer to a cursor model.

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
func (r *routeResolver) Alerts(ctx context.Context, obj *model.Route, active *bool, limit *int) ([]*model.Alert, error) {
    return model.ForContext(ctx).RTFinder.FindAlertsForRoute(ctx, obj, resolverCheckLimit(limit), active), nil
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

## 4. Real-World Example: BookingRule

This section shows a complete example from the GTFS Flex implementation.

### Schema Definition (`schema/graphql/schema.graphqls`)

When adding a new entity, you need to:
1. Define the type itself
2. Add filter input type  
3. Add fields to parent types that reference this entity

```graphql
"""Record from a static GTFS [booking_rules.txt](https://gtfs.org/schedule/reference/#booking_rulestxt) file."""
type BookingRule {
  "Internal integer ID"
  id: Int!
  "GTFS booking_rules.booking_rule_id"
  booking_rule_id: String!
  "GTFS booking_rules.booking_type"
  booking_type: Int!
  "GTFS booking_rules.prior_notice_duration_min"
  prior_notice_duration_min: Int
  "Prior notice service calendar"
  prior_notice_service: Calendar
  "Feed version"
  feed_version: FeedVersion!
  # ... other fields
}

input BookingRuleFilter {
  "Restrict to specific ids"
  ids: [Int!]
  "Search for booking rules with this booking_rule_id"
  booking_rule_id: String
}

# Also add to FeedVersion type:
type FeedVersion {
  # ... existing fields ...
  "GTFS Flex booking rules associated with this feed version, if imported"
  booking_rules(limit: Int, where: BookingRuleFilter): [BookingRule!]!
}
```

### Model Type (`server/model/models.go`)

```go
type BookingRule struct {
    FeedOnestopID   string
    FeedVersionSHA1 string
    gtfs.BookingRule
}
```

**Type Aliases**: For entities that share the same underlying data but have different GraphQL resolvers, use Go type aliases:

```go
// FlexStopTime is an alias for StopTime.
// Both types represent stop_times records and share the same underlying model.
// The separation exists only for GraphQL schema purposes where they have different resolvers.
type FlexStopTime = StopTime
```

### Finder Interface (`server/model/finders.go`)

```go
type EntityLoader interface {
    // ...
    BookingRulesByFeedVersionIDs(context.Context, *int, *BookingRuleFilter, []int) ([][]*BookingRule, error)
    BookingRulesByIDs(context.Context, []int) ([]*BookingRule, []error)
}
```

### Database Finder (`server/finders/dbfinder/booking_rule.go`)

```go
func (f *Finder) BookingRulesByFeedVersionIDs(ctx context.Context, limit *int, where *model.BookingRuleFilter, keys []int) ([][]*model.BookingRule, error) {
    var ents []*model.BookingRule
    q := bookingRuleSelect(limit, nil, nil, where).Where(In("gtfs_booking_rules.feed_version_id", keys))
    err := dbutil.Select(ctx, f.db, q, &ents)
    return arrangeGroup(keys, ents, func(ent *model.BookingRule) int { return ent.FeedVersionID }), err
}

func (f *Finder) BookingRulesByIDs(ctx context.Context, ids []int) ([]*model.BookingRule, []error) {
    var ents []*model.BookingRule
    q := bookingRuleSelect(nil, nil, ids, nil)
    if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
        return nil, []error{err}
    }
    return arrangeBy(ids, ents, func(ent *model.BookingRule) int { return ent.ID }), nil
}
```

### Loader Param (`server/gql/loader_params.go`)

```go
type bookingRuleLoaderParam struct {
    FeedVersionID int
    Limit         *int
    Where         *model.BookingRuleFilter
}
```

### Loaders (`server/gql/loaders.go`)

```go
// In Loaders struct:
BookingRulesByFeedVersionIDs *dataloader.Loader[bookingRuleLoaderParam, []*model.BookingRule]
BookingRulesByIDs            *dataloader.Loader[int, *model.BookingRule]

// In NewLoaders():
BookingRulesByFeedVersionIDs: withWaitAndCapacityGroup(waitTime, batchSize,
    dbf.BookingRulesByFeedVersionIDs,
    func(p bookingRuleLoaderParam) (int, *model.BookingRuleFilter, *int) {
        return p.FeedVersionID, p.Where, p.Limit
    },
),
BookingRulesByIDs: withWaitAndCapacity(waitTime, batchSize, dbf.BookingRulesByIDs),
```

### Resolver (`server/gql/booking_rule_resolver.go`)

```go
package gql

import (
    "context"
    "github.com/interline-io/transitland-lib/server/model"
)

type bookingRuleResolver struct{ *Resolver }

func (r *bookingRuleResolver) FeedVersion(ctx context.Context, obj *model.BookingRule) (*model.FeedVersion, error) {
    return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

func (r *bookingRuleResolver) PriorNoticeService(ctx context.Context, obj *model.BookingRule) (*model.Calendar, error) {
    if !obj.PriorNoticeServiceID.Valid {
        return nil, nil  // Handle nullable field
    }
    return LoaderFor(ctx).CalendarsByIDs.Load(ctx, obj.PriorNoticeServiceID.Int())()
}
```

### Register Resolver (`server/gql/resolver.go`)

```go
func (r *Resolver) BookingRule() gqlout.BookingRuleResolver {
    return &bookingRuleResolver{r}
}
```

### Add Field to Parent Resolver (`server/gql/feed_version_resolver.go`)

When you add a field to a parent type (like `booking_rules` on `FeedVersion`), you need to add the resolver method:

```go
func (r *feedVersionResolver) BookingRules(ctx context.Context, obj *model.FeedVersion, limit *int, where *model.BookingRuleFilter) ([]*model.BookingRule, error) {
    return LoaderFor(ctx).BookingRulesByFeedVersionIDs.Load(ctx, bookingRuleLoaderParam{
        FeedVersionID: obj.ID,
        Limit:         resolverCheckLimit(limit),
        Where:         where,
    })()
}
```

## 5. Summary Checklist

When adding a new entity to GraphQL, ensure you complete all these steps:

- [ ] **Schema** (`schema/graphql/schema.graphqls`)
  - [ ] Add type definition with all fields
  - [ ] Add filter input type if needed
  - [ ] Add fields to parent types (e.g., `FeedVersion.booking_rules`)
  
- [ ] **Generate Code**
  - [ ] Run `(cd internal/generated/gqlout && go generate)`
  
- [ ] **Model** (`server/model/models.go`)
  - [ ] Add model struct with `FeedOnestopID`, `FeedVersionSHA1` fields
  - [ ] Embed the underlying GTFS type
  - [ ] (Optional) Use type alias if sharing data with another GraphQL type
  
- [ ] **Finder Interface** (`server/model/finders.go`)
  - [ ] Add `EntityByIDs` method
  - [ ] Add `EntityByParentIDs` method(s)
  
- [ ] **Database Finder** (`server/finders/dbfinder/<entity>.go`)
  - [ ] Implement finder methods
  - [ ] Create select function with explicit field enumeration (no `SELECT *`)
  - [ ] Use `arrangeBy()` and `arrangeGroup()` helpers
  
- [ ] **Loader Params** (`server/gql/loader_params.go`)
  - [ ] Add loader param struct for group loaders
  
- [ ] **Loaders** (`server/gql/loaders.go`)
  - [ ] Add loader fields to `Loaders` struct
  - [ ] Initialize loaders in `NewLoaders()`
  
- [ ] **Resolver** (`server/gql/<entity>_resolver.go`)
  - [ ] Create resolver struct
  - [ ] Implement all field resolvers (especially `FeedVersion`)
  - [ ] Handle nullable fields with `.Valid` checks
  
- [ ] **Register Resolver** (`server/gql/resolver.go`)
  - [ ] Add resolver method
  
- [ ] **Parent Resolver** (e.g., `server/gql/feed_version_resolver.go`)
  - [ ] Add resolver method for the new field on the parent type
  
- [ ] **Tests** (`server/gql/<entity>_resolver_test.go`)
  - [ ] Add test cases for basic queries
  - [ ] Test filters and relationships
  - [ ] Test accessing entity from parent (e.g., `feed_versions { booking_rules { ... } }`)
