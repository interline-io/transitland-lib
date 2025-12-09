# GraphQL Resolver Test Guide

This guide documents the testing patterns used in the transitland-lib GraphQL resolver tests.

## Test Infrastructure

### Core Types

```go
type hw = map[string]interface{}  // Helper type for variables

type testcaseSelector struct {
    selector     string
    expect       []string
    expectUnique []string
    expectCount  int
}

type testcase struct {
    name               string                    // Test name (used in t.Run)
    query              string                    // GraphQL query string
    vars               hw                        // Query variables
    expect             string                    // Exact JSON match
    user               string                    // User to run query as (optional)
    expectError        bool                      // Expect query to error
    selector           string                    // gjson path for array extraction
    selectExpect       []string                  // Expected values (order-independent)
    selectExpectUnique []string                  // Expected unique values
    selectExpectCount  int                       // Expected count only
    sel                []testcaseSelector        // Multiple selectors
    f                  func(*testing.T, string)  // Custom validation function
}

type testcaseWithClock struct {
    testcase
    whenUtc string
}
```

### Test Execution Helpers

```go
// Create test client
c, cfg := newTestClient(t)

// Create test client with options (e.g. time travel)
c, cfg := newTestClientWithOpts(t, testconfig.Options{WhenUtc: "..."})

// Run multiple test cases
queryTestcases(t, c, testcases)

// Run single test case
queryTestcase(t, c, tc)

// Run benchmarks
benchmarkTestcases(b, c, testcases)
```

### Test File Naming Convention

Test files should follow this naming convention:
- `<entity>_resolver_test.go` - Main resolver tests
- `<entity>_resolver_rt_test.go` - Real-time data tests

Examples:
- `booking_rule_resolver_test.go`
- `location_resolver_test.go`
- `agency_resolver_rt_test.go`

---

## Testing Patterns

### Pattern 1: Exact JSON Match (`expect`)

Use for single-result queries where the exact output is known and stable.

```go
{
    name:   "basic fields",
    query:  `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {agency_id agency_name}}`,
    vars:   hw{"agency_id": "caltrain-ca-us"},
    expect: `{"agencies":[{"agency_id":"caltrain-ca-us","agency_name":"Caltrain"}]}`,
}
```

**When to use:**
- Single entity queries with stable, known values
- Testing all fields of an entity
- Navigation relationships (e.g., `feed_version { sha1 }`)

**Tips:**
- JSON keys are alphabetically sorted in output
- Use variables to avoid hardcoding values in query string
- Good for testing null values explicitly

---

### Pattern 2: Selector with Expected Values (`selector` + `selectExpect`)

Use for queries returning arrays where order may vary.

```go
{
    name:         "routes",
    query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {routes { route_id }}}`,
    vars:         hw{"agency_id": "caltrain-ca-us"},
    selector:     "agencies.0.routes.#.route_id",
    selectExpect: []string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
}
```

**Selector syntax (gjson paths):**
- `entities.#.field` - Extract field from all array elements
- `entities.0.field` - First element's field
- `entities.0.children.#.field` - Nested array extraction

**When to use:**
- List queries with multiple results
- Relationships returning arrays (routes, stops, trips)
- When order is not guaranteed

---

### Pattern 3: Selector with Count (`selectExpectCount`)

Use when you only care about the number of results.

```go
{
    name:              "booking_rules",
    query:             `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { booking_rules { booking_rule_id }}}`,
    vars:              hw{"sha1": "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"},
    selector:          "feed_versions.0.booking_rules.#.booking_rule_id",
    selectExpectCount: 100,
}
```

**When to use:**
- Large result sets where listing all values is impractical
- Counting children without caring about specific values
- Verifying limits work correctly

---

### Pattern 4: Custom Validation Function (`f`)

Use for complex validations that can't be expressed with simple matchers.

```go
{
    name:  "file details",
    query: `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { files { name rows sha1 }}}`,
    vars:  hw{"sha1": "d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
    f: func(t *testing.T, jj string) {
        for _, file := range gjson.Get(jj, "feed_versions.0.files").Array() {
            if file.Get("name").String() == "trips.txt" {
                assert.Equal(t, int64(185), file.Get("rows").Int())
                assert.Equal(t, "1ad77955e41e33cb1fceb694df27ced80e0ecbd3", file.Get("sha1").String())
            }
        }
    },
}
```

