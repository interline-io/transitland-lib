package tl

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"io"

	geom "github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/ewkb"
	"github.com/twpayne/go-geom/encoding/geojson"
	"github.com/twpayne/go-geom/encoding/wkb"
	"github.com/twpayne/go-geom/encoding/wkbcommon"
)

////////////////////////

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

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (g *Point) UnmarshalGQL(v interface{}) error {
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (g Point) MarshalGQL(w io.Writer) {
	if !g.Valid {
		w.Write([]byte("null"))
		return
	}
	x, err := geojson.Marshal(&g.Point)
	if err != nil {
		panic(err)
	}
	w.Write(x)
}

/////////////////////

// LineString is an EWKB/SL encoded LineString
type LineString struct {
	Valid bool
	geom.LineString
}

// NewLineStringFromFlatCoords returns a new LineString from flat (3) coordinates
func NewLineStringFromFlatCoords(coords []float64) LineString {
	g := geom.NewLineStringFlat(geom.XYM, coords)
	if g == nil {
		return LineString{}
	}
	g.SetSRID(4326)
	return LineString{LineString: *g, Valid: true}
}

// Value implements driver.Value
func (g LineString) Value() (driver.Value, error) {
	if !g.Valid {
		return nil, nil
	}
	return wkbEncode(&g.LineString)
}

// Scan implements Scanner
func (g *LineString) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	b, ok := src.([]byte)
	if !ok {
		return wkb.ErrExpectedByteSlice{Value: src}
	}
	var p geom.T
	var err error
	p, err = wkbDecode(b)
	if err != nil {
		return err
	}
	p1, ok := p.(*geom.LineString)
	if !ok {
		return wkbcommon.ErrUnexpectedType{Got: p1, Want: p1}
	}
	g.Valid = true
	g.LineString = *p1
	return nil
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (g *LineString) UnmarshalGQL(v interface{}) error {
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (g LineString) MarshalGQL(w io.Writer) {
	if !g.Valid {
		w.Write([]byte("null"))
		return
	}
	x, err := geojson.Marshal(&g.LineString)
	if err != nil {
		panic(err)
	}
	w.Write(x)
}

///////////////////////

// Polygon is an EWKB/SL encoded Polygon
type Polygon struct {
	Valid bool
	geom.Polygon
}

// Value implements driver.Value
func (g Polygon) Value() (driver.Value, error) {
	if !g.Valid {
		return nil, nil
	}
	return wkbEncode(&g.Polygon)
}

// Scan implements Scanner
func (g *Polygon) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	b, ok := src.([]byte)
	if !ok {
		return wkb.ErrExpectedByteSlice{Value: src}
	}
	var p geom.T
	var err error
	p, err = wkbDecode(b)
	if err != nil {
		return err
	}
	p1, ok := p.(*geom.Polygon)
	if !ok {
		return wkbcommon.ErrUnexpectedType{Got: p1, Want: p1}
	}
	g.Valid = true
	g.Polygon = *p1
	return nil
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (g *Polygon) UnmarshalGQL(v interface{}) error {
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (g Polygon) MarshalGQL(w io.Writer) {
	if !g.Valid {
		w.Write([]byte("null"))
		return
	}
	x, err := geojson.Marshal(&g.Polygon)
	if err != nil {
		panic(err)
	}
	w.Write(x)
}

/////////// helpers

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
	var data []byte
	data = make([]byte, len(b)/2)
	hex.Decode(data, b)
	got, err := ewkb.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	return got, nil
}
