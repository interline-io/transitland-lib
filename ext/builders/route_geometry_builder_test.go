package builders

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/interline-io/transitland-lib/internal/testutil"
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
			testutil.ExampleFeedCaltrain.URL,
			[]testcase{
				{RouteID: "Bu-130", ExpectLength: 75274.982973, ExpectLineStrings: 4},
				{RouteID: "Lo-130", ExpectLength: 75274.982973, ExpectLineStrings: 4},
			},
		},
		"BART": {
			testutil.ExampleFeedBART.URL,
			[]testcase{
				{RouteID: "07", ExpectLength: 58890.123340, ExpectLineStrings: 2},
				{RouteID: "03", ExpectLength: 65574.875547, ExpectLineStrings: 2},
				{RouteID: "05", ExpectLength: 69808.892350, ExpectLineStrings: 2},
				{RouteID: "11", ExpectLength: 62611.513781, ExpectLineStrings: 2},
				{RouteID: "19", ExpectLength: 5270.877425, ExpectLineStrings: 2},
			},
		},
		"TriMet-2Routes": {
			testpath.RelPath("testdata/external/trimet-2routes.zip"),
			[]testcase{
				{RouteID: "193", ExpectLength: 6452.065660, ExpectLineStrings: 4},
				{RouteID: "200", ExpectLength: 23012.874312, ExpectLineStrings: 5},
			},
		},
	}
	for groupName, testGroup := range groups {
		t.Run(groupName, func(t *testing.T) {
			cp, writer, err := newMockCopier(testGroup.URL)
			if err != nil {
				t.Fatal(err)
			}
			e := NewRouteGeometryBuilder()
			cp.AddExtension(e)
			cpr := cp.Copy()
			if cpr.WriteError != nil {
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
