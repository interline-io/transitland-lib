package segments

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/geomcache"
	"github.com/interline-io/transitland-lib/tl"
	xy "github.com/interline-io/transitland-lib/tlxy"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
	"golang.org/x/exp/maps"
)

var COLORS = []string{
	"#00ff00",
	"#ff0000",
	"#0000ff",
	"#a6cee3",
	"#1f78b4",
	"#b2df8a",
	"#33a02c",
	"#fb9a99",
	"#e31a1c",
	"#fdbf6f",
	"#ff7f00",
	"#cab2d6",
	"#6a3d9a",
	"#ffff99",
}

type routeInfo struct {
	MaxStops   int
	Color      string
	RouteDescs []string
	RouteIDs   []string
}

type stopPatternInfo struct {
	ID          int
	DirectionID int
	Count       int
	Pattern     []string
	Routes      map[string]int
	ShapeCounts map[string]int
}

type stopStopKey struct {
	FromStopID string
	ToStopID   string
}

func (key *stopStopKey) Reverse() stopStopKey {
	k := stopStopKey{
		FromStopID: key.ToStopID,
		ToStopID:   key.FromStopID,
	}
	return k
}

type SegmentBuilder struct {
	stopPatterns map[int]stopPatternInfo
	stopKeyMap   map[string]string
	routes       map[string]routeInfo
	routeKeyMap  map[string]string
	geomCache    *geomcache.GeomCache
	// options
	outfile     string
	minTripPct  float64
	offsetWidth float64
	lineWidth   float64
}

func NewSegmentBuilder() *SegmentBuilder {
	return &SegmentBuilder{
		stopKeyMap:   map[string]string{},
		stopPatterns: map[int]stopPatternInfo{},
		routeKeyMap:  map[string]string{},
		routes:       map[string]routeInfo{},
		minTripPct:   0.01,
		offsetWidth:  6,
		lineWidth:    6,
		geomCache:    geomcache.NewGeomCache(),
	}
}

func (pp *SegmentBuilder) AfterWrite(eid string, ent tl.Entity, emap *tl.EntityMap) error {
	switch v := ent.(type) {
	case *tl.Stop:
		stopKey := v.StopID
		if v.ParentStation.Valid {
			stopKey = pp.stopKeyMap[v.ParentStation.Val]
		}
		// stopKey = v.StopName
		pp.stopKeyMap[v.StopID] = stopKey
	case *tl.Route:
		routeKey := v.RouteColor
		rk := pp.routes[routeKey]
		rk.Color = v.RouteColor
		rk.RouteIDs = append(rk.RouteIDs, v.RouteID)
		rk.RouteDescs = append(
			rk.RouteDescs,
			fmt.Sprintf("%s: %s %s (%s)", v.AgencyID, v.RouteShortName, v.RouteLongName, v.RouteID),
		)
		pp.routes[routeKey] = rk
		pp.routeKeyMap[v.RouteID] = routeKey
	case *tl.Trip:
		patKey := v.StopPatternID
		routeKey, ok := pp.routeKeyMap[v.RouteID]
		if !ok {
			log.Trace().Str("trip", v.TripID).Msgf("trip: %s has no routeKey, skipping", v.TripID)
			return nil
		}
		if v.ShapeID.Val == "" {
			log.Trace().Str("trip", v.TripID).Msgf("trip: %s has no shape, skipping", v.TripID)
			return nil
		}
		// Check or create pattern
		pat, ok := pp.stopPatterns[patKey]
		if !ok {
			pat = stopPatternInfo{
				ID:          patKey,
				DirectionID: v.DirectionID,
				ShapeCounts: map[string]int{},
				Routes:      map[string]int{},
			}
			for i := 0; i < len(v.StopTimes); i++ {
				pat.Pattern = append(pat.Pattern, pp.stopKeyMap[v.StopTimes[i].StopID])
			}
		}
		if v.DirectionID != pat.DirectionID {
			log.Error().Str("trip", v.TripID).Msgf("trip: %s pattern %d has multiple directions, skipping", v.TripID, patKey)
			return nil
		}

		// Save pattern
		pat.ShapeCounts[v.ShapeID.Val] += 1
		pat.Count += 1
		pat.Routes[routeKey] += 1
		pp.stopPatterns[patKey] = pat
	}
	return nil
}

