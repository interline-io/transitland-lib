package builders

import (
	"errors"
	"fmt"
	"sort"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/xy"
	"github.com/interline-io/transitland-lib/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/twpayne/go-geom"
)

type RouteGeometry struct {
	RouteID          string
	Generated        bool
	Geometry         tl.LineString
	CombinedGeometry tl.Geometry
	Length           tl.Float
	MaxSegmentLength tl.Float
	tl.MinEntity
	tl.FeedVersionEntity
}

func (ent *RouteGeometry) Filename() string {
	return "tl_route_geometries.txt"
}

func (ent *RouteGeometry) TableName() string {
	return "tl_route_geometries"
}

///////

type shapeInfo struct {
	Line             []xy.Point
	Generated        bool
	Length           float64
	MaxSegmentLength float64
}

////////

// RouteGeometryBuilder creates default shapes for routes.
type RouteGeometryBuilder struct {
	shapeInfos  map[string]shapeInfo
	shapeCounts map[string]map[int]map[string]int
}

// NewRouteGeometryBuilder returns a new RouteGeometryBuilder.
func NewRouteGeometryBuilder() *RouteGeometryBuilder {
	return &RouteGeometryBuilder{
		shapeInfos:  map[string]shapeInfo{},
		shapeCounts: map[string]map[int]map[string]int{},
	}
}

// Counts the number of times a shape is used for each route,direction_id
func (pp *RouteGeometryBuilder) AfterWrite(eid string, ent tl.Entity, emap *tl.EntityMap) error {
	switch v := ent.(type) {
	case *tl.Shape:
		pts := make([]xy.Point, v.Geometry.NumCoords())
		for _, c := range v.Geometry.Coords() {
			pts = append(pts, xy.Point{Lon: c[0], Lat: c[1]})
		}
		// If we've already seen this line, re-use shapeInfo to reduce mem usage
		for x, si := range pp.shapeInfos {
			// Match on generated value too
			if xy.PointSliceEqual(pts, si.Line) && si.Generated == v.Generated {
				// Add to shape cache
				fmt.Println("already seen identical line for:", eid, ":", x)
				pp.shapeInfos[eid] = si
				return nil
			}
		}
		// Get distances
		maxLength := 0.0
		length := 0.0
		prevPoint := xy.Point{}
		for i, pt := range pts {
			if i > 0 {
				d := xy.DistanceHaversine(prevPoint.Lon, prevPoint.Lat, pt.Lon, pt.Lat)
				length += d
				if d > maxLength {
					maxLength = d
				}
			}
			prevPoint = pt
		}
		// Add to shape cache
		pp.shapeInfos[eid] = shapeInfo{
			Generated:        v.Generated,
			Length:           length,
			MaxSegmentLength: maxLength,
			Line:             pts,
		}
	case *tl.Trip:
		// shapeCounts is layered by: route id, direction id, shape id
		if v.ShapeID.Valid {
			if _, ok := pp.shapeCounts[v.RouteID]; !ok {
				pp.shapeCounts[v.RouteID] = map[int]map[string]int{}
			}
			if _, ok := pp.shapeCounts[v.RouteID][v.DirectionID]; !ok {
				pp.shapeCounts[v.RouteID][v.DirectionID] = map[string]int{}
			}
			pp.shapeCounts[v.RouteID][v.DirectionID][v.ShapeID.Key]++
		}
	}
	return nil
}

// Collects and assembles the default shapes and writes to the database
func (pp *RouteGeometryBuilder) Copy(copier *copier.Copier) error {
	// Process shapes for each route
	for rid, _ := range pp.shapeCounts {
		ent, err := pp.buildRouteShape(rid)
		if err != nil {
			log.Info().Err(err).Str("route_id", rid).Msg("failed to build route geometry")
			continue
		}
		if _, _, err := copier.CopyEntity(ent); err != nil {
			return err
		}
	}
	return nil
}

