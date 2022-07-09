package enum

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"io"

	geom "github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/ewkb"
	"github.com/twpayne/go-geom/encoding/geojson"
	"github.com/twpayne/go-geom/encoding/wkb"
)

// Geometry is an EWKB/GeoJSON wrapper for arbitary geometry.
type Geometry[T geom.T] struct {
	Val   T
	Valid bool
}

func (g *Geometry[T]) FlatCoords() []float64 {
	return g.Val.FlatCoords()
}

func (g *Geometry[T]) Stride() int {
	return g.Val.Stride()
}

// Scan implements the Scanner interface
func (g *Geometry[T]) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	b, ok := src.([]byte)
	if !ok {
		return nil
	}
	got, err := wkbDecode(b)
	if err != nil {
		return err
	}
	g.Val, g.Valid = got.(T)
	return nil
}

// Value implements driver.Value
func (g Geometry[T]) Value() (driver.Value, error) {
	if !g.Valid {
		return nil, nil
	}
	a, err := wkbEncode(g.Val)
	return a, err
}

// String returns the GeoJSON representation
func (g Geometry[T]) String() string {
	a, _ := g.MarshalJSON()
	return string(a)
}

// MarshalJSON implements the json.Marshaler interface
func (g *Geometry[T]) MarshalJSON() ([]byte, error) {
	if !g.Valid {
		return []byte("null"), nil
	}
	return geojsonEncode(g.Val)
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (g *Geometry[T]) UnmarshalGQL(v interface{}) error {
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (g Geometry[T]) MarshalGQL(w io.Writer) {
	b, _ := g.MarshalJSON()
	w.Write(b)
}

// helpers

// wkbEncode encodes a geometry into EWKB.
func wkbEncode(g geom.T) ([]byte, error) {
	b := &bytes.Buffer{}
	if err := ewkb.Write(b, wkb.NDR, g); err != nil {
		return nil, err
	}
	bb := b.Bytes()
	data := make([]byte, len(bb)*2)
	hex.Encode(data, bb)
	return data, nil
}

// wkbDecode tries to guess the encoding returned from the driver.
// When not wrapped in anything, postgis returns EWKB, and spatialite returns its internal blob format.
func wkbDecode(b []byte) (geom.T, error) {
	data := make([]byte, len(b)/2)
	hex.Decode(data, b)
	got, err := ewkb.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	return got, nil
}

// geojsonEncode encodes a geometry into geojson.
func geojsonEncode(g geom.T) ([]byte, error) {
	b, err := geojson.Marshal(g)
	if err != nil {
		return []byte("null"), err
	}
	return b, nil
}