func (pp *SegmentBuilder) Copy(copier *copier.Copier) error {
	shapeSlices := pp.makeShapeSlices()
	stopToStops := pp.makeStopToStops()
	// maps.Clear(stopToStops)

	// Sort patterns by direction, count, length
	sortedPatterns := maps.Values(pp.stopPatterns)
	slices.SortFunc(sortedPatterns, func(a, b stopPatternInfo) int {
		return cmp.Or(
			cmp.Compare(a.DirectionID, b.DirectionID),
			inv(cmp.Compare(a.Count, b.Count)),
			inv(cmp.Compare(len(a.Pattern), len(b.Pattern))),
			cmp.Compare(a.ID, b.ID),
		)
	})

	// Group stop patterns by route
	routeMaxStops := map[string]int{}
	routePatterns := map[string][]stopPatternInfo{}
	for _, pat := range sortedPatterns {
		for routeKey := range pat.Routes {
			routePatterns[routeKey] = append(routePatterns[routeKey], pat)
			routeMaxStops[routeKey] = maxInt(routeMaxStops[routeKey], len(pat.Pattern))
		}
	}

	// Sort routes by number of stops
	sortedRoutes := maps.Keys(pp.routes)
	slices.SortFunc(sortedRoutes, func(a, b string) int {
		return cmp.Or(
			inv(cmp.Compare(routeMaxStops[a], routeMaxStops[b])),
			strings.Compare(a, b),
		)
	})

	// Create drawable segments and expand intermediate stops
	routeRenderSegments := map[string][][]stopStopKey{}
	for _, routeKey := range sortedRoutes {
		lgRt := log.Logger.With().Str("route", routeKey).Logger()
		lgRt.Info().Int("max_stops", routeMaxStops[routeKey]).Msgf("process route: %s", routeKey)
		lgRt.Info().Msgf("\tgroup: %v", pp.routes[routeKey].RouteDescs)

		// Get total trip count
		totalTripCount := 0
		for _, pat := range routePatterns[routeKey] {
			totalTripCount += pat.Count
		}

		// Determine which segments to render for each route
		for _, pat := range routePatterns[routeKey] {
			lgPat := lgRt.With().Int("pat", pat.ID).Logger()
			lgPat.Info().Msgf(
				"\tpat: %d dir: %d count: %d len: %d, origin: %s dest: %s",
				pat.ID,
				pat.DirectionID,
				pat.Count,
				len(pat.Pattern),
				pat.Pattern[0],
				pat.Pattern[len(pat.Pattern)-1],
			)

			// Check trip pct
			tripPct := float64(pat.Count) / float64(totalTripCount)
			if tripPct < pp.minTripPct {
				lgPat.Error().Msgf("\t\tskipping: pat trip percent %f is less than %f", tripPct, pp.minTripPct)
				continue
			}

			// Go through each stop segment
			var render []stopStopKey
			for i := 0; i < len(pat.Pattern)-1; i++ {
				from := pat.Pattern[i]
				to := pat.Pattern[i+1]

				// Check if we need to expand to intermediate stops
				var expanded []string
				if a, ok := stopToStops[from][to]; ok {
					lgPat.Info().Msgf("\t\tusing expanded pattern for: %s -> %s: %v", from, to, a)
					expanded = a
				} else {
					expanded = append(expanded, from, to)
				}

				// Check if we've seen each sub-segment
				for j := 0; j < len(expanded)-1; j++ {
					key := stopStopKey{
						FromStopID: expanded[j],
						ToStopID:   expanded[j+1],
					}
					lgPat.Info().Msgf("\t\t%v", key)
					render = append(render, key)
				}
			}
			if len(render) > 0 {
				routeRenderSegments[routeKey] = append(routeRenderSegments[routeKey], render)
			}
		}
	}

	// sortedRoutes = []string{
	// 	"FF0000", // red
	// 	"FFFF33", // yellow
	// 	"0099CC", // blue
	// 	"339933", // green
	// 	"FF9933", // orange
	// 	"D5CFA3", // beige
	// }

	// Take drawable segments and convert to geojson features
	departSlots := map[string]map[string]int{}
	for key := range shapeSlices {
		departSlots[key.FromStopID] = map[string]int{}
		departSlots[key.ToStopID] = map[string]int{}
	}
	checkOffset := func(routeKey string, key stopStopKey, prevRouteOffset int) int {
		taken := map[int]bool{}
		for k, v := range departSlots[key.FromStopID] {
			if k != routeKey {
				taken[v] = true
			} else {
				prevRouteOffset = v // minInt(prevRouteOffset, v)
			}
		}
		for k, v := range departSlots[key.ToStopID] {
			if k != routeKey {
				taken[v] = true
			} else {
				prevRouteOffset = v // minInt(prevRouteOffset, v)
			}
		}
		routeOffset := prevRouteOffset
		if taken[routeOffset] {
			routeOffset = prevRouteOffset
			for ; ; routeOffset++ {
				if !taken[routeOffset] {
					break
				}
			}
		}
		return routeOffset
	}

	checkDir := map[stopStopKey]stopStopKey{}
	var features []*geojson.Feature
	for routeKeyIdx, routeKey := range sortedRoutes {
		_ = routeKeyIdx
		splitSegments := routeRenderSegments[routeKey]
		routeRendered := map[stopStopKey]bool{}
		lgRt := log.Logger.With().Str("route", routeKey).Logger()
		lgRt.Info().Msgf("render route: %s", routeKey)
		lgRt.Info().Msgf("\tgroup: %v", pp.routes[routeKey].RouteDescs)
		for i, keys := range splitSegments {
			lgRt.Info().Msgf("\tsegment: %d", i)
			checkFwd := 0
			checkRev := 0
			for _, key := range keys {
				if _, ok := checkDir[key]; ok {
					checkFwd += 1
				} else if _, ok := checkDir[key.Reverse()]; ok {
					checkRev += 1
				}
			}
			if checkRev > checkFwd {
				lgRt.Info().Msgf("\t\treversing... checkFwd: %d checkRev: %d", checkFwd, checkRev)
				slices.Reverse(keys)
				for i := 0; i < len(keys); i++ {
					keys[i] = keys[i].Reverse()
				}
			}

			prevRouteOffset := 1
			for _, key := range keys {
				if routeRendered[key] {
					lgRt.Info().Msgf("\t\tkey %v: already rendered, skipping...", key)
					continue
				}

				checkDir[key] = key
				routeOffset := checkOffset(routeKey, key, prevRouteOffset)
				prevRouteOffset = routeOffset
				line, ok := shapeSlices[key]
				if !ok {
					lgRt.Info().Msgf("\t\tkey: %v, no shape, skipping", key)
					continue
				}

				lgRt.Info().Msgf("\t\tkey: %v: using offset: %d", key, routeOffset)
				routeRendered[key] = true
				routeRendered[key.Reverse()] = true
				departSlots[key.FromStopID][routeKey] = routeOffset

				// Add feature
				featureColor := "#" + pp.routes[routeKey].Color
				offset := float64(routeOffset)
				// offset := math.Floor(float64(routeOffset+1)/2) * math.Pow(-1, float64(routeOffset+1))
				feature := &geojson.Feature{
					Geometry: geom.NewLineStringFlat(geom.XY, lineFlat(line)),
					Properties: map[string]any{
						"route":        routeKey,
						"key":          fmt.Sprintf("%v", key),
						"stroke":       featureColor,
						"stroke-width": pp.lineWidth,
						"line-offset":  pp.offsetWidth * offset,
					},
				}
				features = append(features, feature)
				// fmt.Println("feat:", debugLine(line))

			}
		}
	}

	// Write output
	if pp.outfile != "" {
		if f, err := os.Create(pp.outfile); err == nil {
			fc := geojson.FeatureCollection{Features: features}
			d, _ := fc.MarshalJSON()
			f.Write(d)
		} else {
			return err
		}
	}
	return nil
}

