package builders

import (
	"sort"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/log"
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
	geomCache   *xy.GeomCache
	shapeCache  map[string][]xy.Point
}

// NewRouteGeometryBuilder returns a new RouteGeometryBuilder.
func NewRouteGeometryBuilder() *RouteGeometryBuilder {
	return &RouteGeometryBuilder{
		shapeInfos:  map[string]shapeInfo{},
		shapeCounts: map[string]map[int]map[string]int{},
		shapeCache:  map[string][]xy.Point{},
	}
}

// AfterValidate counts the number of times a shape is used for each route,direction_id
func (pp *RouteGeometryBuilder) AfterWrite(eid string, ent tl.Entity, emap *tl.EntityMap) error {
	switch v := ent.(type) {
	case *tl.Shape:
		pp.shapeInfos[eid] = shapeInfo{generated: v.Generated, length: v.Geometry.Length()}
		pts := []xy.Point{}
		for _, c := range v.Geometry.Coords() {
			pts = append(pts, xy.Point{Lon: c[0], Lat: c[1]})
		}
		pp.shapeCache[eid] = pts
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

// AfterCopy collects and assembles the default shapes and writes to the database
func (pp *RouteGeometryBuilder) Copy(copier *copier.Copier) error {
	// Get the candidate shapes
	commonCount := 5
	selectedShapes := map[string][]string{}
	mostFrequentShapes := map[string]string{}
	for rid, dirs := range pp.shapeCounts {
		selected := map[string]bool{}
		for _, dirshapes := range dirs {
			// Check if this is the longest shape
			longestShape := ""
			longestShapelength := 0.0
			for shapeid := range dirshapes {
				si, ok := pp.shapeInfos[shapeid]
				if !ok {
					continue
				}
				if si.length > longestShapelength && !si.generated {
					longestShape = shapeid
					longestShapelength = si.length
				}
			}
			if longestShape != "" {
				selected[longestShape] = true
			}
			// Number of trips for this direction
			dirTripCount := float64(0)
			for _, v := range dirshapes {
				dirTripCount += float64(v)
			}
			// Always include the most common shape
			bycount := sortMap(dirshapes)
			if len(bycount) > 0 {
				mostFrequentShapes[rid] = bycount[0]
			}
			// Include the n most common shapes that are at least 10% of trips
			for i, k := range bycount {
				if float64(dirshapes[k])/dirTripCount < 0.1 {
					continue
				}
				if i > commonCount {
					break
				}
				selected[k] = true
			}
		}
		// Prefer to use real shapes; only use generated if no real shapes exist.
		selectedReal := []string{}
		selectedGenerated := []string{}
		for v := range selected {
			if pp.shapeInfos[v].generated {
				selectedGenerated = append(selectedGenerated, v)
			} else {
				selectedReal = append(selectedReal, v)
			}
		}
		if len(selectedReal) > 0 {
			selectedShapes[rid] = selectedReal
		} else {
			selectedShapes[rid] = selectedGenerated
		}
	}
	// Now build the selected shapes
	for rid, shapeids := range selectedShapes {
		ent := RouteGeometry{RouteID: rid}
		// most frequent shape
		if shapeid, ok := mostFrequentShapes[rid]; ok {
			if coords, ok := pp.shapeCache[shapeid]; ok && len(coords) > 2 {
				pnts := []float64{}
				for _, c := range coords {
					pnts = append(pnts, c.Lon, c.Lat)
				}
				sl := geom.NewLineStringFlat(geom.XY, pnts)
				if sl != nil {
					ent.Geometry = tl.LineString{Valid: true, LineString: *sl}
				}
			}
		}
		// Build combined shape
		g := geom.NewMultiLineString(geom.XY)
		for _, shapeid := range shapeids {
			coords := pp.shapeCache[shapeid]
			if coords == nil || len(coords) < 2 {
				continue
			}
			pnts := []float64{}
			for _, c := range coords {
				pnts = append(pnts, c.Lon, c.Lat)
			}
			sl := geom.NewLineStringFlat(geom.XY, pnts)
			if sl != nil {
				if err := g.Push(sl); err != nil {
					log.Debug("failed to build route geometry:", err)
				}
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
	length    float64
	generated bool
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
		return ss[i].Value > ss[j].Value
	})
	ret := []string{}
	for _, k := range ss {
		ret = append(ret, k.Key)
	}
	return ret
}
