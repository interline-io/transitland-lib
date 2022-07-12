package tt

import (
	"io"

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

// Needed for gqlgen - issue with generics
func (r *Polygon) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

func (r Polygon) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