**When to use:**
- Conditional logic based on values
- Multiple related assertions
- Checking existence (`.Exists()`)
- Numeric comparisons on counts
- Iterating over results with validation

---

### Pattern 5: Unique Values (`selectExpectUnique`)

Use when testing distinct values from a result set.

```go
{
    name:               "headways dow_category",
    query:              `query($route_id: String!) { routes(where:{route_id:$route_id}) { headways { dow_category }}}`,
    vars:               hw{"route_id": "03"},
    selector:           "routes.0.headways.#.dow_category",
    selectExpectUnique: []string{"1", "6", "7"},
}
```

### Pattern 6: Multiple Selectors (`sel`)

Use when you need to validate multiple parts of the response independently.

```go
{
    name:  "multiple checks",
    query: `query { agencies { agency_id agency_name }}`,
    sel: []testcaseSelector{
        {
            selector: "agencies.#.agency_id",
            expect:   []string{"caltrain-ca-us", "BART"},
        },
        {
            selector:    "agencies.#.agency_name",
            expectCount: 2,
        },
    },
}
```

### Pattern 7: Time-Dependent Tests (`testcaseWithClock`)

Use when the query result depends on the current time (e.g., `stop_times` with `next` filter).

```go
func TestStopResolver_StopTimes_Next(t *testing.T) {
    testcases := []testcaseWithClock{
        {
            whenUtc: "2018-05-30T22:00:00Z",
            testcase: testcase{
                name:     "where next 3600",
                query:    `query{ stops(where:{stop_id:"MCAR_S"}) { stop_times(where:{next:3600}) {arrival_time}}}`,
                selector: "stops.0.stop_times.#.arrival_time",
                selectExpect: []string{"15:01:00", "15:09:00"},
            },
        },
    }

    for _, tc := range testcases {
        t.Run(tc.name, func(t *testing.T) {
            c, _ := newTestClientWithOpts(t, testconfig.Options{
                RTJsons: testconfig.DefaultRTJson(),
                WhenUtc: tc.whenUtc,
            })
            queryTestcase(t, c, tc.testcase)
        })
    }
}
```

**Note:** These tests require manual iteration and `newTestClientWithOpts` to set the clock.

---

### Pattern 8: Error Testing (`expectError`)

Use when testing error conditions like invalid input or permission failures.

```go
{
    name:        "where bbox too large",
    query:       `query($bbox:BoundingBox) {agencies(where:{bbox:$bbox}) {agency_id}}`,
    vars:        hw{"bbox": hw{"min_lon": -137.88, "min_lat": 30.07, "max_lon": -109.00, "max_lat": 45.02}},
    expectError: true,
    f: func(t *testing.T, jj string) {
        // Optional: additional validation
    },
}
```

**When to use:**
- Testing input validation (e.g., bbox too large)
- Testing limit/offset errors
- Testing permission denials

---

### Pattern 9: Real-Time Data Testing (`rtTestCase`)

Use for testing real-time GTFS-RT data (alerts, trip updates, vehicle positions).

```go
type rtTestCase struct {
    name    string
    query   string
    vars    map[string]interface{}
    rtfiles []testconfig.RTJsonFile
    cb      func(t *testing.T, jj string)
    whenUtc string
}

// Example RT test
func TestAgencyRT_Alerts(t *testing.T) {
    tcs := []rtTestCase{
        {
            name:  "stop alerts",
            query: rtTestStopQuery,
            vars:  rtTestStopQueryVars(),
            rtfiles: []testconfig.RTJsonFile{
                {Feed: "BA", Ftype: "realtime_alerts", Fname: "BA-alerts.json"},
            },
            cb: func(t *testing.T, jj string) {
                alerts := gjson.Get(jj, "stops.0.stop_times.0.trip.route.agency.alerts").Array()
                assert.Equal(t, 2, len(alerts))
            },
        },
    }
    for _, tc := range tcs {
        testRt(t, tc)
    }
}
```

**RT File Types:**
- `realtime_alerts` - GTFS-RT Service Alerts
- `realtime_trip_updates` - Trip Updates
- `vehicle_positions` - Vehicle Positions

