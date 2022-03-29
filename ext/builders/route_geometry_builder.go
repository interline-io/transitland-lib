package builders

import (
	"sort"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/xy"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/twpayne/go-geom"
)

type RouteGeometry struct {
	RouteID          string
	Generated        bool
	Geometry         tl.LineString
	CombinedGeometry tl.Geometry
	tl.MinEntity
	tl.FeedVersionEntity
}

func (ent *RouteGeometry) Filename() string {
	return "tl_route_geometries.txt"
}

func (ent *RouteGeometry) TableName() string {
	return "tl_route_geometries"
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
		pts := []xy.Point{}
		for _, c := range v.Geometry.Coords() {
			pts = append(pts, xy.Point{Lon: c[0], Lat: c[1]})
		}
		pp.shapeInfos[eid] = shapeInfo{Generated: v.Generated, Length: v.Geometry.Length(), Line: pts}
	case *tl.Trip:
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
	// Get the candidate shapes
	selectedShapes := map[string][]string{}
	for rid, dirs := range pp.shapeCounts {
		shapeTripCount := map[string]int{}
		routeSelectedShapes := map[string]int{}
		for _, dirShapes := range dirs {
			// Ensure stable selection of longest shape (most trips wins for equal length)
			dirShapesSorted := sortMap(dirShapes)
			dirCount := float64(0)
			longestShape := ""
			longestShapeLength := 0.0
			for _, shapeId := range dirShapesSorted {
				v := dirShapes[shapeId]
				shapeTripCount[shapeId] += v
				dirCount += float64(v)
				// Include the longest, non-generated shape
				if si, ok := pp.shapeInfos[shapeId]; ok && !si.Generated && si.Length > longestShapeLength {
					longestShape = shapeId
					longestShapeLength = si.Length
				}
			}
			for shapeId, v := range dirShapes {
				if shapeId == longestShape || float64(v)/dirCount > 0.1 {
					routeSelectedShapes[shapeId] += v
				}
			}
		}
		// Prefer to use real shapes; only use generated if no real shapes exist.
		var routeSelectedReal []string
		var routeSelectedGenerated []string
		routeSelectedSorted := sortMap(routeSelectedShapes) // sort
		// fmt.Println("sorted:", routeSelectedSorted)
		for _, v := range routeSelectedSorted {
			if pp.shapeInfos[v].Generated {
				routeSelectedGenerated = append(routeSelectedGenerated, v)
			} else {
				routeSelectedReal = append(routeSelectedReal, v)
			}
		}
		if len(routeSelectedReal) > 0 {
			selectedShapes[rid] = routeSelectedReal
		} else {
			selectedShapes[rid] = routeSelectedGenerated
		}
	}
	// Now build the selected shapes
	for rid, shapeIds := range selectedShapes {
		if len(shapeIds) == 0 {
			continue
		}
		ent := RouteGeometry{RouteID: rid}
		g := geom.NewMultiLineString(geom.XY)
		g.SetSRID(4326)
		for i, shapeId := range shapeIds {
			si, ok := pp.shapeInfos[shapeId]
			if !ok || len(si.Line) < 2 {
				continue
			}
			var pnts []float64
			for _, c := range si.Line {
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
		}
		_, _, err := copier.CopyEntity(&ent)
		if err != nil {
			return err
		}
	}
	return nil
}

///////

type shapeInfo struct {
	Line      []xy.Point
	Length    float64
	Generated bool
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
