package gql

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

func TestFeedVersionResolver_Shapes(t *testing.T) {
	testcases := []testcase{
		{
			name:     "shapes",
			query:    `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes { shape_id }}}`,
			vars:     hw{"sha1": bartShapeSha1},
			selector: "feed_versions.0.shapes.#.shape_id",
			selectExpect: []string{
				"01_shp", "02_shp", "03_shp", "04_shp", "05_shp", "06_shp",
				"07_shp", "08_shp", "11_shp", "12_shp", "19_shp", "20_shp",
			},
		},
		{
			name:              "shapes count",
			query:             `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes { shape_id }}}`,
			vars:              hw{"sha1": bartShapeSha1},
			selector:          "feed_versions.0.shapes.#.shape_id",
			selectExpectCount: 12,
		},
		{
			name:              "shapes limit",
			query:             `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes(limit:3) { shape_id }}}`,
			vars:              hw{"sha1": bartShapeSha1},
			selector:          "feed_versions.0.shapes.#.shape_id",
			selectExpectCount: 3,
		},
		{
			name:               "shape geometry type",
			query:              `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes { geometry }}}`,
			vars:               hw{"sha1": bartShapeSha1},
			selector:           "feed_versions.0.shapes.#.geometry.type",
			selectExpectUnique: []string{"LineString"},
		},
		{
			name:               "shape generated",
			query:              `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes { generated }}}`,
			vars:               hw{"sha1": bartShapeSha1},
			selector:           "feed_versions.0.shapes.#.generated",
			selectExpectUnique: []string{"false"},
		},
		// where: shape_id
		{
			name:         "where shape_id",
			query:        `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes(where:{shape_id:"02_shp"}) { shape_id }}}`,
			vars:         hw{"sha1": bartShapeSha1},
			selector:     "feed_versions.0.shapes.#.shape_id",
			selectExpect: []string{"02_shp"},
		},
		{
			name:         "where shape_id not found",
			query:        `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes(where:{shape_id:"nope"}) { shape_id }}}`,
			vars:         hw{"sha1": bartShapeSha1},
			selector:     "feed_versions.0.shapes.#.shape_id",
			selectExpect: []string{},
		},
		// where: route_type (trip->route join)
		{
			name:              "where route_type 1",
			query:             `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes(where:{route_type:1}) { shape_id }}}`,
			vars:              hw{"sha1": bartShapeSha1},
			selector:          "feed_versions.0.shapes.#.shape_id",
			selectExpectCount: 12,
		},
		{
			name:         "where route_type none",
			query:        `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes(where:{route_type:3}) { shape_id }}}`,
			vars:         hw{"sha1": bartShapeSha1},
			selector:     "feed_versions.0.shapes.#.shape_id",
			selectExpect: []string{},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

func TestShapeResolver_Trips(t *testing.T) {
	// shapes(where:{shape_id:"02_shp"}) yields exactly one shape, so shapes.0 is
	// unambiguous. Every BART trip on 02_shp runs route 01, direction 1.
	testcases := []testcase{
		{
			name:               "shape trips route",
			query:              `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes(where:{shape_id:"02_shp"}) { trips(limit:1000) { route { route_id } }}}}`,
			vars:               hw{"sha1": bartShapeSha1},
			selector:           "feed_versions.0.shapes.0.trips.#.route.route_id",
			selectExpectUnique: []string{"01"},
		},
		{
			name:               "shape trips direction",
			query:              `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes(where:{shape_id:"02_shp"}) { trips(limit:1000) { direction_id }}}}`,
			vars:               hw{"sha1": bartShapeSha1},
			selector:           "feed_versions.0.shapes.0.trips.#.direction_id",
			selectExpectUnique: []string{"1"},
		},
		{
			name:  "shape trips contains known trip",
			query: `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { shapes(where:{shape_id:"02_shp"}) { trips(limit:1000) { trip_id }}}}`,
			vars:  hw{"sha1": bartShapeSha1},
			f: func(t *testing.T, jj string) {
				var tripIDs []string
				for _, tr := range gjson.Get(jj, "feed_versions.0.shapes.0.trips").Array() {
					tripIDs = append(tripIDs, tr.Get("trip_id").String())
				}
				assert.Contains(t, tripIDs, "3850526WKDY")
			},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

// TestFeedVersionResolver_Shapes_Cursor exercises keyset (after) pagination,
// mirroring TestAgencyResolver_Cursor.
func TestFeedVersionResolver_Shapes_Cursor(t *testing.T) {
	c, cfg := newTestClient(t)
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
	if len(allEnts) < 2 {
		t.Fatalf("expected multiple shapes, got %d", len(allEnts))
	}
	allIds := []string{}
	for _, ent := range allEnts {
		allIds = append(allIds, ent.ShapeID.Val)
	}

	testcases := []testcase{
		{
			name:         "no cursor",
			query:        `query($sha1:String!){feed_versions(where:{sha1:$sha1}){shapes(limit:100){id shape_id}}}`,
			vars:         hw{"sha1": bartShapeSha1},
			selector:     "feed_versions.0.shapes.#.shape_id",
			selectExpect: allIds,
		},
		{
			name:         "after first",
			query:        `query($sha1:String!,$after:Int!){feed_versions(where:{sha1:$sha1}){shapes(after:$after,limit:100){shape_id}}}`,
			vars:         hw{"sha1": bartShapeSha1, "after": allEnts[0].ID},
			selector:     "feed_versions.0.shapes.#.shape_id",
			selectExpect: allIds[1:],
		},
	}
	queryTestcases(t, c, testcases)
}
