package validator

import (
	"sort"

	"github.com/interline-io/transitland-lib/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/twpayne/go-geom"
)

func buildRouteShapes(reader tl.Reader) map[string]*geom.MultiLineString {
	// Generate some route geoms...
	shapeLengths := map[string]float64{}
	for ent := range reader.Shapes() {
		if !ent.Geometry.Valid {
			continue
		}
		// cartesian units are fine for relative lengths
		shapeLengths[ent.ShapeID] = ent.Geometry.Length()
	}

	shapeCounts := map[string]map[int]map[string]int{}
	for ent := range reader.Trips() {
		if !ent.ShapeID.Valid {
			continue
		}
		if _, ok := shapeCounts[ent.RouteID]; !ok {
			shapeCounts[ent.RouteID] = map[int]map[string]int{}
		}
		if _, ok := shapeCounts[ent.RouteID][ent.DirectionID]; !ok {
			shapeCounts[ent.RouteID][ent.DirectionID] = map[string]int{}
		}
		shapeCounts[ent.RouteID][ent.DirectionID][ent.ShapeID.Key]++
	}
	commonCount := 5
	selectedShapes := map[string]map[string]bool{}
	for rid, dirs := range shapeCounts {
		selected := map[string]bool{}
		for _, dirshapes := range dirs {
			longest := ""
			longestlength := 0.0
			for shapeid := range dirshapes {
				sl, ok := shapeLengths[shapeid]
				if !ok {
					continue
				}
				if sl > longestlength {
					longest = shapeid
					longestlength = sl
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
		selectedShapes[rid] = selected
	}

	// Now load selected shapes
	loadShapes := map[string]*geom.LineString{}
	for _, shapeids := range selectedShapes {
		for shapeid := range shapeids {
			loadShapes[shapeid] = nil
		}
	}
	for ent := range reader.Shapes() {
		if _, ok := loadShapes[ent.ShapeID]; ok {
			// Transitland uses M coord for distance; must force 2D
			coords := []float64{}
			for _, coord := range ent.Geometry.LineString.Coords() {
				coords = append(coords, coord[0], coord[1])
			}
			loadShapes[ent.ShapeID] = geom.NewLineStringFlat(geom.XY, coords)
		}
	}

	routeShapes := map[string]*geom.MultiLineString{}
	for rid, shapeids := range selectedShapes {
		for shapeid := range shapeids {
			if shape, ok := loadShapes[shapeid]; ok && shape != nil {
				g, ok := routeShapes[rid]
				if !ok {
					g = geom.NewMultiLineString(geom.XY)
				}
				if err := g.Push(shape); err != nil {
					log.Errorf("failed to build route geometry: %s", err.Error())
				} else {
					routeShapes[rid] = g
				}
			}
		}
	}
	return routeShapes
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
