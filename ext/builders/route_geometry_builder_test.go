package builders

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/interline-io/transitland-lib/internal/testreader"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/stretchr/testify/assert"
	"github.com/twpayne/go-geom"
)

func Test_sortMap(t *testing.T) {
	// Sorted by highest int first, then by key
	tc := map[string]int{
		"f": 10,
		"a": 10,
		"y": 5,
		"x": 5,
		"c": 100,
	}
	assert.Equal(t, []string{"c", "a", "f", "x", "y"}, sortMap(tc))
}

func TestRouteGeometryBuilder(t *testing.T) {
	type testcase struct {
		RouteID           string
		ExpectLength      float64
		ExpectLineStrings int
	}
	type testgroup struct {
		URL   string
		Cases []testcase
	}
	groups := map[string]testgroup{
		"Caltrain": {
			testreader.ExampleFeedCaltrain.URL,
			[]testcase{
				{RouteID: "Bu-130", ExpectLength: 75274.982973, ExpectLineStrings: 4},
				{RouteID: "Lo-130", ExpectLength: 75274.982973, ExpectLineStrings: 4},
			},
		},
		"BART": {
			testreader.ExampleFeedBART.URL,
			[]testcase{
				{RouteID: "07", ExpectLength: 58890.123340, ExpectLineStrings: 2},
				{RouteID: "03", ExpectLength: 65574.875547, ExpectLineStrings: 2},
				{RouteID: "05", ExpectLength: 69808.892350, ExpectLineStrings: 2},
				{RouteID: "11", ExpectLength: 62611.513781, ExpectLineStrings: 2},
				{RouteID: "19", ExpectLength: 5270.877425, ExpectLineStrings: 2},
			},
		},
		"TriMet-2Routes": {
			testpath.RelPath("testdata/gtfs-external/trimet-2routes.zip"),
			[]testcase{
				{RouteID: "193", ExpectLength: 6452.065660, ExpectLineStrings: 4},
				{RouteID: "200", ExpectLength: 23012.874312, ExpectLineStrings: 5},
			},
		},
	}
	for groupName, testGroup := range groups {
		t.Run(groupName, func(t *testing.T) {
			e := NewRouteGeometryBuilder()
			_, writer, err := newMockCopier(testGroup.URL, e)
			if err != nil {
				t.Fatal(err)
			}
			routeGeoms := map[string]*RouteGeometry{}
			for _, ent := range writer.Reader.OtherList {
				switch v := ent.(type) {
				case *RouteGeometry:
					routeGeoms[v.RouteID] = v
				}
			}
			for _, tc := range testGroup.Cases {
				t.Run(tc.RouteID, func(t *testing.T) {
					rg, ok := routeGeoms[tc.RouteID]
					if !ok {
						t.Errorf("no route: %s", tc.RouteID)
						return
					}

					pts := []tlxy.Point{}
					for _, c := range rg.Geometry.Val.Coords() {
						pts = append(pts, tlxy.Point{Lon: c[0], Lat: c[1]})
					}
					length := tlxy.LengthHaversine(pts)
					assert.InEpsilonf(t, length, tc.ExpectLength, 1.0, "got %f expect %f", length, tc.ExpectLength)
					if mls, ok := rg.CombinedGeometry.Val.(*geom.MultiLineString); !ok {
						t.Errorf("not MultiLineString")
					} else {
						// t.Logf(`{RouteID:"%s", ExpectLength: %f, ExpectLineStrings: %d},`+"\n", tc.RouteID, length, mls.NumLineStrings())
						assert.Equal(t, tc.ExpectLineStrings, mls.NumLineStrings())
					}
				})
			}
		})
	}
}

// The representative shapes must be able to reconstruct what the geometry columns
// hold today: one pointer per linestring in CombinedGeometry, ranked, with rank 0
// the linestring stored as Geometry. The route-level scalars must likewise be
// recoverable as aggregates over the pointers.
func TestRouteGeometryBuilder_RepresentativeShapes(t *testing.T) {
	groups := map[string]struct {
		URL               string
		RouteID           string
		ExpectLineStrings int
	}{
		"Caltrain": {testreader.ExampleFeedCaltrain.URL, "Bu-130", 4},
		"BART":     {testreader.ExampleFeedBART.URL, "07", 2},
		"TriMet":   {testpath.RelPath("testdata/gtfs-external/trimet-2routes.zip"), "200", 5},
	}
	for groupName, tg := range groups {
		t.Run(groupName, func(t *testing.T) {
			_, writer, err := newMockCopier(tg.URL, NewRouteGeometryBuilder())
			if err != nil {
				t.Fatal(err)
			}
			routeGeoms := map[string]*RouteGeometry{}
			selected := map[string][]*RouteRepresentativeShape{}
			for _, ent := range writer.Reader.OtherList {
				switch v := ent.(type) {
				case *RouteGeometry:
					routeGeoms[v.RouteID] = v
				case *RouteRepresentativeShape:
					selected[v.RouteID] = append(selected[v.RouteID], v)
				}
			}
			assert.NotEmpty(t, routeGeoms)

			for rid, rg := range routeGeoms {
				sel := selected[rid]
				mls, ok := rg.CombinedGeometry.Val.(*geom.MultiLineString)
				if !ok {
					t.Errorf("route %s: not MultiLineString", rid)
					continue
				}
				// One pointer per linestring: this is what makes the combined geometry
				// reconstructable by collecting the pointed-at shapes.
				assert.Equal(t, mls.NumLineStrings(), len(sel), "route %s: pointers != combined linestrings", rid)

				// Ranks are 0..n-1, each exactly once -- what UNIQUE(route_id, rank)
				// enforces, and what lets rank = 0 join without fanning out.
				seenRank := map[int]bool{}
				for _, s := range sel {
					assert.False(t, seenRank[s.Rank], "route %s: duplicate rank %d", rid, s.Rank)
					seenRank[s.Rank] = true
					assert.GreaterOrEqual(t, s.Rank, 0, "route %s", rid)
					assert.Less(t, s.Rank, len(sel), "route %s", rid)
					assert.True(t, s.DirectionID.Valid, "route %s: direction_id not set", rid)
					assert.NotEmpty(t, s.ShapeID, "route %s: shape_id not set", rid)
				}

				// Route-level scalars as aggregates over the pointers. Phase 3 drops the
				// geometry columns and derives these, so the identity has to hold.
				maxLength := 0.0
				anyGenerated := false
				for _, s := range sel {
					if s.Length.Val > maxLength {
						maxLength = s.Length.Val
					}
					anyGenerated = anyGenerated || s.Generated
				}
				assert.Equal(t, rg.Length.Val, maxLength, "route %s: length != max(shape length)", rid)
				assert.Equal(t, rg.Generated, anyGenerated, "route %s: generated != any(shape generated)", rid)
			}

			assert.Equal(t, tg.ExpectLineStrings, len(selected[tg.RouteID]), "route %s", tg.RouteID)
		})
	}
}

func TestRouteGeometryBuilder_FlexFeed(t *testing.T) {
	reader, err := tlcsv.NewReader(testpath.RelPath("testdata/gtfs-external/ctran-flex.zip"))
	if err != nil {
		t.Fatal(err)
	}
	writer := direct.NewWriter()
	cpOpts := copier.Options{
		CreateMissingShapes: true,
	}
	cpOpts.AddExtension(NewRouteGeometryBuilder())

	cpResult, err := copier.CopyWithOptions(context.Background(), reader, writer, cpOpts)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, cpResult)
}
