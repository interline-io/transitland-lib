package geomcache

import (
	"errors"
	"fmt"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tlxy"
)

func arePositionsSorted(a []float64) bool {
	if len(a) < 2 {
		return true
	}
	if a[0] == a[len(a)-1] {
		return false
	}
	for i := 1; i < len(a); i++ {
		if a[i] < a[i-1] {
			return false
		}
	}
	return true
}

type stopPositionInfo struct {
	Positions  []float64
	DistLength float64
}

// GeomCache helps speed up StopTime interpolating by caching various results
type GeomCache struct {
	stops         map[string]tlxy.Point
	shapes        map[string]tlxy.ShapeInfo
	stopPositions map[string]stopPositionInfo
}

// NewGeomCache returns an initialized geomCache
func NewGeomCache() *GeomCache {
	return &GeomCache{
		stops:         map[string]tlxy.Point{},
		shapes:        map[string]tlxy.ShapeInfo{},
		stopPositions: map[string]stopPositionInfo{},
	}
}

// AddStopGeom adds a Stop to the geometry cache.
func (g *GeomCache) AddStopGeom(eid string, pt tlxy.Point) {
	g.stops[eid] = pt
}

// GetStop returns the coordinates for the cached stop.
func (g *GeomCache) GetStop(eid string) tlxy.Point {
	return g.stops[eid]
}

// GetShape returns the coordinates for the cached shape.
func (g *GeomCache) GetShape(eid string) []tlxy.Point {
	return g.shapes[eid].Line()
}

func (g *GeomCache) GetShapeInfo(eid string) (tlxy.ShapeInfo, bool) {
	si, ok := g.shapes[eid]
	if !ok {
		log.Info().Msgf("shape '%s' not in cache", eid)
		// Show shapes that are in the cache
		for k := range g.shapes {
			log.Info().Msgf("  shape in cache: %s", k)
		}
	}
	return si, ok
}

// ShapeIDs returns all shape IDs in the cache.
func (g *GeomCache) ShapeIDs() []string {
	ids := make([]string, 0, len(g.shapes))
	for id := range g.shapes {
		ids = append(ids, id)
	}
	return ids
}

func (g *GeomCache) AddShapeGeom(eid string, line []tlxy.Point, dists []float64) {
	g.AddShapeGeomGenerated(eid, line, dists, false)
}

func (g *GeomCache) AddShapeGeomGenerated(eid string, line []tlxy.Point, dists []float64, generated bool) {
	if len(line) == 0 {
		return
	}
	// Calculate length, max segment length, and first point max distance
	var distLength float64
	var maxSegmentLength float64
	var firstPointMaxDistance float64
	firstPoint := line[0]
	for i := 1; i < len(line); i++ {
		d := tlxy.DistanceHaversine(line[i-1], line[i])
		distLength += d
		if d > maxSegmentLength {
			maxSegmentLength = d
		}
		if d2 := tlxy.DistanceHaversine(firstPoint, line[i]); d2 > firstPointMaxDistance {
			firstPointMaxDistance = d2
		}
	}
	// Validate ShapeDistTraveled values - use provided if valid
	if len(dists) > 0 && len(dists) == len(line) && dists[len(dists)-1]-dists[0] > 0 {
		distLength = dists[len(dists)-1]
	}
	// Create shapeInfo with encoded polyline for memory efficiency
	g.shapes[eid] = tlxy.NewShapeInfo(line, distLength, generated, maxSegmentLength, firstPointMaxDistance)
}

