package tt

import (
	"github.com/interline-io/transitland-lib/tlxy"
	geom "github.com/twpayne/go-geom"
)

// Point is an EWKB/SL encoded point
type Point struct {
	GeometryOption[*geom.Point]
}

func (g *Point) ToPoint() tlxy.Point {
	c := g.Val.Coords()
	if len(c) != 2 {
		return tlxy.Point{}
	}
	return tlxy.Point{Lon: c[0], Lat: c[1]}
}

func (g *Point) X() float64 {
	if g.Val == nil {
		return 0
	}
	return g.Val.X()
}

func (g *Point) Y() float64 {
	if g.Val == nil {
		return 0
	}
	return g.Val.Y()
}

// NewPoint returns a Point from lon, lat
func NewPoint(lon, lat float64) Point {
	g := geom.NewPointFlat(geom.XY, geom.Coord{lon, lat})
	if g == nil {
		return Point{}
	}
	g.SetSRID(4326)
	return Point{GeometryOption: NewGeometryOption(g)}
}
