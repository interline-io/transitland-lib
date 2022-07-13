package tt

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
type Geometry struct {
	Valid    bool
	Geometry geom.T
}

func (g *Geometry) Scan(src interface{}) error {
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
	g.Geometry = got
	g.Valid = true
	return nil
}

func (g Geometry) Value() (driver.Value, error) {
	if g.Geometry == nil || !g.Valid {
		return nil, nil
	}
	a, err := wkbEncode(g.Geometry)
	return a, err
}

// String returns the GeoJSON representation
func (g Geometry) String() string {
	a, _ := g.MarshalJSON()
	return string(a)
}

func (g *Geometry) MarshalJSON() ([]byte, error) {
	if !g.Valid {
		return []byte("null"), nil
	}
	return geojsonEncode(g.Geometry)
}

func (g *Geometry) UnmarshalGQL(v interface{}) error {
	return nil
}

func (g Geometry) MarshalGQL(w io.Writer) {
	b, _ := g.MarshalJSON()
	w.Write(b)
}

// Errors, helpers

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
