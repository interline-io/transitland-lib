package tt

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/interline-io/transitland-lib/tlxy"
	geom "github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/ewkb"
	"github.com/twpayne/go-geom/encoding/geojson"
	"github.com/twpayne/go-geom/encoding/wkb"
	"github.com/twpayne/go-geom/encoding/wkbcommon"
)

// Point is an EWKB/SL encoded point
type GeometryOption[T geom.T] struct {
	Val   T
	Valid bool
}

func NewGeometryOption[T geom.T](v T) GeometryOption[T] {
	return GeometryOption[T]{Val: v, Valid: true}
}

func (g GeometryOption[T]) FlatCoords() []float64 {
	if !g.Valid {
		return nil
	}
	return g.Val.FlatCoords()
}

func (g GeometryOption[T]) Stride() int {
	if !g.Valid {
		return 0
	}
	return g.Val.Stride()
}

func (g GeometryOption[T]) Value() (driver.Value, error) {
	if !g.Valid {
		return nil, nil
	}
	a, b := wkbEncode(g.Val)
	return string(a), b
}

func (g *GeometryOption[T]) Scan(src interface{}) error {
	// Parse as WKB, GeoJSON or map[string]any
	g.Valid = false
	if src == nil {
		return nil
	}
	var err error
	switch v := src.(type) {
	case string:
		g.Val, err = geomAutoDecode[T]([]byte(v))
	case []byte:
		g.Val, err = geomAutoDecode[T]([]byte(v))
	case map[string]any:
		jj, jjErr := json.Marshal(v)
		if jjErr != nil {
			return jjErr
		}
		g.Val, err = geojsonDecode[T](jj)
	}
	g.Valid = (err == nil)
	return err
}

func (g *GeometryOption[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	return g.Scan(data)
}

func (g GeometryOption[T]) MarshalJSON() ([]byte, error) {
	if !g.Valid {
		return jsonNull(), nil
	}
	return geojsonEncode(g.Val)
}

//////////

type Geometry struct {
	GeometryOption[geom.T]
}

func NewGeometry(v geom.T) Geometry {
	return Geometry{GeometryOption: NewGeometryOption(v)}
}

//////////

// Point is an EWKB/SL encoded point
type Point struct {
	GeometryOption[*geom.Point]
}

func (g *Point) ToPoint() tlxy.Point {
	if !g.Valid {
		return tlxy.Point{}
	}
	c := g.Val.Coords()
	if len(c) != 2 {
		return tlxy.Point{}
	}
	return tlxy.Point{Lon: c[0], Lat: c[1]}
}

func (g *Point) X() float64 {
	if !g.Valid {
		return 0
	}
	return g.Val.X()
}

func (g *Point) Y() float64 {
	if !g.Valid {
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

//////////

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
	if !g.Valid {
		return ret
	}
	for _, c := range g.Val.Coords() {
		ret = append(ret, tlxy.Point{Lon: c[0], Lat: c[1]})
	}
	return ret
}

func (g LineString) ToLineM() tlxy.LineM {
	var ret []tlxy.Point
	if !g.Valid {
		return tlxy.LineM{}
	}
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

//////////

// Polygon is an EWKB/SL encoded Polygon
type Polygon struct {
	GeometryOption[*geom.Polygon]
}

func NewPolygon(v *geom.Polygon) Polygon {
	return Polygon{GeometryOption: NewGeometryOption(v)}
}

//////////

// MultiPolygon is an EWKB/SL encoded MultiPolygon
type MultiPolygon struct {
	GeometryOption[*geom.MultiPolygon]
}

func NewMultiPolygon(v *geom.MultiPolygon) MultiPolygon {
	return MultiPolygon{GeometryOption: NewGeometryOption(v)}
}

//////////

// Errors, helpers

func geomAutoDecode[T any](data []byte) (T, error) {
	if len(data) > 0 && data[0] == '{' {
		return geojsonDecode[T](data)
	}
	return wkbDecode[T](data)
}

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

func wkbDecode[T any](data []byte) (T, error) {
	var ret T
	b := make([]byte, len(data)/2)
	hex.Decode(b, []byte(data))
	got, err := ewkb.Unmarshal(b)
	if err != nil {
		return ret, err
	}
	ret, ok := got.(T)
	if !ok {
		return ret, wkbcommon.ErrUnexpectedType{Got: got, Want: ret}
	}
	return ret, nil
}

// geojsonEncode encodes a geometry into geojson.
func geojsonEncode(g geom.T) ([]byte, error) {
	b, err := geojson.Marshal(g)
	if err != nil {
		return jsonNull(), err
	}
	return b, nil
}

func geojsonDecode[T any](data []byte) (T, error) {
	var ok bool
	var gg geom.T
	var v T
	if err := geojson.Unmarshal(data, &gg); err != nil {
		return v, err
	}
	v, ok = gg.(T)
	if !ok {
		return v, fmt.Errorf("could not decode into %T", v)
	}
	return v, nil
}
