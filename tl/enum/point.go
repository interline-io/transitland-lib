package enum

import (
	geom "github.com/twpayne/go-geom"
)

// Point is an EWKB/SL encoded point
type Point struct {
	Geometry[*geom.Point]
}

// NewPoint returns a Point from lon, lat
func NewPoint(lon, lat float64) Point {
	g := geom.NewPointFlat(geom.XY, geom.Coord{lon, lat})
	if g == nil {
		return Point{}
	}
	g.SetSRID(4326)
	pp := Point{}
	pp.Geometry.Val = g
	pp.Geometry.Valid = true
	return pp
}

func (g *Point) X() float64 {
	return g.Val.X()
}

func (g *Point) Y() float64 {
	return g.Val.Y()
}

func (g *Point) Coords() geom.Coord {
	return g.Val.Coords()
}
