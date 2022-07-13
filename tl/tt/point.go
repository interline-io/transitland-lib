package tt

import (
	"database/sql/driver"
	"io"

	geom "github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/wkb"
	"github.com/twpayne/go-geom/encoding/wkbcommon"
)

// Point is an EWKB/SL encoded point
type Point struct {
	Valid bool
	geom.Point
}

// NewPoint returns a Point from lon, lat
func NewPoint(lon, lat float64) Point {
	g := geom.NewPointFlat(geom.XY, geom.Coord{lon, lat})
	if g == nil {
		return Point{}
	}
	g.SetSRID(4326)
	return Point{Point: *g, Valid: true}
}

// Value implements driver.Value
func (g Point) Value() (driver.Value, error) {
	if !g.Valid {
		return nil, nil
	}
	return wkbEncode(&g.Point)
}

// Scan implements Scanner
func (g *Point) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	b, ok := src.([]byte)
	if !ok {
		return wkb.ErrExpectedByteSlice{Value: src}
	}
	// Parse
	var p geom.T
	var err error
	p, err = wkbDecode(b)
	if err != nil {
		return err
	}
	p1, ok := p.(*geom.Point)
	if !ok {
		return wkbcommon.ErrUnexpectedType{Got: p1, Want: g}
	}
	g.Valid = true
	g.Point = *p1
	return nil
}

// String returns the GeoJSON representation
func (g Point) String() string {
	a, _ := g.MarshalJSON()
	return string(a)
}

// MarshalJSON implements the json.Marshaler interface
func (g *Point) MarshalJSON() ([]byte, error) {
	if !g.Valid {
		return []byte("null"), nil
	}
	return geojsonEncode(&g.Point)
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (g *Point) UnmarshalGQL(v interface{}) error {
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (g Point) MarshalGQL(w io.Writer) {
	b, _ := g.MarshalJSON()
	w.Write(b)
}
