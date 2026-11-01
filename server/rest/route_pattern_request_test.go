package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/gql"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestRoutePatternRequest(t *testing.T) {
	cfg := testconfig.Config(t, testconfig.Options{
		WhenUtc: "2018-06-01T00:00:00Z",
		RTJsons: testconfig.DefaultRTJson(),
	})
	ctx := model.WithConfig(context.Background(), cfg)
	d, err := makeGraphQLRequest(ctx, gql.NewDefaultHandler(), `query{routes(where:{feed_onestop_id:"BA",route_id:"03"}) {id onestop_id}}`, nil)
	if err != nil {
		t.Fatal("failed to get route id for tests")
	}
	routeID := gjson.Get(toJson(d), "routes.0.id").Int()
	assert.NotZero(t, routeID, "the route is found")
	routeOnestopID := gjson.Get(toJson(d), "routes.0.onestop_id").String()
	day := "2018-05-30"

	// patternKeys checks that the route has patterns and that each carries the
	// keys every pattern has, whatever the include flags.
	patternKeys := func(t *testing.T, jj string) []gjson.Result {
		pats := gjson.Get(jj, "routes.0.patterns").Array()
		assert.NotEmpty(t, pats, "patterns returned")
		for i, pat := range pats {
			for _, key := range []string{"stop_pattern_id", "direction_id", "count", "service_date"} {
				assert.True(t, pat.Get(key).Exists(), "pattern %d has %s", i, key)
			}
		}
		return pats
	}

	testcases := []testCase{
		{
			name:         "route key is an id",
			h:            RoutePatternRequest{RouteKey: strconv.Itoa(int(routeID)), ServiceDate: day},
			selector:     "routes.#.route_id",
			expectSelect: []string{"03"},
		},
		{
			name:         "route key is a onestop id",
			h:            RoutePatternRequest{RouteKey: routeOnestopID, ServiceDate: day},
			selector:     "routes.#.route_id",
			expectSelect: []string{"03"},
		},
		{
			name:         "route key is a feed and route id",
			h:            RoutePatternRequest{RouteKey: "BA:03", ServiceDate: day},
			selector:     "routes.#.route_id",
			expectSelect: []string{"03"},
		},
		{
			// A key in none of the three forms matches no route rather than every one.
			name:         "route key 0",
			h:            RoutePatternRequest{RouteKey: "0", ServiceDate: day},
			selector:     "routes.#.route_id",
			expectLength: 0,
		},
		{
			name:         "route key is a negative id",
			h:            RoutePatternRequest{RouteKey: "-1", ServiceDate: day},
			selector:     "routes.#.route_id",
			expectLength: 0,
		},
		{
			name:         "route key is a feed without a route id",
			h:            RoutePatternRequest{RouteKey: "BA:", ServiceDate: day},
			selector:     "routes.#.route_id",
			expectLength: 0,
		},
		{
			name: "no includes",
			h:    RoutePatternRequest{RouteKey: "BA:03", ServiceDate: day},
			f: func(t *testing.T, jj string) {
				for i, pat := range patternKeys(t, jj) {
					assert.Equal(t, day, pat.Get("service_date").String(), "pattern %d", i)
					for _, key := range []string{"stops", "shapes", "timetable"} {
						assert.False(t, pat.Get(key).Exists(), "pattern %d has no %s", i, key)
					}
				}
			},
		},
		{
			name: "date's trips",
			h:    RoutePatternRequest{RouteKey: "BA:03", ServiceDate: day, IncludeTimetable: true},
			f: func(t *testing.T, jj string) {
				patternKeys(t, jj)
				// Pattern 31 runs 26 trips on this Wednesday.
				pat := gjson.Get(jj, "routes.0.patterns.#(stop_pattern_id==31)")
				assert.Equal(t, int64(26), pat.Get("count").Int())
				assert.Equal(t, int64(26), pat.Get("timetable.trips.#").Int())
			},
		},
		{
			name: "array encoding",
			h:    RoutePatternRequest{RouteKey: "BA:03", ServiceDate: day, IncludeTimetable: true, IncludeShapes: true, IncludeStops: true},
			f: func(t *testing.T, jj string) {
				for i, pat := range patternKeys(t, jj) {
					assert.NotEmpty(t, pat.Get("stops").Array(), "pattern %d stops", i)
					var ids []int64
					for k, shape := range pat.Get("shapes").Array() {
						ids = append(ids, shape.Get("id").Int())
						coords := shape.Get("geometry.coordinates").Array()
						assert.NotEmpty(t, coords, "pattern %d shape %d", i, k)
						assert.Len(t, shape.Get("distances").Array(), len(coords), "pattern %d shape %d", i, k)
						assert.False(t, shape.Get("polyline").Exists(), "pattern %d shape %d", i, k)
						assert.False(t, shape.Get("polyline6").Exists(), "pattern %d shape %d", i, k)
					}
					// Every timetable trip finds its shape among the pattern's.
					for j, id := range pat.Get("timetable.trips.#.shape.id").Array() {
						assert.Contains(t, ids, id.Int(), "pattern %d trip %d", i, j)
					}
					for _, grid := range []string{"arrival_times", "departure_times", "shape_distances"} {
						assert.True(t, pat.Get("timetable."+grid+".values").Exists(), "pattern %d %s", i, grid)
						assert.False(t, pat.Get("timetable."+grid+".polyline").Exists(), "pattern %d %s", i, grid)
					}
				}
			},
		},
		{
			name: "polyline encoding",
			h:    RoutePatternRequest{RouteKey: "BA:03", ServiceDate: day, IncludeTimetable: true, IncludeShapes: true, Encoding: "polyline"},
			f: func(t *testing.T, jj string) {
				for i, pat := range patternKeys(t, jj) {
					for k, shape := range pat.Get("shapes").Array() {
						assert.NotEmpty(t, shape.Get("polyline").String(), "pattern %d shape %d", i, k)
						assert.False(t, shape.Get("polyline6").Exists(), "pattern %d shape %d", i, k)
						assert.False(t, shape.Get("geometry").Exists(), "pattern %d shape %d", i, k)
						assert.NotEmpty(t, shape.Get("distances").Array(), "pattern %d shape %d keeps its distances", i, k)
					}
					for _, grid := range []string{"arrival_times", "departure_times", "shape_distances"} {
						assert.NotEmpty(t, pat.Get("timetable."+grid+".polyline").Array(), "pattern %d %s", i, grid)
						assert.False(t, pat.Get("timetable."+grid+".values").Exists(), "pattern %d %s", i, grid)
					}
				}
			},
		},
		{
			name: "polyline6 encoding",
			h:    RoutePatternRequest{RouteKey: "BA:03", ServiceDate: day, IncludeTimetable: true, IncludeShapes: true, Encoding: "polyline6"},
			f: func(t *testing.T, jj string) {
				for i, pat := range patternKeys(t, jj) {
					for k, shape := range pat.Get("shapes").Array() {
						assert.NotEmpty(t, shape.Get("polyline6").String(), "pattern %d shape %d", i, k)
						assert.False(t, shape.Get("polyline").Exists(), "pattern %d shape %d", i, k)
						assert.False(t, shape.Get("geometry").Exists(), "pattern %d shape %d", i, k)
					}
					// The grids hold whole numbers, the same polyline either way.
					assert.NotEmpty(t, pat.Get("timetable.departure_times.polyline").Array(), "pattern %d", i)
				}
			},
		},
		{
			name: "service window used by default",
			h:    RoutePatternRequest{RouteKey: "BA:03", ServiceDate: "2030-01-02", IncludeTimetable: true},
			f: func(t *testing.T, jj string) {
				// Far outside the feed's dates, the fall-back week's Wednesday runs.
				for i, pat := range patternKeys(t, jj) {
					assert.NotEmpty(t, pat.Get("timetable.trips").Array(), "pattern %d", i)
				}
			},
		},
		{
			name: "service window turned off",
			h:    RoutePatternRequest{RouteKey: "BA:03", ServiceDate: "2030-01-02", UseServiceWindow: new(bool)},
			f: func(t *testing.T, jj string) {
				// The route is found, but no trip runs on the date itself.
				assert.Equal(t, "03", gjson.Get(jj, "routes.0.route_id").String())
				assert.Empty(t, gjson.Get(jj, "routes.0.patterns").Array())
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}

// TestRoutePatternRequestParams sends requests through the REST server, which
// refuses bad parameters before any query runs.
func TestRoutePatternRequestParams(t *testing.T) {
	_, restSrv, _ := testHandlersWithOptions(t, testconfig.Options{
		WhenUtc: "2018-06-01T00:00:00Z",
		Storage: testdata.Path("server", "tmp"),
	})
	tests := []struct {
		name         string
		path         string
		wantStatus   int
		wantContains string
	}{
		{"service date", "/routes/BA:03/patterns?service_date=2018-05-30", http.StatusOK, `"patterns"`},
		{"relative date", "/routes/BA:03/patterns?relative_date=wednesday", http.StatusOK, `"patterns"`},
		{"no date", "/routes/BA:03/patterns", http.StatusBadRequest, "service_date or relative_date is required"},
		{"malformed service date", "/routes/BA:03/patterns?service_date=2018-02-30", http.StatusBadRequest, "is not a date in YYYY-MM-DD form"},
		{"unknown relative date", "/routes/BA:03/patterns?relative_date=someday", http.StatusBadRequest, "is not a relative date"},
		{"array encoding", "/routes/BA:03/patterns?service_date=2018-05-30&encoding=array", http.StatusOK, `"patterns"`},
		{"polyline encoding", "/routes/BA:03/patterns?service_date=2018-05-30&encoding=polyline", http.StatusOK, `"patterns"`},
		{"polyline6 encoding", "/routes/BA:03/patterns?service_date=2018-05-30&encoding=polyline6", http.StatusOK, `"patterns"`},
		{"unknown encoding", "/routes/BA:03/patterns?service_date=2018-05-30&encoding=wkt", http.StatusBadRequest, "encoding must be array, polyline or polyline6"},
		// A parameter that fails to parse anywhere is the caller's error too.
		{"malformed bbox", "/routes?bbox=1,2,3", http.StatusBadRequest, "parameter error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()
			restSrv.ServeHTTP(rr, req)
			assert.Equal(t, tt.wantStatus, rr.Result().StatusCode, tt.path)
			assert.Contains(t, rr.Body.String(), tt.wantContains, tt.path)
		})
	}
}
