package builders

import (
	"fmt"
	"sort"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/xy"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/twpayne/go-geom"
)

type RouteGeometry struct {
	RouteID   string
	Generated bool
	Geometry  tl.Geometry
	tl.BaseEntity
}

func (ent *RouteGeometry) Filename() string {
	return "tl_route_geometries.txt"
}

func (ent *RouteGeometry) TableName() string {
	return "tl_route_geometries2"
}

type shapeInfo struct {
	length    float64
	generated bool
}

// RouteGeometryBuilder creates default shapes for routes.
type RouteGeometryBuilder struct {
	shapeInfos  map[string]shapeInfo
	shapeCounts map[string]map[int]map[string]int
	geomCache   *xy.GeomCache
}

// NewRouteGeometryBuilder returns a new RouteGeometryBuilder.
func NewRouteGeometryBuilder() *RouteGeometryBuilder {
	return &RouteGeometryBuilder{
		shapeInfos:  map[string]shapeInfo{},
		shapeCounts: map[string]map[int]map[string]int{},
	}
}

// SetGeomCache sets a shared geometry cache.
func (pp *RouteGeometryBuilder) SetGeomCache(g *xy.GeomCache) {
	pp.geomCache = g
}

// AfterValidate counts the number of times a shape is used for each route,direction_id
func (pp *RouteGeometryBuilder) AfterValidator(ent tl.Entity, emap *tl.EntityMap) error {
	switch v := ent.(type) {
	case *tl.Shape:
		pp.shapeInfos[v.ShapeID] = shapeInfo{generated: v.Generated, length: v.Geometry.Length()}
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
	emap := copier.EntityMap
	commonCount := 2
	selectedShapes := map[string][]string{}
	for rid, dirs := range pp.shapeCounts {
		dbid, ok := emap.Get("routes.txt", rid)
		if !ok {
			continue
		}
		selected := map[string]bool{}
		for _, dirshapes := range dirs {
			longest := ""
			longestlength := 0.0
			for shapeid := range dirshapes {
				si, ok := pp.shapeInfos[shapeid]
				if !ok {
					continue
				}
				if si.length > longestlength && !si.generated {
					longest = shapeid
					longestlength = si.length
				}
			}
			if longest != "" {
				selected[longest] = true
			}
			// Now get the n most common
			bycount := sortMap(dirshapes)
			for i, k := range bycount {
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
			selectedShapes[dbid] = selectedReal
		} else {
			selectedShapes[dbid] = selectedGenerated
		}
	}

	// Now build the selected shapes
	for rid, shapeids := range selectedShapes {
		g := geom.NewMultiLineString(geom.XY)
		for _, shapeid := range shapeids {
			coords := pp.geomCache.GetShape(shapeid)
			if coords == nil || len(coords) < 2 {
				fmt.Println("no shape:", shapeid)
				continue
			}
			pnts := []float64{}
			for _, c := range coords {
				pnts = append(pnts, c[0], c[1])
			}
			sl := geom.NewLineStringFlat(geom.XY, pnts)
			if sl != nil {
				if err := g.Push(sl); err != nil {
					fmt.Println("failed to build route geometry:", err)
				}
			}
		}
		_, _, err := copier.CopyEntity(&RouteGeometry{
			RouteID:   rid,
			Generated: false,
			Geometry:  tl.Geometry{Geometry: g, Valid: true},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

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
