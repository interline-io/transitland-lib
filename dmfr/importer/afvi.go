package importer

import (
	"fmt"
	"sort"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/xy"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

// Post processing that occurs after import...

// "SELECT tl_generate_feed_version_geometries($1)",
// "SELECT tl_generate_route_geometries($1)",
// "SELECT tl_generate_route_stops($1)",
// "SELECT tl_generate_agency_geometries($1)",
// "SELECT tl_generate_route_headways($1)",
// "SELECT tl_generate_agency_places($1)",
// "SELECT tl_generate_onestop_ids($1)",

type shapeInfo struct {
	length    float64
	generated bool
}

// DefaultShapeBuilder creates default shapes for routes.
type DefaultShapeBuilder struct {
	shapeInfos  map[string]shapeInfo
	shapeCounts map[string]map[int]map[string]int
	geomCache   *xy.GeomCache
}

// NewDefaultShapeBuilder returns a new DefaultShapeBuilder.
func NewDefaultShapeBuilder() *DefaultShapeBuilder {
	return &DefaultShapeBuilder{
		shapeInfos:  map[string]shapeInfo{},
		shapeCounts: map[string]map[int]map[string]int{},
		geomCache:   xy.NewGeomCache(),
	}
}

// SetGeomCache sets a shared geometry cache.
func (pp *DefaultShapeBuilder) SetGeomCache(g *xy.GeomCache) {
	pp.geomCache = g
}

// Validate .
func (pp *DefaultShapeBuilder) AfterValidate(ent tl.Entity) []error {
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

// BuildRouteShapes .
func (pp *DefaultShapeBuilder) Copy(copier *copier.Copier) error {
	emap := copier.EntityMap
	v, ok := copier.Writer.(*tldb.Writer)
	if !ok {
		return nil
	}
	atx := v.Adapter
	// Get the candidate shapes
	commonCount := 2
	selectedShapes := map[string][]string{}
	for rid, dirs := range pp.shapeCounts {
		dbid, ok := emap.Get("routes.txt", rid)
		if !ok {
			continue
		}
		rid = dbid
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
			selectedShapes[rid] = selectedReal
		} else {
			selectedShapes[rid] = selectedGenerated
		}
	}

	// Now build the selected shapes
	for rid, shapeids := range selectedShapes {
		fmt.Println("======== building shape for route:", rid)
		routeShape := geom.NewMultiLineString(geom.XY)
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
				if err := routeShape.Push(sl); err != nil {
					fmt.Println("failed to build route geometry:", err)
				}
			}
		}
		bb, _ := geojson.Marshal(routeShape)
		fmt.Println(string(bb))
		q := atx.Sqrl().
			Insert("tl_route_geometries2").
			Columns("route_id", "feed_version_id", "generated", "geometry").
			Values(rid, 0, false, &tl.Geometry{Geometry: routeShape, Valid: true})
		if _, err := q.Exec(); err != nil {
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
