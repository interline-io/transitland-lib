# GraphQL Resolver Test Guide

This guide documents the testing patterns used in the transitland-lib GraphQL resolver tests.

## Where Cases Go

Resolver cases live in two places:
- `server/gql/gqltest` holds the cases every `model.Finder` must answer the same way. `gqltest.Run` runs them against a finder; `TestDBFinder` runs them against the test database. Write a new case here unless it needs something only the database finder has.
- `server/gql` keeps the cases that need the database finder: census and places data, segments, stop observations, external references, route attributes, previous onestop ids, GBFS data, permissions, mutations and validation. Realtime tests that change the database's feed and agency rows also stay.

In `gqltest`, a group of cases is a builder that returns them, registered by name in `caseGroups` in `gqltest.go`:

```go
func levelResolverTestcases(t testing.TB, cfg model.Config) []testcase {
    testcases := []testcase{ ... }
    return testcases
}
```

A case that needs an optional finder capability, such as `Spatial`, goes in that capability's `optionalGroups` entry, and finders that lack the capability skip it. Time-dependent cases go in `clockGroups`. A check that spans several queries, such as one that reads ids and then queries by them, is a function in `testGroups`, which gets the client and config to query itself. A case group's `configure` field sets config options for that group alone. Each client `Run` builds reads the case's time and holds the default realtime fixtures. Realtime tests, which load their own fixtures for each case, are functions in `rtGroups`, and run only against finders that claim the `Realtime` capability. Find ids through `cfg.Finder`, never with SQL, so the case runs against every finder.

## Test Infrastructure

### Core Types

The case types and the functions that run them live in `server/gql/internal/gqlcase`. Both `server/gql` and `gqltest` alias the types as `testcase` and `testcaseSelector`:

```go
type hw = map[string]interface{} // Helper type for variables

type TestcaseSelector struct {
    Selector          string
    Expect            []string
    ExpectUnique      []string
    ExpectContains    []string
    ExpectCount       int
    ExpectUniqueCount int
}

type Testcase struct {
    Name                    string                   // Test name (used in t.Run)
    Query                   string                   // GraphQL query string
    Vars                    map[string]any           // Query variables
    Expect                  string                   // Exact JSON match
    User                    string                   // User to run query as (optional)
    ExpectError             bool                     // Expect query to error
    ExpectErrorContains     string                   // Expect an error containing this text
    Sel                     []TestcaseSelector       // Multiple selectors
    F                       func(*testing.T, string) // Custom validation function
    Selector                string                   // gjson path for array extraction
    SelectExpect            []string                 // Expected values (order-independent)
    SelectExpectUnique      []string                 // Expected unique values
    SelectExpectContains    []string                 // Values the result must contain
    SelectExpectCount       int                      // Expected count only
    SelectExpectUniqueCount int                      // Expected count of unique values
}
```

`gqltest` also defines `testcaseWithClock`, which pairs a case with the clock time `whenUtc`:

```go
type testcaseWithClock struct {
    testcase
    whenUtc string
}
```

### Test Execution Helpers

```go
// Create test client (server/gql)
c, cfg := newTestClient(t)

// Create test client with options (e.g. time travel)
c, cfg := newTestClientWithOpts(t, testconfig.Options{WhenUtc: "..."})

// Run multiple test cases
gqlcase.QueryTestcases(t, c, testcases)

// Run single test case
gqlcase.QueryTestcase(t, c, tc)

// Benchmark test cases
gqlcase.BenchmarkTestcases(b, c, testcases)
```

`gqltest` has its own `newTestClient`, which serves the config a finder builds at the case's time, with the realtime fixtures loaded.

### Test File Naming Convention

Test files should follow this naming convention:
- `<entity>_resolver_test.go` - Main resolver tests
- `<entity>_resolver_rt_test.go` - Real-time data tests

