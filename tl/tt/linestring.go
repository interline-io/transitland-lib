package tt

import (
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/twpayne/go-geom"
)

// LineString is an EWKB/SL encoded LineString
type LineString struct {
	GeometryOption[*geom.LineString]
}

func NewLineString(v *geom.LineString) LineString {
	return LineString{GeometryOption: NewGeometryOption(v)}
}

// NewLineStringFromFlatCoords returns a new LineString from flat (3) coordinates
func NewLineStringFromFlatCoords(coords []float64) LineString {
	g := geom.NewLineStringFlat(geom.XYM, coords)
	if g == nil {
		return LineString{}
	}
	g.SetSRID(4326)
	return LineString{GeometryOption: NewGeometryOption(g)}
}

func (g LineString) ToPoints() []tlxy.Point {
	var ret []tlxy.Point
	for _, c := range g.Val.Coords() {
		ret = append(ret, tlxy.Point{Lon: c[0], Lat: c[1]})
	}
	return ret
}

func (g LineString) ToLineM() tlxy.LineM {
	var ret []tlxy.Point
	var ms []float64
	for _, c := range g.Val.Coords() {
		ret = append(ret, tlxy.Point{Lon: c[0], Lat: c[1]})
		if len(c) > 2 {
			ms = append(ms, c[2])
		} else {
			ms = append(ms, 0)
		}
	}
	return tlxy.LineM{
		Coords: ret,
		Data:   ms,
	}
}
