// Package tlxy provides simple XY geometry helper functions; these are approximate and designed for our specific use cases.
package tlxy

import (
	"math"
)

// GeomCache is an interface for returning stop and shape points.
type GeomCache interface {
	GetStop(string) Point
	GetShape(eid string) []Point
	GetShapeInfo(eid string) (ShapeInfo, bool)
}

// ShapeInfo contains cached shape geometry and metadata.
type ShapeInfo struct {
	encodedLine           []byte
	DistLength            float64
	Generated             bool
	MaxSegmentLength      float64
	FirstPointMaxDistance float64
}

// NewShapeInfo creates a new ShapeInfo with encoded polyline.
func NewShapeInfo(line []Point, distLength float64, generated bool, maxSegmentLength, firstPointMaxDistance float64) ShapeInfo {
	return ShapeInfo{
		encodedLine:           EncodePolyline(line),
		DistLength:            distLength,
		Generated:             generated,
		MaxSegmentLength:      maxSegmentLength,
		FirstPointMaxDistance: firstPointMaxDistance,
	}
}

// Line decodes and returns the shape geometry as a slice of points.
func (s ShapeInfo) Line() []Point {
	if len(s.encodedLine) == 0 {
		return nil
	}
	pts, _ := DecodePolyline(s.encodedLine)
	return pts
}

// Simple XY geometry helper functions.

const latMeter = 111195.0662709627
const epsilon = 1e-6
const earthRadiusMetres float64 = 6371008

func deg2rad(v float64) float64 {
	return v * math.Pi / 180
}

func distanceHaversineCoords(lon1, lat1, lon2, lat2 float64) float64 {
	lon1 = deg2rad(lon1)
	lat1 = deg2rad(lat1)
	lon2 = deg2rad(lon2)
	lat2 = deg2rad(lat2)
	dlat := lat2 - lat1
	dlon := lon2 - lon1
	d := math.Pow(math.Sin(dlat/2), 2) + math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(dlon/2), 2)
	c := 2 * math.Asin(math.Sqrt(d))
	return earthRadiusMetres * c
}

// Point distances

// Distance2d returns the cartesian distance between two points.
func Distance2d(p1, p2 Point) float64 {
	a := p2.Lon - p1.Lon
	b := p2.Lat - p1.Lat
	return math.Sqrt(a*a + b*b)
}

// DistanceHaversine returns the Haversine distance between two points.
func DistanceHaversine(a, b Point) float64 {
	return distanceHaversineCoords(a.Lon, a.Lat, b.Lon, b.Lat)
}

// Approximate point distances

// ApproxLatMeters returns the approximate number of meters per degree of latitude at the given point.
func ApproxLonMeters(p Point) float64 {
	return distanceHaversineCoords(p.Lon, p.Lat, p.Lon+0.01, p.Lat) / 0.01
}

// ApproxDistance returns the approximate distance in meters between two points.
func ApproxDistance(lonCheck float64, p Point, s Point) float64 {
	dx := (p.Lon - s.Lon) * (lonCheck)
	dy := (p.Lat - s.Lat) * (latMeter)
	return math.Sqrt((dx * dx) + (dy * dy))
}

// Approx is a helper for approximate distance calculations.
type Approx struct {
	lonMeter float64
}

// NewApprox returns a new Approx distance helper.
func NewApprox(p Point) Approx {
	return Approx{
		lonMeter: ApproxLonMeters(p),
	}
}

// LonMeters returns the approximate number of meters per degree of longitude.
func (a *Approx) LonMeters() float64 {
	return a.lonMeter
}

// LatMeters returns the approximate number of meters per degree of latitude.
func (a *Approx) LatMeters() float64 {
	return latMeter
}

// ApproxDistance returns the approximate distance in meters between two points.
func (a *Approx) ApproxDistance(p Point, s Point) float64 {
	if a.lonMeter == 0 {
		a.lonMeter = ApproxLonMeters(p)
	}
	dx := (p.Lon - s.Lon) * (a.lonMeter)
	dy := (p.Lat - s.Lat) * (latMeter)
	return math.Sqrt((dx * dx) + (dy * dy))
}

// Line lengths

// LengthHaversine returns the Haversine approximate length of a line.
func LengthHaversine(line []Point) float64 {
	length := 0.0
	for i := 1; i < len(line); i++ {
		length += DistanceHaversine(line[i-1], line[i])
	}
	return length
}

// Length2d returns the cartesian length of line
func Length2d(line []Point) float64 {
	length := 0.0
	for i := 1; i < len(line); i++ {
		length += Distance2d(line[i-1], line[i])
	}
	return length
}

// Distance2dLength returns the cartesian length of line
func Distance2dLength(line []Point) float64 {
	return Length2d(line)
}
