package tlxy

import (
	"math"
)

// SegmentClosestPoint calculates the closest point on a line segment AB to a given point P,
// and returns both the closest point and the distance to it.
//
// Given three points:
//   - a: Start point of line segment
//   - b: End point of line segment
//   - p: Point to find closest position to
//
// Returns:
//   - Point: The closest point on segment AB to point P
//   - float64: The distance between point P and the closest point
//
// The algorithm first checks if P is closest to either endpoint.
// If not, it projects P onto line AB and clamps the result to the segment.
// Distance calculation assumes a simplified 2D Euclidean space suitable for small distances.
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

// LineClosestPoint finds the nearest point on a line (represented as a slice of Points) to a given point.
// Based on go-geom DistanceFromPointToLineString
// It returns three values:
// - The closest point on the line
// - The index of the line segment containing the closest point (0-based)
// - The normalized position along the line (0.0 to 1.0) where the closest point lies
//
// The line must contain at least 2 points. If the line has length 0 (single point or empty),
// it returns the input point, index 0, and position 0.
//
// The calculation uses Haversine distance for geographic coordinates.
//
// Parameters:
//   - line: A slice of Points representing the line segments
//   - point: The reference Point to find the closest position to
//
// Returns:
//   - Point: The closest point on the line
//   - int: Index of the segment containing the closest point
//   - float64: Normalized position along the line (0.0 to 1.0)
func LineClosestPoint(line []Point, point Point) (Point, int, float64) {
	position := 0.0
	length := LengthHaversine(line)
	if length == 0 {
		return point, 0, position
	}
	segpos := 0.0
	minidx := 0
	mind := math.MaxFloat64
	minp := Point{}
	for i := 1; i < len(line); i++ {
		start := line[i-1]
		end := line[i]
		segp, segd := SegmentClosestPoint(start, end, point)
		if segd < mind {
			mind = segd
			minp = segp
			minidx = i
			position = segpos + DistanceHaversine(start, minp)
			if segd == 0 {
				break
			}
		}
		segpos += DistanceHaversine(start, end)
		start = end
	}
	return minp, minidx, position / length
}

// CutBetweenPoints extracts a portion of a line between two points by finding the closest segments.
// It takes a line (represented as a slice of Points), and two points (from and to) as input.
// The function finds the closest segments to both input points and returns a new line that starts
// at the projection of 'from' on its closest segment and ends at the projection of 'to' on its closest segment.
// The returned line includes all original vertices between these segments.
//
// Parameters:
//   - line: []Point - Input line as a slice of Points
//   - from: Point - Starting point to cut from
//   - to: Point - Ending point to cut to
//
// Returns:
//   - []Point - A new slice of Points representing the cut portion of the original line
//   - nil if the input line has fewer than 2 points
//
// Note: The function assumes the input points are relatively close to the line.
// The search for the end point starts from the start point's segment to maintain proper ordering.
func CutBetweenPoints(line []Point, from Point, to Point) []Point {
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

// CutBetweenPositions returns a slice of points representing a segment of the input line
// between the specified start and end distances along the line.
//
// The function takes a line represented as a slice of Points, a slice of cumulative distances
// along the line (dists), and start/end distances (startDist, endDist) indicating where to cut.
//
// It returns a new slice containing:
// - An interpolated point at startDist
// - All original points between start and end positions
// - An interpolated point at endDist
//
// Returns nil if the cut positions cannot be determined (e.g. distances out of range).
//
// Parameters:
//   - line: Slice of Points representing the polyline
//   - dists: Slice of cumulative distances along the line
//   - startDist: Distance along the line where cut should start
//   - endDist: Distance along the line where cut should end
//
// Returns:
//   - Slice of Points representing the cut segment, or nil if cut is not possible
func CutBetweenPositions(line []Point, dists []float64, startDist float64, endDist float64) []Point {
	spt, ept, sidx, eidx, ok := cutBetweenPositions(line, dists, startDist, endDist)
	if !ok {
		return nil
	}
	var ret []Point
	ret = append(ret, spt)
	ret = append(ret, line[sidx:eidx]...)
	ret = append(ret, ept)
	return ret
}

func cutBetweenPositions(line []Point, dists []float64, startDist float64, endDist float64) (Point, Point, int, int, bool) {
	for i := 0; i < len(dists)-1; i++ {
		if startDist >= dists[i] && startDist <= dists[i+1] {
			// fmt.Println("idist:", dists[i], dists[i+1], "pt:", line[i], line[i+1], "startDist:", startDist)
			for j := i; j < len(dists)-1; j++ {
				// fmt.Println("\tjdist:", dists[j], dists[j+1], "pt:", line[j], line[j+1], "endDist:", endDist)
				if endDist >= dists[j] && endDist <= dists[j+1] {
					spt := segPos(line[i], line[i+1], dists[i], dists[i+1], startDist)
					ept := segPos(line[j], line[j+1], dists[j], dists[j+1], endDist)
					return spt, ept, i + 1, j + 1, true
				}
			}
		}
	}
	return Point{}, Point{}, 0, 0, false
}

func segPos(apt Point, bpt Point, apos float64, bpos float64, dist float64) Point {
	segrel := (dist - apos) / (bpos - apos)
	segx := bpt.Lon - apt.Lon
	segy := bpt.Lat - apt.Lat
	return Point{
		Lon: apt.Lon + segrel*segx,
		Lat: apt.Lat + segrel*segy,
	}
}