---

### Pattern 10: Authorization/Permission Testing

Use when testing user-specific access control.

```go
func TestAgencyResolver_FGA(t *testing.T) {
    // Create config with FGA model tuples for authorization testing
    cfg := testconfig.Config(t, testconfig.Options{
        FGAModelTuples: fgaTestTuples,
    })

    srv, _ := NewServer()
    srv = model.AddConfigAndPerms(cfg, srv)

    testcases := []testcase{
        {
            name:         "user ian sees all agencies",
            query:        `query { agencies {agency_id}}`,
            user:         "ian",  // User context
            selector:     "agencies.#.agency_id",
            selectExpect: []string{"caltrain-ca-us", "BART", "", "573"},
        },
        {
            name:         "public user sees fewer agencies",
            query:        `query { agencies {agency_id}}`,
            user:         "public",
            selector:     "agencies.#.agency_id",
            selectExpect: []string{"caltrain-ca-us", "BART", ""},
        },
    }

    // Note: Must create client with user middleware for each test
    for _, tc := range testcases {
        t.Run(tc.name, func(t *testing.T) {
            c := client.New(usercheck.UserDefaultMiddleware(tc.user)(srv))
            queryTestcase(t, c, tc)
        })
    }
}
```

**Note:** The `user` field in `testcase` is NOT automatically used by `queryTestcase`. You must manually create a client with the user middleware.

---

## Common Test Categories

### 1. Basic Entity Tests

Every resolver should have:

```go
// List all entities
{
    name:         "basic",
    query:        `query { agencies { agency_id }}`,
    selector:     "agencies.#.agency_id",
    selectExpect: []string{...},
}

// All fields for single entity
{
    name:   "basic fields",
    query:  `query($id:String!) { agencies(where:{agency_id:$id}) { all fields here }}`,
    vars:   hw{"agency_id": "caltrain-ca-us"},
    expect: `{...exact json...}`,
}
```

### 2. Filter Tests (`where` clauses)

```go
{
    name:         "where onestop_id",
    query:        `query { routes(where:{onestop_id:"r-9q9j-bullet"}) { route_id }}`,
    selector:     "routes.#.route_id",
    selectExpect: []string{"Bu-130"},
}

{
    name:         "where feed_version_sha1",
    query:        `query { routes(where:{feed_version_sha1:"..."}) { route_id }}`,
    selector:     "routes.#.route_id",
    selectExpect: []string{...},
}
```

### 3. Relationship Navigation Tests

```go
// Parent to children
{
    name:         "agency routes",
    query:        `query($id:String!) { agencies(where:{agency_id:$id}) { routes { route_id }}}`,
    vars:         hw{"agency_id": "caltrain-ca-us"},
    selector:     "agencies.0.routes.#.route_id",
    selectExpect: []string{...},
}

// Child to parent
{
    name:   "route feed_version",
    query:  `query($id:String!) { routes(where:{route_id:$id}) { feed_version { sha1 }}}`,
    vars:   hw{"route_id": "03"},
    expect: `{"routes":[{"feed_version":{"sha1":"..."}}]}`,
}

// Circular navigation
{
    name: "stop_time navigates back to stop",
    query: `query { stops(where:{stop_id:"..."}) { stop_times(limit:1) { stop { stop_id }}}}`,
    f: func(t *testing.T, jj string) {
        stStop := gjson.Get(jj, "stops.0.stop_times.0.stop.stop_id").String()
        assert.Equal(t, "...", stStop)
    },
}
```

### 4. Geometry Tests

```go
{
    name:         "geometry type",
    query:        `query($id:String!) { agencies(where:{agency_id:$id}) { geometry }}`,
    vars:         hw{"agency_id": "caltrain-ca-us"},
    selector:     "agencies.0.geometry.type",
    selectExpect: []string{"Polygon"},
}
```

### 5. Nested Filter Tests

```go
{
    name:         "routes filtered by type",
    query:        `query { agencies(where:{agency_id:"..."}) { routes(where:{route_type:2}) { route_id }}}`,
    selector:     "agencies.0.routes.#.route_id",
    selectExpect: []string{...},
}
```

### 6. Admin Cache Tests

