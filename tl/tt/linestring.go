package tt

import (
	"io"

	geom "github.com/twpayne/go-geom"
)

/////////////////////

// LineString is an EWKB/SL encoded LineString
type LineString struct {
	Geometry[*geom.LineString]
}

// NewLineStringFromFlatCoords returns a new LineString from flat (3) coordinates
func NewLineStringFromFlatCoords(coords []float64) LineString {
	g := geom.NewLineStringFlat(geom.XYM, coords)
	if g == nil {
		return LineString{}
	}
	g.SetSRID(4326)
	p := LineString{}
	p.Val = g
	p.Valid = (g != nil)
	return p
}

func NewLineString(g *geom.LineString) LineString {
	g.SetSRID(4326)
	p := LineString{}
	p.Val = g
	p.Valid = (g != nil)
	return p
}

func (g *LineString) Length() float64 {
	return g.Val.Length()
}

func (g *LineString) Coords() []geom.Coord {
	return g.Val.Coords()
}

func (g *LineString) NumCoords() int {
	return g.Val.NumCoords()
}

// Needed for gqlgen - issue with generics
func (r *LineString) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

func (r LineString) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
