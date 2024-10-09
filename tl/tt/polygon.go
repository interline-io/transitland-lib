package tt

import (
	geom "github.com/twpayne/go-geom"
)

// Polygon is an EWKB/SL encoded Polygon
type Polygon struct {
	GeometryOption[*geom.Polygon]
}

func NewPolygon(v *geom.Polygon) Polygon {
	return Polygon{GeometryOption: NewGeometryOption(v)}
}