For testing place/admin lookups that require the admin cache to be loaded:

```go
func TestStopResolver_AdminCache(t *testing.T) {
    type canLoadAdmins interface {
        LoadAdmins(context.Context) error
    }
    c, cfg := newTestClient(t)

    // Load the admin cache
    if v, ok := cfg.Finder.(canLoadAdmins); !ok {
        t.Fatal("finder cant load admins")
    } else {
        if err := v.LoadAdmins(context.Background()); err != nil {
            t.Fatal(err)
        }
    }

    q := `query($feed_version_sha1:String!, $stop_id:String!) {
        stops(where:{stop_id:$stop_id, feed_version_sha1:$feed_version_sha1}) {
            place { adm0_name adm1_name adm0_iso adm1_iso }
        }
    }`

    tcs := []testcase{
        {
            name:         "california",
            query:        q,
            vars:         hw{"feed_version_sha1": "e535eb2b...", "stop_id": "FTVL"},
            selector:     "stops.#.place.adm1_name",
            selectExpect: []string{"California"},
        },
    }
    queryTestcases(t, c, tcs)
}
```

### 7. Benchmarking Tests

For performance testing resolvers:

```go
func BenchmarkStopResolver(b *testing.B) {
    c, cfg := newTestClient(b)
    benchmarkTestcases(b, c, stopResolverTestcases(b, cfg))
}
```

**Note:** Benchmark functions use `testing.B` instead of `testing.T` and use `benchmarkTestcases` helper.

---

## Test Data References

### Feed Version SHA1s

| Feed | SHA1 |
|------|------|
| BART | `e535eb2b3b9ac3ef15d82c56575e914575e732e0` |
| Caltrain | `d2813c293bcfd7a97dde599527ae6c62c98e66c6` |
| C-TRAN Flex | `e8bc76c3c8602cad745f41a49ed5c5627ad6904c` |
| HART | `c969427f56d3a645195dd8365cde6d7feae7e99b` |

### Feed Onestop IDs

| Feed | Onestop ID |
|------|------------|
| BART | `BA` |
| Caltrain | `CT` |

---

## Best Practices

### 1. Use Variables, Not Hardcoded Values

```go
// Good
vars: hw{"sha1": ctranFlexSha1, "location_id": roseVillageLocationID}

// Avoid
query: `query { feed_versions(where:{sha1:"e8bc76c3..."}) {...}}`
```

### 2. Define Constants for Reused Values

```go
func TestLocationResolver_StopTimes(t *testing.T) {
    ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
    roseVillageLocationID := "location_id__c7400cc8-959c-42c8-991f-8f601ec9ea59"
    testcases := []testcase{...}
}
```

Or define test constants for the file:

```go
var (
    bartSha1     = "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
    caltrainSha1 = "d2813c293bcfd7a97dde599527ae6c62c98e66c6"
    ctranFlexSha1 = "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
)
```

### 3. Group Related Tests

Use separate top-level test functions for complex relationships:

```go
func TestAgencyResolver(t *testing.T) { ... }           // Basic fields, filters
func TestAgencyResolver_Routes(t *testing.T) { ... }    // Agency->Route relationship
func TestAgencyResolver_Places(t *testing.T) { ... }    // Agency->Place relationship
func TestAgencyResolver_License(t *testing.T) { ... }   // License filter tests
```

**Pattern for complex relationships:**
- `TestEntityResolver` - Basic entity tests (fields, simple filters)
- `TestEntityResolver_ChildEntity` - Complex child relationship tests

Examples from the codebase:
- `TestStopResolver` + `TestStopResolver_Cursor` + `TestStopResolver_License`
- `TestLocationResolver` + `TestLocationResolver_StopTimes`
- `TestLocationGroupResolver` + `TestLocationGroupResolver_StopTimes`

### 4. Test Both Positive and Negative Cases

```go
{
    name:         "where state found",
    selectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
}
{
    name:         "where state not found",
    selectExpect: []string{},  // Empty result
}
```

### 5. Use `selector` for IDs, `expect` for Details

```go
// List of IDs - use selector
selector:     "routes.#.route_id"
selectExpect: []string{"Bu-130", "Li-130"}

// Full entity details - use expect
expect: `{"routes":[{"route_id":"Bu-130","route_name":"Bullet",...}]}`
```

