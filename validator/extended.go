package validator

import (
	"sort"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/twpayne/go-geom"
)

func buildRouteShapes(reader adapters.Reader) map[string]*geom.MultiLineString {
	// Generate some route geoms...
	shapeLengths := map[string]float64{}
	for shapeEnts := range reader.ShapesByShapeID() {
		ent := gtfs.NewShapeLineFromShapes(shapeEnts)
		if !ent.Geometry.Valid {
			continue
		}
		// cartesian units are fine for relative lengths
		shapeLengths[ent.ShapeID.Val] = ent.Geometry.Val.Length()
	}

	shapeCounts := map[string]map[int]map[string]int{}
	for ent := range reader.Trips() {
		if !ent.ShapeID.Valid {
			continue
		}
		if _, ok := shapeCounts[ent.RouteID.Val]; !ok {
			shapeCounts[ent.RouteID.Val] = map[int]map[string]int{}
		}
		if _, ok := shapeCounts[ent.RouteID.Val][ent.DirectionID.Int()]; !ok {
			shapeCounts[ent.RouteID.Val][ent.DirectionID.Int()] = map[string]int{}
		}
		shapeCounts[ent.RouteID.Val][ent.DirectionID.Int()][ent.ShapeID.Val]++
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
	for shapeEnts := range reader.ShapesByShapeID() {
		ent := gtfs.NewShapeLineFromShapes(shapeEnts)
		if _, ok := loadShapes[ent.ShapeID.Val]; ok {
			// Transitland uses M coord for distance; must force 2D
			coords := []float64{}
			for _, coord := range ent.Geometry.Val.Coords() {
				coords = append(coords, coord[0], coord[1])
			}
			loadShapes[ent.ShapeID.Val] = geom.NewLineStringFlat(geom.XY, coords)
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
