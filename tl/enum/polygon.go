package enum

import (
	geom "github.com/twpayne/go-geom"
)

// Polygon is an EWKB/SL encoded Polygon
type Polygon struct {
	Geometry[*geom.Polygon]
}

func NewPolygon(g *geom.Polygon) Polygon {
	g.SetSRID(4326)
	p := Polygon{}
	p.Val = g
	p.Valid = (g != nil)
	return p
}
