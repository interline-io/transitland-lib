package gotransit

import (
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"errors"

	geom "github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/ewkb"
	"github.com/twpayne/go-geom/encoding/wkb"
	"github.com/twpayne/go-geom/encoding/wkbcommon"
)

////////////

// Geometry holds any EWKB encoded/decoded Geometry
type Geometry struct {
	Geometry geom.T
	Invalid  bool
}

// Value implements driver.Value
func (g *Geometry) Value() (driver.Value, error) {
	if g.Invalid {
		return nil, nil
	}
	return wkbEncode(g.Geometry)
}

// Scan implements Scanner
func (g *Geometry) Scan(src interface{}) error {
	g.Invalid = true
	if src == nil {
		return nil
	}
	b, ok := src.([]byte)
	if !ok {
		return wkb.ErrExpectedByteSlice{Value: src}
	}
	var p geom.T
	var err error
	p, err = autoDecode(b)
	if err != nil {
		return err
	}
	g.Invalid = false
	g.Geometry = p
	return nil
}

////////////////////////

// Point is an EWKB/SL encoded point
type Point struct {
	geom.Point
	Invalid bool
}

// NewPoint returns a Point from lon, lat
func NewPoint(lon, lat float64) *Point {
	g := geom.NewPointFlat(geom.XY, geom.Coord{lon, lat})
	g.SetSRID(4326)
	return &Point{Point: *g}
}

// Value implements driver.Value
func (g *Point) Value() (driver.Value, error) {
	if g.Invalid {
		return nil, nil
	}
	return wkbEncode(&g.Point)
}

// Scan implements Scanner
func (g *Point) Scan(src interface{}) error {
	g.Invalid = true
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
	p, err = autoDecode(b)
	if err != nil {
		return err
	}
	p1, ok := p.(*geom.Point)
	if !ok {
		return wkbcommon.ErrUnexpectedType{Got: p1, Want: g}
	}
	g.Invalid = false
	g.Point = *p1
	return nil
}

/////////////////////

// LineString is an EWKB/SL encoded LineString
type LineString struct {
	geom.LineString
	Invalid bool
}

// NewLineStringFromFlatCoords returns a new LineString from flat (3) coordinates
func NewLineStringFromFlatCoords(coords []float64) *LineString {
	geom := geom.NewLineStringFlat(geom.XYM, coords)
	geom.SetSRID(4326)
	return &LineString{LineString: *geom}
}

// Value implements driver.Value
func (g *LineString) Value() (driver.Value, error) {
	if g.Invalid {
		return nil, nil
	}
	return wkbEncode(&g.LineString)
}

// Scan implements Scanner
func (g *LineString) Scan(src interface{}) error {
	g.Invalid = true
	if src == nil {
		return nil
	}
	b, ok := src.([]byte)
	if !ok {
		return wkb.ErrExpectedByteSlice{Value: src}
	}
	var p geom.T
	var err error
	p, err = autoDecode(b)
	if err != nil {
		return err
	}
	p1, ok := p.(*geom.LineString)
	if !ok {
		return wkbcommon.ErrUnexpectedType{Got: p1, Want: p1}
	}
	g.Invalid = false
	g.LineString = *p1
	return nil
}

/////////// helpers

