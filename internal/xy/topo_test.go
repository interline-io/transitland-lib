package xy

import (
	"io/ioutil"
	"testing"

	geom "github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

func TestTopoSimplify(t *testing.T) {
	testcases := []struct {
		name string
		line [][]Point
	}{
		{
			"a",
			[][]Point{
				{{0, 1}, {0, 2}, {0, 3}},
				{{0, 1}, {0, 2}, {0, 3}},
			},
		},
		{
			"b",
			[][]Point{
				{{0, 1}, {0, 4}, {0, 2}, {0, 3}},
				{{0, 1}, {0, 2}, {0, 3}},
			},
		},
		{
			"c",
			[][]Point{
				{{0, 1}, {0, 1}, {0, 1}, {0, 1}},
				{{0, 1}, {0, 2}, {0, 3}},
			},
		},
		{
			"d",
			[][]Point{
				{{1, 1}, {1, 2}, {1, 3}},
				{{0, 1}, {0, 2}, {0, 3}},
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			TopoSimplify(tc.line)
		})
	}
}

func TestTopoSimplify_File(t *testing.T) {
	testcases := []struct {
		name string
		fn   string
	}{
		{
			"a",
			"../../test/data/geojson/mls.geojson",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := ioutil.ReadFile(tc.fn)
			if err != nil {
				t.Fatal(err)
			}
			fc := geojson.FeatureCollection{}
			if err := fc.UnmarshalJSON(data); err != nil {
				t.Fatal(err)
			}
			g, ok := fc.Features[0].Geometry.(*geom.MultiLineString)
			if !ok {
				return
			}
			var lines [][]Point
			for i := 0; i < g.NumLineStrings(); i++ {
				var line []Point
				for _, c := range g.LineString(i).Coords() {
					line = append(line, Point{c.X(), c.Y()})
				}
				lines = append(lines, line)
			}
			TopoSimplify(lines)
		})
	}
}