// SetGeomCache sets a shared geometry cache.
func (pp *SegmentBuilder) SetGeomCache(g *geomcache.GeomCache) {
	pp.geomCache = g
}

func (pp *SegmentBuilder) makeShapeSlices() map[stopStopKey][]xy.Point {
	// Slice shapes
	shapeSlices := map[stopStopKey][]xy.Point{}
	for _, pat := range pp.stopPatterns {
		shapeId := mapMax(pat.ShapeCounts)
		lgPat := log.Logger.With().Int("pat", pat.ID).Str("shape", shapeId).Logger()
		lgPat.Info().Msgf("makeShapeSlices")
		// fmt.Println("shape:", debugLine(shapeInfo.Line))
		// fmt.Println("\tshape dists:", shapeInfo.Dists)
		// patDists := pat.ShapeDistTraveled[shapeId]
		// lgPat.Info().Msgf("\tusing pat dists: %v", patDists)

		shapeInfo := pp.geomCache.GetShapeInfo(shapeId)
		for i := 0; i < len(pat.Pattern)-1; i++ {
			key := stopStopKey{
				FromStopID: pat.Pattern[i],
				ToStopID:   pat.Pattern[i+1],
			}
			if _, ok := shapeSlices[key]; ok {
				lgPat.Info().Msgf("\tkey: %v (cached)", key)
				continue
			}
			lgPat.Info().Msgf("\tkey: %v", key)
			spt := pp.geomCache.GetStop(key.FromStopID)
			ept := pp.geomCache.GetStop(key.ToStopID)

			var sliceLine []xy.Point
			if len(shapeInfo.Line) > 0 {
				// sliceLine = xy.LineSliceShapeDistTraveled(
				// 	shapeInfo.Line,
				// 	shapeInfo.Dists,
				// 	patDists[i],
				// 	patDists[i+1],
				// 	pp.geomCache.GetStop(key.FromStopID),
				// 	pp.geomCache.GetStop(key.ToStopID),
				// )
				sliceLine = xy.CutBetweenPoints(shapeInfo.Line, spt, ept)
			}
			// fmt.Println(debugLine(sliceLine, spt, ept))
			if sliceLineDist, ptDist := xy.LengthHaversine(sliceLine), xy.DistanceHaversine(spt, ept); sliceLineDist < ptDist {
				// fmt.Println(debugLine(shapeInfo.Line))
				lgPat.Error().Msgf("\t\tshape slice length %f less than straight line dist %f", sliceLineDist, ptDist)
				continue
			}
			if len(sliceLine) == 0 {
				lgPat.Error().Msgf("\t\tno shape slice for key: %v shape: %s", key, shapeId)
				continue
			}
			// fmt.Println("\t\tresult:\n", debugLine(
			// 	sliceLine,
			// 	pp.geomCache.GetStop(key.FromStopID),
			// 	pp.geomCache.GetStop(key.ToStopID),
			// ))
			shapeSlices[key] = sliceLine
		}
	}
	return shapeSlices
}
func (pp *SegmentBuilder) makeStopToStops() map[string]map[string][]string {
	// Handle shortcuts
	// TODO: make shortcuts agency or route specific?
	stopToStops := map[string]map[string][]string{}
	for _, pat := range pp.stopPatterns {
		// fmt.Println("pat:", pat.ID)
		for i := 0; i < len(pat.Pattern); i++ {
			from := pat.Pattern[i]
			if _, ok := stopToStops[from]; !ok {
				stopToStops[from] = map[string][]string{}
			}
			// Store a ref to slice if more intermediate stops exist
			for j := i + 1; j < len(pat.Pattern); j++ {
				to := pat.Pattern[j]
				stoplen := j - i
				if stoplen > 2 && stoplen > len(stopToStops[from][to]) {
					stopToStops[from][to] = pat.Pattern[i : j+1]
				}
			}
		}
	}
	return stopToStops
}

