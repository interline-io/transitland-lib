package builders

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/internal/xy"
	"github.com/stretchr/testify/assert"
	"github.com/twpayne/go-geom"
)

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
				{RouteID: "Lo-130", ExpectLength: 75274.982973, ExpectLineStrings: 5},
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
			testutil.RelPath("test/data/external/trimet-2routes.zip"),
			[]testcase{
				{RouteID: "193", ExpectLength: 6452.065660, ExpectLineStrings: 4},
				{RouteID: "200", ExpectLength: 23012.874312, ExpectLineStrings: 7},
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

					pts := []xy.Point{}
					for _, c := range rg.Geometry.Coords() {
						pts = append(pts, xy.Point{Lon: c[0], Lat: c[1]})
					}
					length := xy.LengthHaversine(pts)
					assert.InEpsilonf(t, length, tc.ExpectLength, 1.0, "got %f expect %f", length, tc.ExpectLength)
					if mls, ok := rg.CombinedGeometry.Geometry.(*geom.MultiLineString); !ok {
						t.Errorf("not MultiLineString")
					} else {
						// fmt.Printf(`{RouteID:"%s", ExpectLength: %f, ExpectLineStrings: %d},`+"\n", tc.RouteID, length, mls.NumLineStrings())
						assert.Equal(t, mls.NumLineStrings(), tc.ExpectLineStrings)
					}
				})
			}
		})
	}
}
