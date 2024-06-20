package tlxy

import (
	"fmt"
	"math"

	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
	"github.com/twpayne/go-polyline"
)

type Line = []Point

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

// LineRelativePositionsFallback returns the relative position along the line for each point.
func LineRelativePositionsFallback(line []Point) []float64 {
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

// LineRelativePositions finds the relative position of the closest point along the line for each point.
func LineRelativePositions(line []Point, points []Point) []float64 {
	positions := make([]float64, len(points))
	for i, p := range points {
		_, _, d := LineClosestPoint(line, p)
		positions[i] = d
	}
	return positions
}

func LineFlatCoords(line []Point) []float64 {
	var ret []float64
	for _, c := range line {
		ret = append(ret, c.Lon, c.Lat)
	}
	return ret
}

func LineContains(a []Point, b []Point) bool {
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

func LineEquals(a []Point, b []Point) bool {
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

func LineSimilarity(a []Point, b []Point) (float64, error) {
	var features []*geojson.Feature
	distances := make([]float64, len(a))
	for i, p := range a {
		minpt, _, _ := LineClosestPoint(b, p)
		d := DistanceHaversine(p, minpt)
		distances[i] = d
		fmt.Println("p:", p, "projected:", minpt, "d:", d)
		features = append(features, &geojson.Feature{
			Properties: map[string]any{"name": "connect", "stroke": "#0000ff", "stroke-width": 1},
			Geometry: geom.NewLineStringFlat(geom.XY, []float64{
				p.Lon, p.Lat,
				minpt.Lon, minpt.Lat,
			}),
		})
	}
	features = append(features, &geojson.Feature{
		Properties: map[string]any{"name": "a", "stroke": "#00ff00", "stroke-width": 1},
		Geometry:   geom.NewLineStringFlat(geom.XY, LineFlatCoords(a)),
	})
	features = append(features, &geojson.Feature{
		Properties: map[string]any{"name": "b", "stroke": "#ff0000", "stroke-width": 1},
		Geometry:   geom.NewLineStringFlat(geom.XY, LineFlatCoords(b)),
	})

	fc := geojson.FeatureCollection{Features: features}
	d, _ := fc.MarshalJSON()
	fmt.Println(string(d))

	// Calculate RMSD like value
	distanceSum := 0.0
	for _, v := range distances {
		distanceSum += v
	}
	rmsd := math.Sqrt((1 / float64(len(distances)) * distanceSum))
	fmt.Println("rmsd", rmsd)
	return rmsd, nil
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