func (pp *RouteGeometryBuilder) buildRouteShape(rid string) (*RouteGeometry, error) {
	// Trip counts and selected shapes for this route
	candidateShapes := map[string]int{}
	// Process shapes for each direction
	dirs := pp.shapeCounts[rid]
	for _, dirShapes := range dirs {
		dirCount := float64(0)
		longestShape := ""
		longestShapeLength := 0.0
		// Sort by trip count to ensure stable selection of longest shape
		// (most trips wins for equal length).
		for _, shapeId := range sortMap(dirShapes) {
			// Check shape info and if this is the longest shape
			if si, ok := pp.shapeInfos[shapeId]; ok {
				dirCount += float64(dirShapes[shapeId])
				if si.Length > longestShapeLength {
					longestShape = shapeId
					longestShapeLength = si.Length
				}
			}
		}
		for shapeId, v := range dirShapes {
			// Ensure we have full shape info
			if si, ok := pp.shapeInfos[shapeId]; ok {
				// Check valid shape
				if len(si.Line) < 2 {
					continue
				}
				// Include if it is the longest shape
				// or accounts for at least 10% of trips in this direction
				if shapeId == longestShape || float64(v)/dirCount > 0.1 {
					candidateShapes[shapeId] += v
				}
			}
		}
	}

	// Split into real and generated shapes
	// Prefer to use real shapes; only use generated if no real shapes exist.
	var routeSelectedReal []string
	var routeSelectedGenerated []string
	for _, v := range sortMap(candidateShapes) {
		if pp.shapeInfos[v].Generated {
			routeSelectedGenerated = append(routeSelectedGenerated, v)
		} else {
			routeSelectedReal = append(routeSelectedReal, v)
		}
	}
	var routeSelectedShapes []string
	if len(routeSelectedReal) > 0 {
		routeSelectedShapes = routeSelectedReal
	} else {
		routeSelectedShapes = routeSelectedGenerated
	}
	if len(routeSelectedShapes) == 0 {
		return nil, errors.New("no shapes selected")
	}

	// Now build the route geometry from selected shapes
	ent := RouteGeometry{RouteID: rid}
	matches := [][]xy.Point{}
	for _, shapeId := range routeSelectedShapes {
		si, ok := pp.shapeInfos[shapeId]
		if !ok {
			continue
		}
		// Check if we've already included this shape entirely
		// This would probably work better if sorted from shortest to longest
		// instead of most frequent to least frequent.
		// A line will only be skipped if it's contained in a more frequent shape.
		// TODO: TopoJson style only store unique segments.
		for _, match := range matches {
			if xy.PointSliceContains(si.Line, match) {
				continue
			}
		}
		// Set if any selected shape is generated
		if si.Generated {
			ent.Generated = true
		}
		// Set to max selected shape length
		if si.Length > ent.Length.Float {
			ent.Length = tl.NewFloat(si.Length)
		}
		// Set to max selected shape segment length
		if si.MaxSegmentLength > ent.MaxSegmentLength.Float {
			ent.MaxSegmentLength = tl.NewFloat(si.MaxSegmentLength)
		}
		// OK
		matches = append(matches, si.Line)
	}

	// Build geom
	g := geom.NewMultiLineString(geom.XY)
	g.SetSRID(4326)
	for i, match := range matches {
		var pnts []float64
		for _, c := range match {
			pnts = append(pnts, c.Lon, c.Lat)
		}
		sl := geom.NewLineStringFlat(geom.XY, pnts)
		sl.SetSRID(4326)
		if sl == nil {
			continue
		}
		// Most frequent shape
		if i == 0 {
			ent.Geometry = tl.LineString{LineString: *sl, Valid: true}
		}
		// Add to MultiLineString
		if err := g.Push(sl); err != nil {
			// log.Debugf("failed to build route geometry:", err)
		}
	}
	if g.NumLineStrings() > 0 {
		ent.CombinedGeometry = tl.Geometry{Geometry: g, Valid: true}
	} else {
		// Skip entity
		return nil, errors.New("no geometries")
	}
	return &ent, nil
}

///////

func sortMap(value map[string]int) []string {
	type kv struct {
		Key   string
		Value int
	}
	var ss []kv
	for k, v := range value {
		ss = append(ss, kv{k, v})
	}
	sort.Slice(ss, func(i, j int) bool {
		a := ss[i]
		b := ss[j]
		if a.Value == b.Value {
			return a.Key < b.Key
		}
		return a.Value > b.Value
	})
	ret := []string{}
	for _, k := range ss {
		ret = append(ret, k.Key)
	}
	return ret
}