Examples:
- `agency_resolver_test.go`
- `stop_resolver_test.go`
- `agency_resolver_rt_test.go`

In `gqltest`, each entity's cases go in `<entity>_cases.go`, such as `booking_rule_cases.go`, and the cases that need the `Spatial` capability go in `spatial_cases.go`.

---

## Testing Patterns

### Pattern 1: Exact JSON Match (`Expect`)

Use for single-result queries where the exact output is known and stable.

```go
{
    Name:   "basic fields",
    Query:  `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {agency_id agency_name}}`,
    Vars:   hw{"agency_id": "caltrain-ca-us"},
    Expect: `{"agencies":[{"agency_id":"caltrain-ca-us","agency_name":"Caltrain"}]}`,
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

### Pattern 2: Selector with Expected Values (`Selector` + `SelectExpect`)

Use for queries returning arrays where order may vary.

```go
{
    Name:         "routes",
    Query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {routes { route_id }}}`,
    Vars:         hw{"agency_id": "caltrain-ca-us"},
    Selector:     "agencies.0.routes.#.route_id",
    SelectExpect: []string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
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

### Pattern 3: Selector with Count (`SelectExpectCount`)

Use when you only care about the number of results.

```go
{
    Name:              "booking_rules",
    Query:             `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { booking_rules { booking_rule_id }}}`,
    Vars:              hw{"sha1": "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"},
    Selector:          "feed_versions.0.booking_rules.#.booking_rule_id",
    SelectExpectCount: 100,
}
```

**When to use:**
- Large result sets where listing all values is impractical
- Counting children without caring about specific values
- Verifying limits work correctly

---

### Pattern 4: Custom Validation Function (`F`)

Use for complex validations that can't be expressed with simple matchers.

```go
{
    Name:  "file details",
    Query: `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { files { name rows sha1 }}}`,
    Vars:  hw{"sha1": "d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
    F: func(t *testing.T, jj string) {
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

**AVOID** using `F:` tests unless the check cannot be performed without custom logic.

---

### Pattern 5: Unique Values (`SelectExpectUnique`)

Use when testing distinct values from a result set.

```go
{
    Name:               "headways dow_category",
    Query:              `query($route_id: String!) { routes(where:{route_id:$route_id}) { headways { dow_category }}}`,
    Vars:               hw{"route_id": "03"},
    Selector:           "routes.0.headways.#.dow_category",
    SelectExpectUnique: []string{"1", "6", "7"},
}
```

### Pattern 6: Multiple Selectors (`Sel`)

Use when you need to validate multiple parts of the response independently.

```go
{
    Name:  "multiple checks",
    Query: `query { agencies { agency_id agency_name }}`,
    Sel: []testcaseSelector{
        {
            Selector: "agencies.#.agency_id",
            Expect:   []string{"caltrain-ca-us", "BART"},
        },
        {
            Selector:    "agencies.#.agency_name",
            ExpectCount: 2,
        },
    },
}
```

### Pattern 7: Time-Dependent Tests (`testcaseWithClock`)

Use when the query result depends on the current time (e.g., `stop_times` with `next` filter).

```go
func stopResolverStopTimesNextTestcases() []testcaseWithClock {
    testcases := []testcaseWithClock{
        {
            whenUtc: "2018-05-30T22:00:00Z",
            testcase: testcase{
                Name:     "where next 3600",
                Query:    `query{ stops(where:{stop_id:"MCAR_S"}) { stop_times(where:{next:3600}) {arrival_time}}}`,
                Selector: "stops.0.stop_times.#.arrival_time",
                SelectExpect: []string{"15:01:00", "15:09:00"},
            },
        },
    }
    return testcases
}
```

**Note:** Register the builder in `clockGroups`, and `Run` sets each case's clock. `testcaseWithClock` exists only in `gqltest`. A time-dependent test in `server/gql` sets its clock with `newTestClientWithOpts(t, testconfig.Options{WhenUtc: "..."})`.

---

### Pattern 8: Error Testing (`ExpectError`)

Use when testing error conditions like invalid input or permission failures.

```go
{
    Name:        "where bbox too large",
    Query:       `query($bbox:BoundingBox) {agencies(where:{bbox:$bbox}) {agency_id}}`,
    Vars:        hw{"bbox": hw{"min_lon": -137.88, "min_lat": 30.07, "max_lon": -109.00, "max_lat": 45.02}},
    ExpectError: true,
    F: func(t *testing.T, jj string) {
        // Optional: additional validation
    },
}
```

To check the message as well, set `ExpectErrorContains` to text the error message must contain. It implies `ExpectError`.

**When to use:**
- Testing input validation (e.g., bbox too large)
- Testing limit/offset errors
- Testing permission denials

---

### Pattern 9: Real-Time Data Testing (`rtTestCase`)

Use for testing real-time GTFS-RT data (alerts, trip updates, vehicle positions). These tests live in `gqltest`, as functions registered in `rtGroups`. Each case runs against a client whose realtime finder holds only the case's `rtfiles`, at the case's `whenUtc`.

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
func agencyRTAlertsTest(t *testing.T, newConfig NewConfig) {
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
        testRt(t, newConfig, tc)
    }
}
```

**RT File Types:**
- `realtime_alerts` - GTFS-RT Service Alerts
- `realtime_trip_updates` - Trip Updates
- `realtime_vehicle_positions` - Vehicle Positions

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
            Name:         "user ian sees all agencies",
            Query:        `query { agencies {agency_id}}`,
            User:         "ian",  // User context
            Selector:     "agencies.#.agency_id",
            SelectExpect: []string{"caltrain-ca-us", "BART", "", "573"},
        },
        {
            Name:         "public user sees fewer agencies",
            Query:        `query { agencies {agency_id}}`,
            User:         "public",
            Selector:     "agencies.#.agency_id",
            SelectExpect: []string{"caltrain-ca-us", "BART", ""},
        },
    }

    // Note: Must create client with user middleware for each test
    for _, tc := range testcases {
        t.Run(tc.Name, func(t *testing.T) {
            c := client.New(usercheck.UserDefaultMiddleware(tc.user)(srv))
            gqlcase.QueryTestcase(t, c, tc)
        })
    }
}
```

**Note:** `gqlcase.QueryTestcase` does NOT use the case's `User` field. You must manually create a client with the user middleware.

---

## Common Test Categories

### 1. Basic Entity Tests

Every resolver should have:

```go
// List all entities
{
    Name:         "basic",
    Query:        `query { agencies { agency_id }}`,
    Selector:     "agencies.#.agency_id",
    SelectExpect: []string{...},
}

// All fields for single entity
{
    Name:   "basic fields",
    Query:  `query($id:String!) { agencies(where:{agency_id:$id}) { all fields here }}`,
    Vars:   hw{"agency_id": "caltrain-ca-us"},
    Expect: `{...exact json...}`,
}
```

### 2. Filter Tests (`where` clauses)

```go
{
    Name:         "where onestop_id",
    Query:        `query { routes(where:{onestop_id:"r-9q9j-bullet"}) { route_id }}`,
    Selector:     "routes.#.route_id",
    SelectExpect: []string{"Bu-130"},
}

{
    Name:         "where feed_version_sha1",
    Query:        `query { routes(where:{feed_version_sha1:"..."}) { route_id }}`,
    Selector:     "routes.#.route_id",
    SelectExpect: []string{...},
}
```

### 3. Relationship Navigation Tests

```go
// Parent to children
{
    Name:         "agency routes",
    Query:        `query($id:String!) { agencies(where:{agency_id:$id}) { routes { route_id }}}`,
    Vars:         hw{"agency_id": "caltrain-ca-us"},
    Selector:     "agencies.0.routes.#.route_id",
    SelectExpect: []string{...},
}

// Child to parent
{
    Name:   "route feed_version",
    Query:  `query($id:String!) { routes(where:{route_id:$id}) { feed_version { sha1 }}}`,
    Vars:   hw{"route_id": "03"},
    Expect: `{"routes":[{"feed_version":{"sha1":"..."}}]}`,
}

// Circular navigation
{
    Name: "stop_time navigates back to stop",
    Query: `query { stops(where:{stop_id:"..."}) { stop_times(limit:1) { stop { stop_id }}}}`,
    F: func(t *testing.T, jj string) {
        stStop := gjson.Get(jj, "stops.0.stop_times.0.stop.stop_id").String()
        assert.Equal(t, "...", stStop)
    },
}
```

### 4. Geometry Tests

```go
{
    Name:         "geometry type",
    Query:        `query($id:String!) { agencies(where:{agency_id:$id}) { geometry }}`,
    Vars:         hw{"agency_id": "caltrain-ca-us"},
    Selector:     "agencies.0.geometry.type",
    SelectExpect: []string{"Polygon"},
}
```

### 5. Nested Filter Tests

```go
{
    Name:         "routes filtered by type",
    Query:        `query { agencies(where:{agency_id:"..."}) { routes(where:{route_type:2}) { route_id }}}`,
    Selector:     "agencies.0.routes.#.route_id",
    SelectExpect: []string{...},
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
            Name:         "california",
            Query:        q,
            Vars:         hw{"feed_version_sha1": "e535eb2b...", "stop_id": "FTVL"},
            Selector:     "stops.#.place.adm1_name",
            SelectExpect: []string{"California"},
        },
    }
    gqlcase.QueryTestcases(t, c, tcs)
}
```

### 7. Benchmarking Tests

For performance testing resolvers:

```go
func BenchmarkStopResolver(b *testing.B) {
    c, cfg := newTestClient(b)
    gqlcase.BenchmarkTestcases(b, c, stopResolverTestcases(b, cfg))
}
```

**Note:** Case builders take `testing.TB`, so a benchmark can build the same cases a test runs. `server/gql`'s `BenchmarkStopResolver` covers the stop cases that need the database finder, and `gqltest`'s `BenchmarkDBFinder_StopResolver` covers the rest.

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
Vars: hw{"sha1": ctranFlexSha1, "location_id": roseVillageLocationID}

// Avoid
Query: `query { feed_versions(where:{sha1:"e8bc76c3..."}) {...}}`
```

### 2. Define Constants for Reused Values

```go
func locationResolverStopTimesTestcases(t testing.TB, cfg model.Config) []testcase {
    ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
    roseVillageLocationID := "location_id__c7400cc8-959c-42c8-991f-8f601ec9ea59"
    testcases := []testcase{...}
    return testcases
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

Use separate case groups for complex relationships:

```go
func agencyResolverTestcases(t testing.TB, cfg model.Config) []testcase { ... }        // Basic fields, filters
func agencyResolverStopsTestcases(t testing.TB, cfg model.Config) []testcase { ... }   // Agency->Stop relationship
func agencyResolverLicenseTestcases(t testing.TB, cfg model.Config) []testcase { ... } // License filter tests
```

**Pattern for complex relationships:**
- `EntityResolver` - Basic entity tests (fields, simple filters)
- `EntityResolver_ChildEntity` - Complex child relationship tests

Examples of groups in `gqltest`:
- `StopResolver` + `StopResolverCursor` + `StopResolverLicense`
- `LocationResolver` + `LocationResolver_StopTimes`
- `LocationGroupResolver` + `LocationGroupResolver_StopTimes`

### 4. Test Both Positive and Negative Cases

```go
{
    Name:         "where state found",
    SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
}
{
    Name:         "where state not found",
    SelectExpect: []string{},  // Empty result
}
```

### 5. Use `Selector` for IDs, `Expect` for Details

```go
// List of IDs - use selector
Selector:     "routes.#.route_id"
SelectExpect: []string{"Bu-130", "Li-130"}

// Full entity details - use expect
Expect: `{"routes":[{"route_id":"Bu-130","route_name":"Bullet",...}]}`
```

---

## Example: Complete Resolver Test

```go
func locationGroupResolverTestcases(t testing.TB, cfg model.Config) []testcase {
    ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
    testcases := []testcase{
        // 1. List all entities with IDs
        {
            Name: "location_groups count and ids",
            Query: `query($sha1: String!) {
                feed_versions(where: {sha1: $sha1}) {
                    location_groups { location_group_id }
                }
            }`,
            Vars:         hw{"sha1": ctranFlexSha1},
            Selector:     "feed_versions.0.location_groups.#.location_group_id",
            SelectExpect: []string{"lg1", "lg2", "lg3"},
        },
        // 2. Single entity with all fields
        {
            Name: "location_group fields",
            Query: `query($sha1: String!, $lg_id: String!) {
                feed_versions(where: {sha1: $sha1}) {
                    location_groups(where: {location_group_id: $lg_id}) {
                        location_group_id
                        location_group_name
                        feed_version { sha1 }
                    }
                }
            }`,
            Vars:   hw{"sha1": ctranFlexSha1, "lg_id": "lg1"},
            Expect: `{"feed_versions":[{"location_groups":[{...}]}]}`,
        },
        // 3. Relationship navigation
        {
            Name: "location_group stops",
            Query: `query($sha1: String!, $lg_id: String!) {
                feed_versions(where: {sha1: $sha1}) {
                    location_groups(where: {location_group_id: $lg_id}) {
                        stops { stop { stop_id }}
                    }
                }
            }`,
            Vars:         hw{"sha1": ctranFlexSha1, "lg_id": "lg1"},
            Selector:     "feed_versions.0.location_groups.0.stops.#.stop.stop_id",
            SelectExpect: []string{"stop1", "stop2"},
        },
    }
    return testcases
}
```

Then register it in `caseGroups`: `{name: "LocationGroupResolver", cases: locationGroupResolverTestcases},`.

---

## Test Checklist for New Entity Types

When adding tests for a new GTFS entity (e.g., BookingRule, Location, LocationGroup):

### Required Tests

- [ ] **List all entities** - Query all entities with IDs (`SelectExpect` or `SelectExpectCount`)
- [ ] **Single entity fields** - Query specific entity with all fields (`Expect` with exact JSON)
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
func bookingRuleResolverTestcases(t testing.TB, cfg model.Config) []testcase {
    testcases := []testcase{
        // List all with count
        {
            Name: "booking rules - returns multiple booking rules",
            Query: `query { feed_versions(where: {sha1: "..."}) { booking_rules { booking_rule_id }}}`,
            Selector: "feed_versions.0.booking_rules.#.booking_rule_id",
            SelectExpectCount: 100,
        },
        // Single entity with all fields + feed_version navigation
        {
            Name: "filters booking rules by booking_rule_id",
            Query: `query($brid: String) {
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
            Vars: hw{"brid": "booking_rule_id__..."},
            Expect: `{...exact JSON...}`,
        },
    }
    return testcases
}
```

---

## Running Tests

```bash
# Run all resolver tests
go test ./server/gql/... -v

# Run specific resolver test
go test ./server/gql/... -run TestAgencyResolver -v

# Run one gqltest group against the database finder
go test ./server/gql/gqltest/ -run 'TestDBFinder/BookingRuleResolver' -v

# Run tests with race detection
go test ./server/gql/... -race -v

# Run benchmarks
go test ./server/gql/... -bench=. -benchmem

# Run specific benchmark
go test ./server/gql/gqltest/ -run '^$' -bench=BenchmarkDBFinder_StopResolver -benchmem
```
