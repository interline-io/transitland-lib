package gql

import (
	"context"
	"fmt"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/gql/internal/gqlcase"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
	"github.com/twpayne/go-polyline"
)

func TestRouteResolver(t *testing.T) {
	vars := hw{"route_id": "03"}
	testcases := []testcase{
		{
			Name:         "route_stop_buffer stop_points 10m",
			Query:        `query($route_id: String!) { routes(where:{route_id:$route_id}) {route_stop_buffer(radius: 100.0) {stop_points	stop_buffer	stop_convexhull}}}`,
			Vars:         vars,
			Selector:     "routes.0.route_stop_buffer.stop_points.type",
			SelectExpect: []string{"MultiPoint"},
		},
		{
			Name:         "route_stop_buffer stop_buffer 10m",
			Query:        `query($route_id: String!) { routes(where:{route_id:$route_id}) {route_stop_buffer(radius: 100.0) {stop_points	stop_buffer	stop_convexhull}}}`,
			Vars:         vars,
			Selector:     "routes.0.route_stop_buffer.stop_buffer.type",
			SelectExpect: []string{"MultiPolygon"},
		},
		{
			Name:         "route_stop_buffer stop_convexhull 10m",
			Query:        `query($route_id: String!) { routes(where:{route_id:$route_id}) {route_stop_buffer(radius: 100.0) {stop_points	stop_buffer	stop_convexhull}}}`,
			Vars:         vars,
			Selector:     "routes.0.route_stop_buffer.stop_convexhull.type",
			SelectExpect: []string{"Polygon"},
		},
		// route attributes
		{
			Name: "route attributes",
			Query: `{
						routes(where: {feed_onestop_id: "BA", route_id: "01"}) {
						  route_id
						  route_attribute {
							category
							subcategory
							running_way
						  }
						}
					  }`,
			F: func(t *testing.T, jj string) {
				assert.EqualValues(t, 2, gjson.Get(jj, "routes.0.route_attribute.category").Int())
				assert.EqualValues(t, 201, gjson.Get(jj, "routes.0.route_attribute.subcategory").Int())
				assert.EqualValues(t, 1, gjson.Get(jj, "routes.0.route_attribute.running_way").Int())
			},
		},
	}
	c, _ := newTestClient(t)
	gqlcase.QueryTestcases(t, c, testcases)
}

// As for stops: both ways of matching an onestop_id must agree.
func TestRouteResolver_OnestopID_Materialized(t *testing.T) {
	testcases := []testcase{
		{
			Name:         "where onestop_id",
			Query:        `query {routes(where:{onestop_id:"r-9q9j-bullet"}) {route_id} }`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"Bu-130"},
		},
		{
			Name:         "where onestop_ids",
			Query:        `query {routes(where:{onestop_ids:["r-9q9j-bullet","r-9q9n-warmsprings~southfremont~richmond"]}) {route_id} }`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"Bu-130", "03"},
		},
		{
			Name:         "selected onestop_id",
			Query:        `query {routes(where:{onestop_id:"r-9q9j-bullet"}) {onestop_id} }`,
			Selector:     "routes.#.onestop_id",
			SelectExpect: []string{"r-9q9j-bullet"},
		},
		{
			Name:         "unknown onestop_id",
			Query:        `query {routes(where:{onestop_id:"r-0000-nowhere"}) {route_id} }`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{},
		},
		// As for stops. This route's onestop_id changed between versions.
		{
			Name:         "previous allowed, current id",
			Query:        `query {routes(where:{onestop_id:"r-9q9-antioch~sfia~millbrae", allow_previous_onestop_ids:true}) {route_id} }`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"01"},
		},
		{
			Name:  "previous allowed, previous id",
			Query: `query {routes(where:{onestop_id:"r-9q9-pittsburg~baypoint~sfia~millbrae", allow_previous_onestop_ids:true}) {route_id onestop_id} }`,
			Sel: []testcaseSelector{
				{Selector: "routes.#.route_id", Expect: []string{"01"}},
				{Selector: "routes.#.onestop_id", Expect: []string{"r-9q9-antioch~sfia~millbrae"}},
			},
		},
		{
			Name:         "previous allowed, current and previous ids",
			Query:        `query {routes(where:{onestop_ids:["r-9q9j-bullet","r-9q9-pittsburg~baypoint~sfia~millbrae"], allow_previous_onestop_ids:true}) {route_id} }`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"Bu-130", "01"},
		},
		{
			Name:         "previous allowed, unknown id",
			Query:        `query {routes(where:{onestop_id:"r-0000-nowhere", allow_previous_onestop_ids:true}) {route_id} }`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{},
		},
	}
	for _, materialized := range []bool{false, true} {
		t.Run(fmt.Sprintf("UseMaterialized=%t", materialized), func(t *testing.T) {
			c, _ := newTestClientWithOpts(t, testconfig.Options{UseMaterialized: materialized})
			gqlcase.QueryTestcases(t, c, testcases)
		})
	}
}