// MakeShape returns geometry for the given stops.
func (g *GeomCache) MakeShape(stopids ...string) ([]tlxy.Point, []float64, error) {
	var line []tlxy.Point
	var dists []float64
	for _, stopid := range stopids {
		newPoint, ok := g.stops[stopid]
		if !ok {
			return line, dists, fmt.Errorf("stop '%s' not in cache", stopid)
		} else if newPoint.Lon == 0 || newPoint.Lat == 0 {
			return line, dists, fmt.Errorf("stop '%s' has zero coordinate", stopid)
		}
		line = append(line, newPoint)
	}
	// Calculate our own ShapeDistTraveled values
	dists = make([]float64, len(line))
	for i := 1; i < len(line); i++ {
		dists[i] = dists[i-1] + tlxy.DistanceHaversine(line[i-1], line[i])
	}
	return line, dists, nil
}

// InterpolateStopTimes uses the cached geometries to interpolate StopTimes.
// TODO: move to somewhere else
func (g *GeomCache) InterpolateStopTimes(trip *gtfs.Trip) ([]gtfs.StopTime, error) {
	sts := trip.StopTimes
	if len(sts) == 0 {
		return sts, nil
	}

	// Skip flex trips that cannot use stop-based geometry
	if !gtfs.CheckFlexStopTimes(sts).CanUseStopBasedGeometry() {
		return sts, nil
	}

	// Do we have valid ShapeDistTraveled values?
	validDists := true
	if sts[len(sts)-1].ShapeDistTraveled.Val-sts[0].ShapeDistTraveled.Val <= 0 {
		validDists = false
	}
	for i := 0; i < len(sts)-1; i++ {
		if sts[i+1].ShapeDistTraveled.Val < sts[i].ShapeDistTraveled.Val {
			validDists = false
		}
	}

	// We need to assign valid ShapeDistTraveled Values
	if !validDists {
		if err := g.setStopTimeDists(trip.ShapeID.Val, trip.StopPatternID.Val, sts); err != nil {
			return sts, err
		}
	}

	// Interpolate stops using the given or assigned ShapeDistTraveled values
	return InterpolateStopTimes(sts)
}

// TODO: move to somewhere else
func (g *GeomCache) setStopTimeDists(shapeId string, patternId int64, sts []gtfs.StopTime) error {
	// Check cache
	stopPositionsKey := fmt.Sprintf("%s-%d", shapeId, patternId)
	stopPositionInfo, ok := g.stopPositions[stopPositionsKey]
	if !ok {
		// Generate the stop-to-stop geometry
		stopLine := make([]tlxy.Point, 0, len(sts))
		for i := 0; i < len(sts); i++ {
			point, ok := g.stops[sts[i].StopID.Val]
			if !ok {
				return fmt.Errorf("stop '%s' not in cache", sts[i].StopID)
			}
			stopLine = append(stopLine, point)
		}

		// Get the known shape line and known shape distance
		var shapeLength float64
		var shapeLine []tlxy.Point
		if si, ok := g.shapes[shapeId]; ok {
			shapeLine = si.Line()
			shapeLength = si.DistLength
		} else {
			shapeLine = stopLine
			shapeLength = tlxy.LengthHaversine(stopLine)
		}

		// Calculate positions
		stopPositions := tlxy.LineRelativePositions(shapeLine, stopLine)

		// Check if the positions are sorted
		if !arePositionsSorted(stopPositions) {
			// log.For(ctx).Debug().Msgf("positions %f not increasing, falling back to stop positions; shapeline %f stopLine %f", positions, shapeline, stopLine)
			stopPositions = tlxy.LineRelativePositionsFallback(stopLine)
		}

		// Check again
		if !arePositionsSorted(stopPositions) {
			return errors.New("fallback positions not sorted")
		}

		stopPositionInfo.Positions = stopPositions
		stopPositionInfo.DistLength = shapeLength
		g.stopPositions[stopPositionsKey] = stopPositionInfo
	}

	if len(sts) != len(stopPositionInfo.Positions) {
		return errors.New("unequal stoptimes and positions")
	}

	// Set ShapeDistTraveled values
	for i := 0; i < len(sts); i++ {
		sts[i].ShapeDistTraveled.Set(stopPositionInfo.Positions[i] * stopPositionInfo.DistLength)
	}
	return nil
}
