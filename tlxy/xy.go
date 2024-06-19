package tlxy

import (
	"math"
	"strconv"
	"strings"
)

// TODO: Replace most of this with go-geom functions. I understand things better than when I originally wrote this :)

type Point struct {
	Lon float64
	Lat float64
}

// Simple XY geometry helper functions

var epsilon = 1e-6
var earthRadiusMetres float64 = 6371008

func deg2rad(v float64) float64 {
	return v * math.Pi / 180
}

// DistanceHaversinePoint .
func DistanceHaversinePoint(a, b []float64) float64 {
	if len(a) < 2 || len(b) < 2 {
		return 0.0
	}
	return DistanceHaversine(a[0], a[1], b[0], b[1])
}

// DistanceHaversine .
func DistanceHaversine(lon1, lat1, lon2, lat2 float64) float64 {
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

// LengthHaversine returns the Haversine approximate length of a line.
func LengthHaversine(line []Point) float64 {
	length := 0.0
	for i := 1; i < len(line); i++ {
		length += DistanceHaversine(line[i-1].Lon, line[i-1].Lat, line[i].Lon, line[i].Lat)
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

// Distance2d returns the cartesian distance
func Distance2d(a, b Point) float64 {
	dx := a.Lon - b.Lon
	dy := a.Lat - b.Lat
	return math.Sqrt(dx*dx + dy*dy)
}

// SegmentClosestPoint returns the point (and position) on AB closest to P.
func SegmentClosestPoint(a, b, p Point) (Point, float64) {
	// check ends
	if Distance2d(a, p) < epsilon {
		return a, 0.0
	}
	if Distance2d(b, p) < epsilon {
		return b, 0.0
	}
	// get the projection of p onto ab
	r := ((p.Lon-a.Lon)*(b.Lon-a.Lon) + (p.Lat-a.Lat)*(b.Lat-a.Lat)) / ((b.Lon-a.Lon)*(b.Lon-a.Lon) + (b.Lat-a.Lat)*(b.Lat-a.Lat))
	if r < 0 {
		return a, Distance2d(a, p)
	} else if r > 1 {
		return b, Distance2d(b, p)
	}
	// get coordinates
	ret := Point{}
	ret.Lon = a.Lon + ((b.Lon - a.Lon) * r)
	ret.Lat = a.Lat + ((b.Lat - a.Lat) * r)
	return ret, Distance2d(ret, p)
}

// LineClosestPoint returns the point (and position) on line closest to point.
// Based on go-geom DistanceFromPointToLineString
func LineClosestPoint(line []Point, point Point) (Point, float64) {
	position := 0.0
	length := Length2d(line)
	if length == 0 {
		return point, position
	}
	segpos := 0.0
	mind := math.MaxFloat64
	minp := Point{}
	start := line[0]
	for i := 1; i < len(line); i++ {
		end := line[i]
		segp, segd := SegmentClosestPoint(start, end, point)
		if segd < mind {
			mind = segd
			minp = segp
			position = segpos + Distance2d(start, minp)
			if segd == 0 {
				break
			}
		}
		segpos += Distance2d(start, end)
		start = end
	}
	return minp, position / length
}

// LinePositionsFallback returns the relative position along the line for each point.
func LinePositionsFallback(line []Point) []float64 {
	ret := make([]float64, len(line))
	length := Length2d(line)
	position := 0.0
	ret[0] = 0.0
	for i := 1; i < len(line); i++ {
		position += Distance2d(line[i], line[i-1])
		ret[i] = position / length
	}
	return ret
}

// LinePositions finds the relative position of the closest point along the line for each point.
func LinePositions(line []Point, points []Point) []float64 {
	positions := make([]float64, len(points))
	for i, p := range points {
		_, d := LineClosestPoint(line, p)
		positions[i] = d
	}
	return positions
}

func PointSliceContains(a []Point, b []Point) bool {
	if len(a) > len(b) {
		return false
	}
	for i := range b {
		if pointSliceStarts(a, b[i:]) {
			return true
		}
	}
	return false
}

func PointSliceEqual(a []Point, b []Point) bool {
	if len(b) != len(a) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func pointSliceStarts(a []Point, b []Point) bool {
	if len(b) < len(a) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type BoundingBox struct {
	MinLon float64 `json:"min_lon"`
	MinLat float64 `json:"min_lat"`
	MaxLon float64 `json:"max_lon"`
	MaxLat float64 `json:"max_lat"`
}

func (v *BoundingBox) Contains(pt Point) bool {
	if pt.Lon >= v.MinLon && pt.Lon <= v.MaxLon && pt.Lat >= v.MinLat && pt.Lat <= v.MaxLat {
		return true
	}
	return false
}

func ParseBbox(v string) (BoundingBox, error) {
	r := BoundingBox{}
	if s := strings.Split(v, ","); len(s) == 4 {
		r.MinLon, _ = strconv.ParseFloat(s[0], 64)
		r.MinLat, _ = strconv.ParseFloat(s[1], 64)
		r.MaxLon, _ = strconv.ParseFloat(s[2], 64)
		r.MaxLat, _ = strconv.ParseFloat(s[3], 64)
	}
	return r, nil
}
