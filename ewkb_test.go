package gotransit

import (
	"testing"

	geom "github.com/twpayne/go-geom"
)

func Test_slEncode_LineStringM(t *testing.T) {
	coords := []float64{}
	coords = append(coords, []float64{1, 2, 3}...)
	coords = append(coords, []float64{4, 5, 6}...)
	coords = append(coords, []float64{7, 8, 9}...)
	g := geom.NewLineStringFlat(geom.XYM, coords)
	g.SetSRID(4326)
	slEncode(g)
}

func Test_slEncode_Point(t *testing.T) {
	lon := -116.40094
	lat := 36.641496
	g := geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{lon, lat})
	g.SetSRID(4326)
	slEncode(g)
}

func Test_slDecode_Point(t *testing.T) {
	lon := -116.40094
	lat := 36.641496
	g := geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{lon, lat})
	g.SetSRID(4326)
	w, _ := slEncode(g)
	ret, err := slDecode(w)
	if err != nil {
		t.Error(err)
	}
	if _, ok := ret.(*geom.Point); !ok {
		t.Error("failed")
	}
}

func Test_slDecode_LineStringM(t *testing.T) {
	coords := []float64{}
	coords = append(coords, []float64{1, 2, 3}...)
	coords = append(coords, []float64{4, 5, 6}...)
	coords = append(coords, []float64{7, 8, 9}...)
	g := geom.NewLineStringFlat(geom.XYM, coords)
	g.SetSRID(4326)
	w, _ := slEncode(g)
	ret, err := slDecode(w)
	if err != nil {
		t.Error(err)
	}
	if _, ok := ret.(*geom.LineString); !ok {
		t.Error("failed")
	}

}
