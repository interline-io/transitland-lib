# GraphQL Server Tests

This directory contains the GraphQL resolver tests for the transitland-lib server.

## Test Harness

The test harness in `resolver_test.go` provides a declarative way to write GraphQL tests using the `testcase` struct.

### Basic Structure

```go
testcase{
    name:  "my test",
    query: `query { ... }`,
    vars:  hw{"var1": "value1"},
    // assertions go here
}
```

### Assertion Options

#### Full Response Match

Use `expect` to match the entire JSON response:

```go
expect: `{"data": {"field": "value"}}`,
```

#### Selector-Based Assertions

For checking specific parts of the response, use selector-based assertions with gjson path syntax.

**Primary method - `sel` array:**

Use `sel` to run multiple selector checks against the same query result:

```go
sel: []testcaseSelector{
    {
        selector:    "stops.#.stop_id",
        expectCount: 10,
    },
    {
        selector:     "stops.#.stop_name",
        expectUnique: []string{"Stop A", "Stop B"},
    },
},
```

**Convenience fields:**

For simple single-selector tests, use the convenience fields directly on `testcase`. These wrap to `sel` with a single `testcaseSelector`:

| Convenience Field | Wraps To | Description |
|-------------------|----------|-------------|
| `selector` | `sel[].selector` | gjson path to select values |
| `selectExpect` | `sel[].expect` | Exact match of all values (order independent) |
| `selectExpectUnique` | `sel[].expectUnique` | Match unique values only |
| `selectExpectContains` | `sel[].expectContains` | Spot-check that result contains these values |
| `selectExpectCount` | `sel[].expectCount` | Assert total count of selected values |
| `selectExpectUniqueCount` | `sel[].expectUniqueCount` | Assert count of unique values |

Example using convenience fields:

```go
testcase{
    name:                    "simple count test",
    query:                   `query { stops { stop_id } }`,
    selector:                "stops.#.stop_id",
    selectExpectCount:       100,
    selectExpectUniqueCount: 50,
    selectExpectContains:    []string{"STOP1", "STOP2"},
}
```

This is equivalent to:

```go
testcase{
    name:  "simple count test", 
    query: `query { stops { stop_id } }`,
    sel: []testcaseSelector{
        {
            selector:          "stops.#.stop_id",
            expectCount:       100,
            expectUniqueCount: 50,
            expectContains:    []string{"STOP1", "STOP2"},
        },
    },
}
```

#### Custom Test Function

For complex assertions, use the `f` callback:

```go
f: func(t *testing.T, jj string) {
    // custom assertions using gjson, etc.
},
```

### testcaseSelector Fields

| Field | Description |
|-------|-------------|
| `selector` | gjson path expression to select values from response |
| `expect` | Assert exact match of all values (order independent) |
| `expectUnique` | Assert these are the unique values (ignores duplicates) |
| `expectContains` | Assert result contains at least these values |
| `expectCount` | Assert total count of values returned by selector |
| `expectUniqueCount` | Assert count of unique values |

### gjson Path Examples

- `stops.#.stop_id` - All stop_id values from stops array
- `stops.0.stop_name` - First stop's name
- `routes.#(route_id=="123").route_long_name` - Filtered selection
- `feed_versions.0.feed.onestop_id` - Nested access

See [gjson syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) for more.
