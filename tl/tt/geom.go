package tt

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"

	geom "github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/ewkb"
	"github.com/twpayne/go-geom/encoding/geojson"
	"github.com/twpayne/go-geom/encoding/wkb"
	"github.com/twpayne/go-geom/encoding/wkbcommon"
)

type Geometry struct {
	GeometryOption[geom.T]
}

func NewGeometry(v geom.T) Geometry {
	return Geometry{GeometryOption: NewGeometryOption(v)}
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

func wkbDecodeG[T any, PT *T](data string) (T, error) {
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
func geojsonDecode[T geom.T](v any) (T, error) {
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
		fmt.Println("??", err)
		return ret, nil
	}
	if a, ok := gg.(T); ok {
		return a, nil
	}
	return ret, nil
	// fmt.Printf("GG: %T PT: %T", gg, x)
	// if a, ok := gg.(T); ok && a != nil {
	// 	ret = *a
	// }
	// return ret, nil
}
