package tlxy

import "math"

func Cut(from Point, to Point, line []Point) []Point {
	startPoint := Point{}
	startNear := 1_000_000.0
	startIdx := 0
	if len(line) < 2 {
		return nil
	}
	for i := 0; i < len(line)-1; i += 1 {
		if cp, d := SegmentClosestPoint(line[i], line[i+1], from); d < startNear {
			startIdx = i
			startNear = d
			startPoint = cp
		}
	}
	endPoint := Point{}
	endNear := 1_000_000.0
	endIdx := 0
	for i := startIdx; i < len(line)-1; i += 1 {
		if cp, d := SegmentClosestPoint(line[i], line[i+1], to); d < endNear {
			endIdx = i
			endNear = d
			endPoint = cp
		}
	}
	var coords []Point
	coords = append(coords, startPoint)
	for i := startIdx + 1; i <= endIdx; i++ {
		coords = append(coords, line[i])
	}
	coords = append(coords, endPoint)
	return coords
}

// SegmentClosestPoint returns the point (and position) on AB closest to P.
func SegmentClosestPoint(a, b, p Point) (Point, float64) {
	// ported from https://stackoverflow.com/questions/849211/shortest-distance-between-a-point-and-a-line-segment
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
	// accurate enough for small distances
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
