package tlxy

import (
	"context"
	"math"

	"github.com/interline-io/log"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

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

// CutBetweenPoints attempts to cut a line based on the
// relative positions of two nearby points projected onto the line.
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

// func CutBetweenPoints(line []Point, startPoint Point, endPoint Point) []Point {
// 	spt, sidx, _ := LineClosestPoint(line, startPoint)
// 	ept, eidx, _ := LineClosestPoint(line, endPoint)
// 	if eidx < sidx {
// 		return nil
// 	}
// 	if DistanceHaversine(startPoint, spt) > 1000 || DistanceHaversine(endPoint, ept) > 1000 {
// 		return nil
// 	}
// 	var ret []Point
// 	ret = append(ret, spt)
// 	ret = append(ret, line[sidx:eidx]...)
// 	ret = append(ret, ept)
// 	return ret
// }

// CutBetweenPositions is similar to CutBetweenPoints but takes absolute positions.
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

// CutBetweenPositions is similar to CutBetweenPoints but takes absolute positions.
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

// cutBetweenPositions is similar to CutBetweenPoints but takes absolute positions.
func cutBetweenPositionsDebug(line []Point, dists []float64, startDist float64, endDist float64, extraPts ...Point) []Point {
	ctx := context.TODO()
	spt, ept, sidx, eidx, ok := cutBetweenPositions(line, dists, startDist, endDist)
	if !ok {
		return nil
	}
	var ret []Point
	ret = append(ret, spt)
	ret = append(ret, line[sidx:eidx]...)
	ret = append(ret, ept)

	i := sidx
	j := eidx

	// DEBUG - Trace log a geojson feature with visualization of result
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
		Properties: map[string]any{"name": "start matched segment", "stroke": "#00ffff", "stroke-width": 5, "stroke-opacity": 0.2},
		Geometry: geom.NewLineStringFlat(geom.XY, []float64{
			line[i-1].Lon, line[i-1].Lat,
			line[i].Lon, line[i].Lat,
		}),
	})
	fs = append(fs, &geojson.Feature{
		Properties: map[string]any{"name": "end matched segment", "stroke": "#ff00ff", "stroke-width": 5, "stroke-opacity": 0.2},
		Geometry: geom.NewLineStringFlat(geom.XY, []float64{
			line[j-1].Lon, line[j-1].Lat,
			line[j].Lon, line[j].Lat,
		}),
	})
	fs = append(fs, &geojson.Feature{
		Properties: map[string]any{"name": "start point", "marker-color": "#00ff00"},
		Geometry:   geom.NewPointFlat(geom.XY, []float64{spt.Lon, spt.Lat}),
	})
	fs = append(fs, &geojson.Feature{
		Properties: map[string]any{"name": "start point to line[i]", "stroke": "#00ff00"},
		Geometry: geom.NewLineStringFlat(geom.XY, []float64{
			spt.Lon, spt.Lat,
			line[i].Lon, line[i].Lat,
		}),
	})
	fs = append(fs, &geojson.Feature{
		Properties: map[string]any{"name": "ept", "marker-color": "#ff0000"},
		Geometry:   geom.NewPointFlat(geom.XY, []float64{ept.Lon, ept.Lat}),
	})
	fs = append(fs, &geojson.Feature{
		Properties: map[string]any{"name": "end point to line[j-1]", "stroke": "#ff0000"},
		Geometry: geom.NewLineStringFlat(geom.XY, []float64{
			line[j-1].Lon, line[j-1].Lat,
			ept.Lon, ept.Lat,
		}),
	})
	fc := geojson.FeatureCollection{Features: fs}
	d, _ := fc.MarshalJSON()
	log.For(ctx).Trace().Msgf("LineBetweenPositions: %s", string(d))
	return ret
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
