package tlxy

import (
	"github.com/twpayne/go-polyline"
)

type LineM struct {
	Coords []Point
	Data   []float64
}

func DecodePolyline(p string) ([]Point, error) {
	return DecodePolylineBytes([]byte(p))
}

func DecodePolylineBytes(p []byte) ([]Point, error) {
	coords, _, err := polyline.DecodeCoords(p)
	var ret []Point
	for _, c := range coords {
		ret = append(ret, Point{Lon: c[1], Lat: c[0]})
	}
	return ret, err
}

func EncodePolyline(coords []Point) []byte {
	var g [][]float64
	for _, c := range coords {
		g = append(g, []float64{c.Lat, c.Lon})
	}
	return polyline.EncodeCoords(g)
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
