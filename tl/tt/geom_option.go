package tt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"

	geom "github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
	"github.com/twpayne/go-geom/encoding/wkb"
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
	return g.Val.FlatCoords()
}

func (g GeometryOption[T]) Stride() int {
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
	g.Valid = false
	if src == nil {
		return nil
	}
	b, ok := src.(string)
	if !ok {
		return wkb.ErrExpectedByteSlice{Value: src}
	}
	var err error
	g.Val, err = wkbDecodeG[T](b)
	g.Valid = (err == nil)
	if err != nil {
		return err
	}
	return nil
}

func (g *GeometryOption[T]) UnmarshalJSON(data []byte) error {
	var x geom.T = g.Val
	if err := geojson.Unmarshal(data, &x); err != nil {
		return err
	}
	var ok bool
	g.Val, ok = x.(T)
	if !ok {
		return errors.New("could not convert geometry")
	}
	return nil
}

func (g GeometryOption[T]) MarshalJSON() ([]byte, error) {
	if !g.Valid {
		return jsonNull(), nil
	}
	return geojsonEncode(g.Val)
}

func (g *GeometryOption[T]) UnmarshalGQL(v interface{}) error {
	jj, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return g.UnmarshalJSON(jj)
}

func (g GeometryOption[T]) MarshalGQL(w io.Writer) {
	b, _ := g.MarshalJSON()
	w.Write(b)
}
