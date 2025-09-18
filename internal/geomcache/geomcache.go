package geomcache

import (
	"errors"
	"fmt"

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

type ShapeInfo struct {
	Line       []tlxy.Point
	DistLength float64
}

type stopPositionInfo struct {
	Positions  []float64
	DistLength float64
}

// GeomCache helps speed up StopTime interpolating by caching various results
type GeomCache struct {
	stops         map[string]tlxy.Point
	shapes        map[string]ShapeInfo
	stopPositions map[string]stopPositionInfo
}

// NewGeomCache returns an initialized geomCache
func NewGeomCache() *GeomCache {
	return &GeomCache{
		stops:         map[string]tlxy.Point{},
		shapes:        map[string]ShapeInfo{},
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
	return g.shapes[eid].Line
}

func (g *GeomCache) GetShapeInfo(eid string) ShapeInfo {
	return g.shapes[eid]
}

func (g *GeomCache) AddShapeGeom(eid string, line []tlxy.Point, dists []float64) {
	// Create shapeInfo
	si := ShapeInfo{Line: line}
	// Validate ShapeDistTraveled values
	if len(dists) > 0 && len(dists) == len(line) && dists[len(dists)-1]-dists[0] > 0 {
		si.DistLength = dists[len(dists)-1]
	}
	// If we don't have ShapeDistTraveled values, calculate them
	if si.DistLength == 0 {
		si.DistLength = tlxy.LengthHaversine(line)
	}
	g.shapes[eid] = si
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
		stopLine := make([]tlxy.Point, len(sts))
		for i := 0; i < len(sts); i++ {
			point, ok := g.stops[sts[i].StopID.Val]
			if !ok {
				return fmt.Errorf("stop '%s' not in cache", sts[i].StopID)
			}
			stopLine[i] = point
		}

		// Get the known shape line and known shape distance
		var shapeLength float64
		var shapeLine []tlxy.Point
		if si, ok := g.shapes[shapeId]; ok {
			shapeLine = si.Line
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
