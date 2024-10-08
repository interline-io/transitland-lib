package tt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"

	geom "github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
	"github.com/twpayne/go-geom/encoding/wkb"
	"github.com/twpayne/go-geom/encoding/wkbcommon"
)

// Polygon is an EWKB/SL encoded Polygon
type Polygon struct {
	Valid bool
	geom.Polygon
}

func (g Polygon) Value() (driver.Value, error) {
	if !g.Valid {
		return nil, nil
	}
	return wkbEncode(&g.Polygon)
}

func (g *Polygon) Scan(src interface{}) error {
	g.Valid = false
	if src == nil {
		return nil
	}
	b, ok := src.(string)
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

func (g Polygon) String() string {
	a, _ := g.MarshalJSON()
	return string(a)
}

func (g Polygon) MarshalJSON() ([]byte, error) {
	if !g.Valid {
		return jsonNull(), nil
	}
	return geojsonEncode(&g.Polygon)
}

func (g *Polygon) UnmarshalGQL(v interface{}) error {
	vb, err := json.Marshal(v)
	if err != nil {
		return errors.New("invalid geometry")
	}
	var x geom.T
	err = geojson.Unmarshal(vb, &x)
	if a, ok := x.(*geom.Polygon); err == nil && ok {
		g.Polygon = *a
		g.Valid = true
	} else {
		return errors.New("invalid geometry")
	}
	return nil
}

func (g Polygon) MarshalGQL(w io.Writer) {
	b, _ := g.MarshalJSON()
	w.Write(b)
}
