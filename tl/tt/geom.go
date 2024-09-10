package tt

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
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
	g.Geometry, g.Valid = nil, false
	if src == nil {
		return nil
	}
	b, ok := src.(string)
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

func (g Geometry) MarshalJSON() ([]byte, error) {
	if !g.Valid {
		return jsonNull(), nil
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
func wkbEncode(g geom.T) (string, error) {
	b := &bytes.Buffer{}
	if err := ewkb.Write(b, wkb.NDR, g); err != nil {
		return "", err
	}
	bb := b.Bytes()
	data := make([]byte, len(bb)*2)
	hex.Encode(data, bb)
	return string(data), nil
}

// wkbDecode tries to guess the encoding returned from the driver.
// When not wrapped in anything, postgis returns EWKB, and spatialite returns its internal blob format.
func wkbDecode(data string) (geom.T, error) {
	b := make([]byte, len(data)/2)
	hex.Decode(b, []byte(data))
	got, err := ewkb.Unmarshal(b)
	if err != nil {
		return nil, err
	}
	return got, nil
}

// geojsonEncode encodes a geometry into geojson.
func geojsonEncode(g geom.T) ([]byte, error) {
	if v, ok := g.(canEncodeGeojson); ok {
		return v.MarshalJSON()
	}
	b, err := geojson.Marshal(g)
	if err != nil {
		return jsonNull(), err
	}
	return b, nil
}

type canEncodeGeojson interface {
	MarshalJSON() ([]byte, error)
}

// geojsonEncode decodes geojson into a geometry.
func geojsonDecode[T any, PT *T](v any) (T, error) {
	var ret T
	var data []byte
	if a, ok := v.([]byte); ok {
		data = a
	} else if a, ok := v.(string); ok {
		data = []byte(a)
	} else {
		var err error
		data, err = json.Marshal(v)
		if err != nil {
			return ret, err
		}
	}
	var gg geom.T
	if err := geojson.Unmarshal(data, &gg); err != nil {
		return ret, nil
	}
	if a, ok := gg.(PT); ok && a != nil {
		ret = *a
	}
	return ret, nil
}