// As for stops, with the operator filter the Explore route page sends. Here the
// old BART version gives route 01 the onestop ID of Caltrain's Baby Bullet.
func TestRouteResolver_OnestopID_PreviousFiltered(t *testing.T) {
	bullet := "r-9q9j-bullet"
	testcases := []testcase{
		{
			Name:         "current match",
			Query:        `query($osid:String!){routes(where:{onestop_id:$osid, allow_previous_onestop_ids:true}) {route_id} }`,
			Vars:         hw{"osid": bullet},
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"Bu-130"},
		},
		{
			Name:         "current match filtered out",
			Query:        `query($osid:String!){routes(where:{onestop_id:$osid, allow_previous_onestop_ids:true, operator_onestop_id:"o-9q9-bayarearapidtransit"}) {route_id} }`,
			Vars:         hw{"osid": bullet},
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"01"},
		},
	}
	for _, materialized := range []bool{false, true} {
		t.Run(fmt.Sprintf("UseMaterialized=%t", materialized), func(t *testing.T) {
			testconfig.ConfigTxRollback(t, testconfig.Options{UseMaterialized: materialized}, func(cfg model.Config) {
				ctx := context.Background()
				sha1 := "dd7aca4a8e4c90908fd3603c097fabee75fea907"
				fvs, err := cfg.Finder.FindFeedVersions(ctx, nil, nil, nil, &model.FeedVersionFilter{Sha1: &sha1})
				if err != nil || len(fvs) != 1 {
					t.Fatalf("feed version %s: %v", sha1, err)
				}
				previous := stats.FeedVersionStats{RouteOnestopIDs: []dmfr.FeedVersionRouteOnestopID{{EntityID: "01", OnestopID: bullet}}}
				if err := stats.WriteFeedVersionStats(ctx, cfg.Adapter, previous, fvs[0].ID, stats.WriteOptions{Stats: []string{stats.StatOnestopIDs}}); err != nil {
					t.Fatal(err)
				}
				srv := model.AddConfigAndPerms(cfg, NewDefaultHandler())
				gqlcase.QueryTestcases(t, client.New(usercheck.UserDefaultMiddleware("test")(srv)), testcases)
			})
		})
	}
}