---

## Example: Complete Resolver Test

```go
func TestLocationGroupResolver(t *testing.T) {
    ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
    testcases := []testcase{
        // 1. List all entities with IDs
        {
            name: "location_groups count and ids",
            query: `query($sha1: String!) {
                feed_versions(where: {sha1: $sha1}) {
                    location_groups { location_group_id }
                }
            }`,
            vars:         hw{"sha1": ctranFlexSha1},
            selector:     "feed_versions.0.location_groups.#.location_group_id",
            selectExpect: []string{"lg1", "lg2", "lg3"},
        },
        // 2. Single entity with all fields
        {
            name: "location_group fields",
            query: `query($sha1: String!, $lg_id: String!) {
                feed_versions(where: {sha1: $sha1}) {
                    location_groups(where: {location_group_id: $lg_id}) {
                        location_group_id
                        location_group_name
                        feed_version { sha1 }
                    }
                }
            }`,
            vars:   hw{"sha1": ctranFlexSha1, "lg_id": "lg1"},
            expect: `{"feed_versions":[{"location_groups":[{...}]}]}`,
        },
        // 3. Relationship navigation
        {
            name: "location_group stops",
            query: `query($sha1: String!, $lg_id: String!) {
                feed_versions(where: {sha1: $sha1}) {
                    location_groups(where: {location_group_id: $lg_id}) {
                        stops { stop { stop_id }}
                    }
                }
            }`,
            vars:         hw{"sha1": ctranFlexSha1, "lg_id": "lg1"},
            selector:     "feed_versions.0.location_groups.0.stops.#.stop.stop_id",
            selectExpect: []string{"stop1", "stop2"},
        },
    }
    c, _ := newTestClient(t)
    queryTestcases(t, c, testcases)
}
```

---

## Test Checklist for New Entity Types

When adding tests for a new GTFS entity (e.g., BookingRule, Location, LocationGroup):

### Required Tests

- [ ] **List all entities** - Query all entities with IDs (`selectExpect` or `selectExpectCount`)
- [ ] **Single entity fields** - Query specific entity with all fields (`expect` with exact JSON)
- [ ] **FeedVersion relationship** - Test `feed_version { sha1 }` navigation
- [ ] **Filter by primary ID** - Test `where: {entity_id: $id}` filter
- [ ] **Filter by feed_version_sha1** - Test `where: {feed_version_sha1: $sha1}` filter

### Common Additional Tests

- [ ] **Child relationships** - If entity has children (e.g., Location→StopTimes)
- [ ] **Parent relationships** - If entity has parents (e.g., StopTime→Trip)
- [ ] **Nullable field handling** - Test fields that can be null return correctly
- [ ] **Circular navigation** - Entity→Child→Entity returns same ID

### Example: BookingRule Test Coverage

```go
func TestBookingRuleResolver(t *testing.T) {
    testcases := []testcase{
        // List all with count
        {
            name: "booking rules - returns multiple booking rules",
            query: `query { feed_versions(where: {sha1: "..."}) { booking_rules { booking_rule_id }}}`,
            selector: "feed_versions.0.booking_rules.#.booking_rule_id",
            selectExpectCount: 100,
        },
        // Single entity with all fields + feed_version navigation
        {
            name: "filters booking rules by booking_rule_id",
            query: `query($brid: String) {
                feed_versions(where: {sha1: "..."}) {
                    booking_rules(where:{booking_rule_id: $brid}) {
                        booking_rule_id
                        booking_type
                        prior_notice_duration_min
                        message
                        phone_number
                        info_url
                        feed_version { sha1 }
                    }
                }
            }`,
            vars: hw{"brid": "booking_rule_id__..."},
            expect: `{...exact JSON...}`,
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

---

## Running Tests

```bash
# Run all resolver tests
go test ./server/gql/... -v

# Run specific resolver test
go test ./server/gql/... -run TestBookingRuleResolver -v

# Run tests with race detection
go test ./server/gql/... -race -v

# Run benchmarks
go test ./server/gql/... -bench=. -benchmem

# Run specific benchmark
go test ./server/gql/... -bench=BenchmarkStopResolver -benchmem
```
