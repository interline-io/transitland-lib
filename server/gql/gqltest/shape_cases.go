package gqltest

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// BART fixture (testdata/server/gtfs/bart.zip): 12 shapes 01_shp..20_shp, each
// used by exactly one route/direction (e.g. 02_shp -> route 01, direction 1).
// All BART routes are route_type 1 (metro).
const bartShapeSha1 = "e535eb2b3b9ac3ef15d82c56575e914575e732e0"

func feedVersionResolverShapesTestcases(t testing.TB, cfg model.Config) []testcase {
	testcases := []testcase{
		{
			Name:     "shapes",
			Query:    `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes { shape_id }}}`,
			Vars:     hw{"sha1": bartShapeSha1},
			Selector: "feed_versions.0.shapes.#.shape_id",
			SelectExpect: []string{
				"01_shp", "02_shp", "03_shp", "04_shp", "05_shp", "06_shp",
				"07_shp", "08_shp", "11_shp", "12_shp", "19_shp", "20_shp",
			},
		},
		{
			Name:              "shapes count",
			Query:             `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes { shape_id }}}`,
			Vars:              hw{"sha1": bartShapeSha1},
			Selector:          "feed_versions.0.shapes.#.shape_id",
			SelectExpectCount: 12,
		},
		{
			Name:              "shapes limit",
			Query:             `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes(limit:3) { shape_id }}}`,
			Vars:              hw{"sha1": bartShapeSha1},
			Selector:          "feed_versions.0.shapes.#.shape_id",
			SelectExpectCount: 3,
		},
		{
			Name:               "shape geometry type",
			Query:              `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes { geometry }}}`,
			Vars:               hw{"sha1": bartShapeSha1},
			Selector:           "feed_versions.0.shapes.#.geometry.type",
			SelectExpectUnique: []string{"LineString"},
		},
		{
			Name:               "shape generated",
			Query:              `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes { generated }}}`,
			Vars:               hw{"sha1": bartShapeSha1},
			Selector:           "feed_versions.0.shapes.#.generated",
			SelectExpectUnique: []string{"false"},
		},
		// where: shape_id
		{
			Name:         "where shape_id",
			Query:        `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes(where:{shape_id:"02_shp"}) { shape_id }}}`,
			Vars:         hw{"sha1": bartShapeSha1},
			Selector:     "feed_versions.0.shapes.#.shape_id",
			SelectExpect: []string{"02_shp"},
		},
		{
			Name:         "where shape_id not found",
			Query:        `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes(where:{shape_id:"nope"}) { shape_id }}}`,
			Vars:         hw{"sha1": bartShapeSha1},
			Selector:     "feed_versions.0.shapes.#.shape_id",
			SelectExpect: []string{},
		},
		// where: route_type (trip->route join)
		{
			Name:              "where route_type 1",
			Query:             `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes(where:{route_type:1}) { shape_id }}}`,
			Vars:              hw{"sha1": bartShapeSha1},
			Selector:          "feed_versions.0.shapes.#.shape_id",
			SelectExpectCount: 12,
		},
		{
			Name:         "where route_type none",
			Query:        `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes(where:{route_type:3}) { shape_id }}}`,
			Vars:         hw{"sha1": bartShapeSha1},
			Selector:     "feed_versions.0.shapes.#.shape_id",
			SelectExpect: []string{},
		},
	}
	return testcases
}

func shapeResolverTripsTestcases(t testing.TB, cfg model.Config) []testcase {
	// shapes(where:{shape_id:"02_shp"}) yields exactly one shape, so shapes.0 is
	// unambiguous. Every BART trip on 02_shp runs route 01, direction 1.
	testcases := []testcase{
		{
			Name:               "shape trips route",
			Query:              `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes(where:{shape_id:"02_shp"}) { trips(limit:1000) { route { route_id } }}}}`,
			Vars:               hw{"sha1": bartShapeSha1},
			Selector:           "feed_versions.0.shapes.0.trips.#.route.route_id",
			SelectExpectUnique: []string{"01"},
		},
		{
			Name:               "shape trips direction",
			Query:              `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes(where:{shape_id:"02_shp"}) { trips(limit:1000) { direction_id }}}}`,
			Vars:               hw{"sha1": bartShapeSha1},
			Selector:           "feed_versions.0.shapes.0.trips.#.direction_id",
			SelectExpectUnique: []string{"1"},
		},
		{
			Name:  "shape trips contains known trip",
			Query: `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes(where:{shape_id:"02_shp"}) { trips(limit:1000) { trip_id }}}}`,
			Vars:  hw{"sha1": bartShapeSha1},
			F: func(t *testing.T, jj string) {
				var tripIDs []string
				for _, tr := range gjson.Get(jj, "feed_versions.0.shapes.0.trips").Array() {
					tripIDs = append(tripIDs, tr.Get("trip_id").String())
				}
				assert.Contains(t, tripIDs, "3850526WKDY")
			},
		},
	}
	return testcases
}

// feedVersionResolverShapesCursorTestcases pages through a feed version's
// shapes with after.
func feedVersionResolverShapesCursorTestcases(t testing.TB, cfg model.Config) []testcase {
	ctx := model.WithConfig(context.Background(), cfg)

	sha1 := bartShapeSha1
	fvs, err := cfg.Finder.FindFeedVersions(ctx, nil, nil, nil, &model.FeedVersionFilter{Sha1: &sha1})
	if err != nil {
		t.Fatal(err)
	}
	if len(fvs) == 0 {
		t.Fatal("bart feed version not found")
	}
	allEnts, err := cfg.Finder.FindShapesByFeedVersion(ctx, fvs[0].ID, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	needAtLeast(t, 2, "shapes", allEnts)
	allIds := []string{}
	for _, ent := range allEnts {
		allIds = append(allIds, ent.ShapeID.Val)
	}

	testcases := []testcase{
		{
			Name:         "no cursor",
			Query:        `query($sha1:String!){feed_versions(where:{sha1:$sha1}){shapes(limit:100){id shape_id}}}`,
			Vars:         hw{"sha1": bartShapeSha1},
			Selector:     "feed_versions.0.shapes.#.shape_id",
			SelectExpect: allIds,
		},
		{
			Name:         "after first",
			Query:        `query($sha1:String!,$after:Int!){feed_versions(where:{sha1:$sha1}){shapes(after:$after,limit:100){shape_id}}}`,
			Vars:         hw{"sha1": bartShapeSha1, "after": allEnts[0].ID},
			Selector:     "feed_versions.0.shapes.#.shape_id",
			SelectExpect: allIds[1:],
		},
	}
	return testcases
}
