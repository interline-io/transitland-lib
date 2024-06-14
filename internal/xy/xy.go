package xy

import (
	"math"
	"strconv"
	"strings"

	"github.com/interline-io/log"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
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

func segPos(apt Point, bpt Point, apos float64, bpos float64, dist float64) Point {
	segrel := (dist - apos) / (bpos - apos)
	segx := bpt.Lon - apt.Lon
	segy := bpt.Lat - apt.Lat
	return Point{
		Lon: apt.Lon + segrel*segx,
		Lat: apt.Lat + segrel*segy,
	}
}

// DistanceHaversinePoint .
func DistanceHaversinePoint(a, b Point) float64 {
	return DistanceHaversine(a.Lon, a.Lat, b.Lon, b.Lat)
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
func LineClosestPoint(line []Point, point Point) (Point, int, float64) {
	minidx := 0
	position := 0.0
	length := LengthHaversine(line)
	if length == 0 {
		return point, minidx, position
	}
	segpos := 0.0
	mind := math.MaxFloat64
	minp := Point{}
	for i := 1; i < len(line); i++ {
		start := line[i-1]
		end := line[i]
		segp, segd := SegmentClosestPoint(start, end, point)
		if segd < mind {
			minidx = i
			mind = segd
			minp = segp
			position = segpos + DistanceHaversinePoint(start, minp)
			if segd == 0 {
				break
			}
		}
		segpos += DistanceHaversinePoint(start, end)
	}
	return minp, minidx, position / length
}

// LineRelativePositionsFallback returns the relative position along the line for each point.
func LineRelativePositionsFallback(line []Point) []float64 {
	ret := make([]float64, len(line))
	length := LengthHaversine(line)
	position := 0.0
	ret[0] = 0.0
	for i := 1; i < len(line); i++ {
		position += DistanceHaversinePoint(line[i], line[i-1])
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

func LineBetweenPoints(line []Point, startPoint Point, endPoint Point) []Point {
	spt, sidx, _ := LineClosestPoint(line, startPoint)
	ept, eidx, _ := LineClosestPoint(line, endPoint)
	if eidx < sidx {
		return nil
	}
	if DistanceHaversinePoint(startPoint, spt) > 1000 || DistanceHaversinePoint(endPoint, ept) > 1000 {
		return nil
	}
	var ret []Point
	ret = append(ret, spt)
	ret = append(ret, line[sidx:eidx]...)
	ret = append(ret, ept)
	return ret
}

// This takes absolute positions, not relative positions.
func LineBetweenPositions(line []Point, dists []float64, startDist float64, endDist float64, extraPts ...Point) []Point {
	var ret []Point
	for i := 0; i < len(dists)-1; i++ {
		if startDist >= dists[i] && startDist <= dists[i+1] {
			// fmt.Println("idist:", dists[i], dists[i+1], "pt:", line[i], line[i+1], "startDist:", startDist)
			for j := i; j < len(dists)-1; j++ {
				// fmt.Println("\tjdist:", dists[j], dists[j+1], "pt:", line[j], line[j+1], "endDist:", endDist)
				if endDist >= dists[j] && endDist <= dists[j+1] {
					spt := segPos(line[i], line[i+1], dists[i], dists[i+1], startDist)
					ept := segPos(line[j], line[j+1], dists[j], dists[j+1], endDist)
					ret = append(ret, spt)
					ret = append(ret, line[i+1:j+1]...)
					ret = append(ret, ept)

					// DEBUG - Trace log a geojson feature with visualization of result
					if len(extraPts) > 0 {
						var fs []*geojson.Feature
						var baseLine []float64
						for _, pt := range ret {
							baseLine = append(baseLine, pt.Lon, pt.Lat)
						}
						var rawLine []float64
						for _, pt := range line {
							rawLine = append(rawLine, pt.Lon, pt.Lat)
						}
						fs = append(fs, &geojson.Feature{
							Properties: map[string]any{"name": "input line", "stroke": "#ff00ff", "stroke-width": 1, "stroke-opacity": 0.5},
							Geometry:   geom.NewLineStringFlat(geom.XY, rawLine),
						})
						fs = append(fs, &geojson.Feature{
							Properties: map[string]any{"name": "return line", "stroke": "#aaaaaa", "stroke-width": 20, "stroke-opacity": 0.5},
							Geometry:   geom.NewLineStringFlat(geom.XY, baseLine),
						})
						for _, extraPt := range extraPts {
							fs = append(fs, &geojson.Feature{
								Properties: map[string]any{"name": "extraPt", "marker-color": "#999999"},
								Geometry:   geom.NewPointFlat(geom.XY, []float64{extraPt.Lon, extraPt.Lat}),
							})
						}
						fs = append(fs, &geojson.Feature{
							Properties: map[string]any{"name": "lineSeg1", "stroke": "#00ffff", "stroke-width": 20, "stroke-opacity": 0.2},
							Geometry: geom.NewLineStringFlat(geom.XY, []float64{
								line[i].Lon, line[i].Lat,
								line[i+1].Lon, line[i+1].Lat,
							}),
						})
						fs = append(fs, &geojson.Feature{
							Properties: map[string]any{"name": "lineSeg2", "stroke": "#ff00ff", "stroke-width": 20, "stroke-opacity": 0.2},
							Geometry: geom.NewLineStringFlat(geom.XY, []float64{
								line[j].Lon, line[j].Lat,
								line[j+1].Lon, line[j+1].Lat,
							}),
						})
						fs = append(fs, &geojson.Feature{
							Properties: map[string]any{"name": "spt", "marker-color": "#00ff00"},
							Geometry:   geom.NewPointFlat(geom.XY, []float64{spt.Lon, spt.Lat}),
						})
						fs = append(fs, &geojson.Feature{
							Properties: map[string]any{"name": "sptSeg", "stroke": "#00ff00"},
							Geometry: geom.NewLineStringFlat(geom.XY, []float64{
								spt.Lon, spt.Lat,
								line[i+1].Lon, line[i+1].Lat,
							}),
						})
						fs = append(fs, &geojson.Feature{
							Properties: map[string]any{"name": "ept", "marker-color": "#ff0000"},
							Geometry:   geom.NewPointFlat(geom.XY, []float64{ept.Lon, ept.Lat}),
						})
						fs = append(fs, &geojson.Feature{
							Properties: map[string]any{"name": "eptSeg", "stroke": "#ff0000"},
							Geometry: geom.NewLineStringFlat(geom.XY, []float64{
								line[j].Lon, line[j].Lat,
								ept.Lon, ept.Lat,
							}),
						})
						fc := geojson.FeatureCollection{Features: fs}
						d, _ := fc.MarshalJSON()
						log.Trace().Str("geojson", string(d)).Msg("LineBetweenPositions")
					}
					return ret
				}
			}
		}
	}
	return ret
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