func TestRouteResolver_PreviousOnestopID(t *testing.T) {
	testcases := []testcase{
		{
			Name:         "default",
			Query:        `query($osid:String!, $previous:Boolean!) { routes(where:{onestop_id:$osid, allow_previous_onestop_ids:$previous}) { route_id onestop_id }}`,
			Vars:         hw{"osid": "r-9q9-antioch~sfia~millbrae", "previous": false},
			Selector:     "routes.#.onestop_id",
			SelectExpect: []string{"r-9q9-antioch~sfia~millbrae"},
		},
		{
			Name:         "old id no result",
			Query:        `query($osid:String!, $previous:Boolean!) { routes(where:{onestop_id:$osid, allow_previous_onestop_ids:$previous}) { route_id onestop_id }}`,
			Vars:         hw{"osid": "r-9q9-pittsburg~baypoint~sfia~millbrae", "previous": false},
			Selector:     "routes.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:         "old id specify fv",
			Query:        `query($osid:String!, $previous:Boolean!) { routes(where:{onestop_id:$osid, allow_previous_onestop_ids:$previous, feed_version_sha1:"dd7aca4a8e4c90908fd3603c097fabee75fea907"}) { route_id onestop_id }}`,
			Vars:         hw{"osid": "r-9q9-pittsburg~baypoint~sfia~millbrae", "previous": false},
			Selector:     "routes.#.onestop_id",
			SelectExpect: []string{"r-9q9-pittsburg~baypoint~sfia~millbrae"},
		},
		{
			Name:         "use previous",
			Query:        `query($osid:String!, $previous:Boolean!) { routes(where:{onestop_id:$osid, allow_previous_onestop_ids:$previous}) { route_id onestop_id }}`,
			Vars:         hw{"osid": "r-9q9-pittsburg~baypoint~sfia~millbrae", "previous": true},
			Selector:     "routes.#.onestop_id",
			SelectExpect: []string{"r-9q9-antioch~sfia~millbrae"},
		},
	}
	c, _ := newTestClient(t)
	gqlcase.QueryTestcases(t, c, testcases)
}

func TestRouteResolver_Segments(t *testing.T) {
	testcases := []testcase{
		{
			Name:         "two segments",
			Query:        `query($fsid:String!, $route_id:String!) { routes(where:{route_id:$route_id, feed_onestop_id:$fsid}) { route_id segments { id way_id } }}`,
			Vars:         hw{"fsid": "HA", "route_id": "12"},
			Selector:     "routes.0.segments.#.way_id",
			SelectExpect: []string{"645693994", "90865590"},
		},
		{
			Name:         "single segment",
			Query:        `query($fsid:String!, $route_id:String!) { routes(where:{route_id:$route_id, feed_onestop_id:$fsid}) { route_id segments { id way_id } }}`,
			Vars:         hw{"fsid": "HA", "route_id": "19"},
			Selector:     "routes.0.segments.#.way_id",
			SelectExpect: []string{"645693994"},
		},
		{
			Name:     "geometry",
			Query:    `query($fsid:String!, $route_id:String!) { routes(where:{route_id:$route_id, feed_onestop_id:$fsid}) { route_id segments { id way_id geometry } }}`,
			Vars:     hw{"fsid": "HA", "route_id": "12"},
			Selector: "routes.0.segments.#.geometry",
			SelectExpect: []string{
				`{"coordinates":[[-82.458062,27.954493],[-82.458044,27.954442]],"type":"LineString"}`,
				`{"coordinates":[[-82.437785,28.058093],[-82.438163,28.058095],[-82.438219,28.058091],[-82.438277,28.058079],[-82.43833,28.058061],[-82.438375,28.058036],[-82.438415,28.058002],[-82.438449,28.057959],[-82.438476,28.057911],[-82.438493,28.05786],[-82.438501,28.057806],[-82.438504,28.057755],[-82.438504,28.057713],[-82.438504,28.057535]],"type":"LineString"}`,
			},
		},
		{
			Name:         "segments to patterns multiple",
			Query:        `query($fsid:String!, $route_id:String!) { routes(where:{route_id:$route_id, feed_onestop_id:$fsid}) { route_id segments { id way_id segment_patterns { stop_pattern_id }} }}`,
			Vars:         hw{"fsid": "HA", "route_id": "12"},
			Selector:     "routes.0.segments.#.segment_patterns.#.stop_pattern_id",
			SelectExpect: []string{"[42,40]", "[42]"}, // multiple lookup results look like this
		},
		{
			Name:         "segments to patterns single",
			Query:        `query($fsid:String!, $route_id:String!) { routes(where:{route_id:$route_id, feed_onestop_id:$fsid}) { route_id segments { id way_id segment_patterns { stop_pattern_id }} }}`,
			Vars:         hw{"fsid": "HA", "route_id": "19"},
			Selector:     "routes.0.segments.#.segment_patterns.#.stop_pattern_id",
			SelectExpect: []string{"[42,40]"},
		},
	}
	c, _ := newTestClient(t)
	gqlcase.QueryTestcases(t, c, testcases)
}

