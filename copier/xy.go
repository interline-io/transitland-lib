package copier

import (
	"math"
)

// Simple XY geometry helper functions

var epsilon = 1e-6
var earthRadiusMetres float64 = 6371008

func deg2rad(v float64) float64 {
	return v * math.Pi / 180
}

// distanceHaversine returns the Haversine approximate spherical distance between two points.
func distanceHaversine(a, b [2]float64) float64 {
	lon1 := deg2rad(a[0])
	lat1 := deg2rad(a[1])
	lon2 := deg2rad(b[0])
	lat2 := deg2rad(b[1])
	dlat := lat2 - lat1
	dlon := lon2 - lon1
	d := math.Pow(math.Sin(dlat/2), 2) + math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(dlon/2), 2)
	c := 2 * math.Asin(math.Sqrt(d))
	return earthRadiusMetres * c
}

// lengthHaversine returns the Haversine approximate length of a line.
func lengthHaversine(line [][2]float64) float64 {
	length := 0.0
	for i := 1; i < len(line); i++ {
		length += distanceHaversine(line[i-1], line[i])
	}
	return length
}

// length2d returns the cartesian length of line
func length2d(line [][2]float64) float64 {
	length := 0.0
	for i := 1; i < len(line); i++ {
		length += distance2d(line[i-1], line[i])
	}
	return length
}

// distance2d returns the cartesian distance
func distance2d(a, b [2]float64) float64 {
	dx := a[0] - b[0]
	dy := a[1] - b[1]
	return math.Sqrt(dx*dx + dy*dy)
}

// segmentClosestPoint returns the point (and position) on AB closest to P.
func segmentClosestPoint(a, b, p [2]float64) ([2]float64, float64) {
	// check ends
	if distance2d(a, p) < epsilon {
		return a, 0.0
	}
	if distance2d(b, p) < epsilon {
		return b, 0.0
	}
	// get the projection of p onto ab
	r := ((p[0]-a[0])*(b[0]-a[0]) + (p[1]-a[1])*(b[1]-a[1])) / ((b[0]-a[0])*(b[0]-a[0]) + (b[1]-a[1])*(b[1]-a[1]))
	if r < 0 {
		return a, distance2d(a, p)
	} else if r > 1 {
		return b, distance2d(b, p)
	}
	// get coordinates
	ret := [2]float64{}
	ret[0] = a[0] + ((b[0] - a[0]) * r)
	ret[1] = a[1] + ((b[1] - a[1]) * r)
	return ret, distance2d(ret, p)
}

// lineClosestPoint returns the point (and position) on line closest to point.
func lineClosestPoint(line [][2]float64, point [2]float64) ([2]float64, float64) {
	position := 0.0
	length := length2d(line)
	if length == 0 {
		return point, position
	}
	segpos := 0.0
	mind := math.MaxFloat64
	minp := [2]float64{}
	start := line[0]
	for i := 1; i < len(line); i++ {
		end := line[i]
		segp, segd := segmentClosestPoint(start, end, point)
		if segd < mind {
			mind = segd
			minp = segp
			position = segpos + distance2d(start, minp)
			if segd == 0 {
				break
			}
		}
		segpos += distance2d(start, end)
		start = end
	}
	return minp, position / length
}

// linePositionsFallback returns the relative position along the line for each point.
func linePositionsFallback(line [][2]float64) []float64 {
	ret := make([]float64, len(line))
	length := length2d(line)
	position := 0.0
	ret[0] = 0.0
	for i := 1; i < len(line); i++ {
		position += distance2d(line[i], line[i-1])
		ret[i] = position / length
	}
	return ret
}

// linePositions finds the relative position of the closest point along the line for each point.
func linePositions(line [][2]float64, points [][2]float64) []float64 {
	positions := make([]float64, len(points))
	for i, p := range points {
		_, d := lineClosestPoint(line, p)
		positions[i] = d
	}
	return positions
}
