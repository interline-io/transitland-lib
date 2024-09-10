package tt

import (
	"database/sql/driver"
	"io"

	"github.com/interline-io/transitland-lib/tlxy"
	geom "github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/wkb"
	"github.com/twpayne/go-geom/encoding/wkbcommon"
)

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

func (g LineString) ToPoints() []tlxy.Point {
	var ret []tlxy.Point
	for _, c := range g.LineString.Coords() {
		ret = append(ret, tlxy.Point{Lon: c[0], Lat: c[1]})
	}
	return ret
}

func (g LineString) ToLineM() tlxy.LineM {
	var ret []tlxy.Point
	var ms []float64
	for _, c := range g.LineString.Coords() {
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

func (g LineString) Value() (driver.Value, error) {
	if !g.Valid {
		return nil, nil
	}
	return wkbEncode(&g.LineString)
}

func (g *LineString) Scan(src interface{}) error {
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
	p1, ok := p.(*geom.LineString)
	if !ok {
		return wkbcommon.ErrUnexpectedType{Got: p1, Want: p1}
	}
	g.Valid = true
	g.LineString = *p1
	return nil
}

func (g LineString) String() string {
	a, _ := g.MarshalJSON()
	return string(a)
}

func (g LineString) MarshalJSON() ([]byte, error) {
	if !g.Valid {
		return jsonNull(), nil
	}
	return geojsonEncode(&g.LineString)
}

func (g *LineString) UnmarshalGQL(v interface{}) error {
	return nil
}

func (g LineString) MarshalGQL(w io.Writer) {
	b, _ := g.MarshalJSON()
	w.Write(b)
}