func debugLine(line []xy.Point, pts ...xy.Point) string {
	var features []*geojson.Feature
	features = append(features, &geojson.Feature{
		Geometry: geom.NewLineStringFlat(geom.XY, lineFlat(line)),
	})
	for i, pt := range pts {
		features = append(features, &geojson.Feature{
			Geometry: geom.NewPointFlat(geom.XY, []float64{pt.Lon, pt.Lat}),
			Properties: map[string]any{
				"marker-color": COLORS[i%len(COLORS)],
			},
		})
	}
	fc := geojson.FeatureCollection{Features: features}
	d, _ := fc.MarshalJSON()
	return string(d)

}

func lineFlat(line []xy.Point) []float64 {
	var ret []float64
	for _, c := range line {
		ret = append(ret, c.Lon, c.Lat)
	}
	return ret
}

func runSplit[T any](v []T, splitFn func(prev, cur T) bool) [][]T {
	var ret [][]T
	var run []T
	run = append(run, v[0])
	for i := 1; i < len(v); i++ {
		if len(run) > 0 && splitFn(run[len(run)-1], v[i]) {
			ret = append(ret, run)
			run = nil
		}
		run = append(run, v[i])
	}
	if len(run) > 0 {
		ret = append(ret, run)
	}
	return ret
}

func checkReverse(line []xy.Point, end xy.Point) bool {
	if len(line) == 0 {
		return false
	}
	d1 := xy.Distance2d(end, line[0])
	d2 := xy.Distance2d(end, line[len(line)-1])
	return d2 > d1
}

func inv(a int) int {
	if a > 0 {
		return -1
	}
	if a < 0 {
		return 1
	}
	return 0
}

func mapMax[K comparable, V cmp.Ordered](m map[K]V) K {
	var ret K
	var cur V
	for k, v := range m {
		if v > cur {
			ret = k
			cur = v
		}
	}
	return ret
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