func TestRoutePatternGridEncodings(t *testing.T) {
	// Each trip has a gap, the last trip's at the first stop, and three values,
	// so a difference from the first value can't pass for one from the previous.
	cell, gap := tt.NewInt, tt.Int{}
	grid := &model.RouteStopPatternTimetableGrid{Values: [][]tt.Int{
		{cell(100), cell(400), gap},
		{cell(160), gap, cell(700)},
		{gap, cell(520), cell(790)},
		{cell(250), cell(610), cell(880)},
	}}
	r := &routePatternGridResolver{}
	delta, err := r.Delta(context.Background(), grid)
	if assert.NoError(t, err) {
		assert.Equal(t, [][]*int{{ptr(100), ptr(400), nil}, {ptr(60), nil, ptr(700)}, {nil, ptr(120), ptr(90)}, {ptr(150), ptr(210), ptr(180)}}, delta)
	}
	lines, err := r.Polyline(context.Background(), grid)
	if assert.NoError(t, err) {
		// The wire format itself, since the encoder and the decoder below share a library.
		assert.Equal(t, []string{"gEwQ.", "wB._g@", ".oFz@", "kHwBz@"}, lines)
		var rows [][]*int64
		for _, line := range lines {
			rows = append(rows, decodePolylineRow(t, line))
		}
		want := [][]*int64{
			{ptr(int64(100)), ptr(int64(400)), nil},
			{ptr(int64(160)), nil, ptr(int64(700))},
			{nil, ptr(int64(520)), ptr(int64(790))},
			{ptr(int64(250)), ptr(int64(610)), ptr(int64(880))},
		}
		assert.Equal(t, want, undoDelta(rows))
	}

	// The same numbers as times encode the same way, in seconds.
	times := &model.RouteStopPatternTimetableTimeGrid{}
	for _, row := range grid.Values {
		var cells []tt.Seconds
		for _, v := range row {
			cells = append(cells, tt.Seconds{Option: v.Option})
		}
		times.Values = append(times.Values, cells)
	}
	tr := &routePatternTimeGridResolver{}
	timeDelta, err := tr.Delta(context.Background(), times)
	if assert.NoError(t, err) {
		assert.Equal(t, delta, timeDelta)
	}
	timeLines, err := tr.Polyline(context.Background(), times)
	if assert.NoError(t, err) {
		assert.Equal(t, lines, timeLines)
	}
}

// undoDelta adds each column's first non-null value back to the later ones.
func undoDelta(rows [][]*int64) [][]*int64 {
	ret := make([][]*int64, len(rows))
	for i, row := range rows {
		ret[i] = make([]*int64, len(row))
	}
	if len(rows) == 0 {
		return ret
	}
	for j := range rows[0] {
		var first *int64
		for i, row := range rows {
			if row[j] == nil {
				continue
			}
			v := *row[j]
			if first == nil {
				first = &v
			} else {
				v += *first
			}
			ret[i][j] = &v
		}
	}
	return ret
}

// decodePolylineRow reads one row of a grid's polyline, nil where '.'.
func decodePolylineRow(t *testing.T, s string) []*int64 {
	ret := make([]*int64, 0, len(s))
	buf, prev := []byte(s), int64(0)
	for len(buf) > 0 {
		if buf[0] == '.' {
			ret = append(ret, nil)
			buf = buf[1:]
			continue
		}
		d, rest, err := polyline.DecodeInt(buf)
		if !assert.NoError(t, err) {
			return ret
		}
		prev += int64(d)
		v := prev
		ret = append(ret, &v)
		buf = rest
	}
	return ret
}