// wkbEncode encodes a geometry into WKB.
// We use WKB instead of EWKB because GeomFromEWKB is inconsistent between postgis and spatialite (bytes vs string).
func wkbEncode(g geom.T) ([]byte, error) {
	b := &bytes.Buffer{}
	if err := wkb.Write(b, wkb.NDR, g); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// autoDecode tries to guess the encoding returned from the driver.
// When not wrapped in anything, postgis returns EWKB, and spatialite returns its internal blob format.
func autoDecode(b []byte) (geom.T, error) {
	if len(b) > 1 && b[0] == byte(0) && b[1] == byte(1) && b[len(b)-1] == byte(254) {
		return slDecode(b)
	}
	var data []byte
	data = make([]byte, len(b)/2)
	hex.Decode(data, b)
	got, err := ewkb.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	return got, nil
}

/////////// spatialite decode...

const (
	ewkbZ    = 0x80000000
	ewkbM    = 0x40000000
	ewkbSRID = 0x20000000
	slStart  = byte(0)
	slEnd    = byte(254)
	slMbrEnd = byte(124)
)

type slHeader struct {
	Start      byte
	Endianness byte
	SRID       uint32
	Bounds     [4]float64
	End        byte
	ClassType  uint32
}

// slDecode reads SpatiaLite binary blobs.
func slDecode(r []byte) (geom.T, error) {
	if len(r) < 44 {
		return nil, errors.New("no geometry data")
	}
	// Parse header
	var byteOrder binary.ByteOrder
	if r[1] == byte(0) {
		byteOrder = binary.BigEndian
	} else if r[1] == byte(1) {
		byteOrder = binary.LittleEndian
	}
	header := slHeader{}
	if err := binary.Read(bytes.NewReader(r), byteOrder, &header); err != nil {
		return nil, err
	}
	// Parse data
	data := bytes.NewReader(r[43 : len(r)-1])
	// Make geom
	var err error
	var g geom.T
	var layout geom.Layout
	switch header.ClassType / 1000 {
	case 0:
		layout = geom.XY
	case 1:
		layout = geom.XYZ
	case 2:
		layout = geom.XYM
	case 3:
		layout = geom.XYZM
	}
	switch header.ClassType % 1000 {
	case 1:
		coords := make([]float64, layout.Stride())
		if err := binary.Read(data, byteOrder, &coords); err != nil {
			return nil, err
		}
		g = geom.NewPointFlat(layout, coords)
	case 2:
		count := uint32(0)
		binary.Read(data, byteOrder, &count)
		coords := make([]float64, int(count)*layout.Stride())
		if err := binary.Read(data, byteOrder, &coords); err != nil {
			return nil, err
		}
		g = geom.NewLineStringFlat(layout, coords)
	case 3:
		numrings := uint32(0) // rings
		binary.Read(data, byteOrder, &numrings)
		poly := geom.NewPolygon(layout)
		for i := 0; i < int(numrings); i++ {
			count := uint32(0)
			binary.Read(data, byteOrder, &count)
			coords := make([]float64, int(count)*layout.Stride())
			if err := binary.Read(data, byteOrder, &coords); err != nil {
				return nil, err
			}
			poly.Push(geom.NewLinearRingFlat(layout, coords))
		}
		g = poly
	default:
		return nil, errors.New("unknown geometry type")
	}
	return g, err
}

// slEncode creates SpatiaLite binary representation
// https://www.gaia-gis.it/gaia-sins/BLOB-Geometry.html
func slEncode(g geom.T) ([]byte, error) {
	gtype := uint32(0)
	layout := g.Layout()
	switch g.(type) {
	case *geom.Point:
		gtype = 1
	case *geom.LineString:
		gtype = 2
	case *geom.Polygon:
		gtype = 3
	default:
		return nil, errors.New("unknown geometry type")
	}
	switch layout {
	case geom.XY:
		gtype += 0
	case geom.XYZ:
		gtype += 1000
	case geom.XYM:
		gtype += 2000
	case geom.XYZM:
		gtype += 3000
	}
	byteOrder := binary.LittleEndian
	bounds := g.Bounds()
	header := slHeader{}
	header.Start = slStart
	header.Endianness = byte(1)
	header.SRID = uint32(4326) // uint32(g.SRID()) - g.SetSRID does not exist
	header.Bounds = [4]float64{bounds.Min(0), bounds.Min(1), bounds.Max(0), bounds.Max(1)}
	header.End = slMbrEnd
	header.ClassType = gtype
	w := bytes.NewBuffer(nil)
	binary.Write(w, byteOrder, header)
	switch gtype % 1000 {
	case 1:
		coords := g.FlatCoords()
		binary.Write(w, byteOrder, coords)
	case 2:
		coords := g.FlatCoords()
		binary.Write(w, byteOrder, uint32(len(coords)/layout.Stride()))
		binary.Write(w, byteOrder, coords)
	case 3:
		q := g.(*geom.Polygon)
		numrings := q.NumLinearRings()
		binary.Write(w, byteOrder, uint32(numrings))
		for i := 0; i < numrings; i++ {
			coords := q.LinearRing(i).FlatCoords()
			binary.Write(w, byteOrder, uint32(len(coords)/layout.Stride()))
			binary.Write(w, byteOrder, coords)
		}
	default:
		return nil, errors.New("unknown geometry type")
	}
	binary.Write(w, byteOrder, slEnd)
	return w.Bytes(), nil
}
