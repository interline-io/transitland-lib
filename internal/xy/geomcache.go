package xy

import (
	"errors"
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
)

func arePositionsSorted(a []float64) bool {
	if len(a) < 2 {
		return true
	}
	for i := 1; i < len(a); i++ {
		if a[i] < a[i-1] {
			return false
		}
	}
	return true
}

type ShapeInfo struct {
	Line       []Point
	Dists      []float64
	DistLength float64
	Length     float64
}

// GeomCache helps speed up StopTime interpolating by caching various results
type GeomCache struct {
	stopPositions map[string][]float64
	stops         map[string]Point
	shapes        map[string]ShapeInfo
}

// NewGeomCache returns an initialized geomCache
func NewGeomCache() *GeomCache {
	return &GeomCache{
		stopPositions: map[string][]float64{},
		stops:         map[string]Point{},
		shapes:        map[string]ShapeInfo{},
	}
}

// AddStop adds a Stop to the geometry cache.
func (g *GeomCache) AddStop(eid string, stop tl.Stop) {
	c := stop.Geometry.FlatCoords()
	g.AddStopGeom(eid, Point{c[0], c[1]})
}

func (g *GeomCache) AddStopGeom(eid string, pt Point) {
	g.stops[eid] = pt
}

// GetStop returns the coordinates for the cached stop.
func (g *GeomCache) GetStop(eid string) Point {
	return g.stops[eid]
}

// GetShape returns the coordinates for the cached shape.
func (g *GeomCache) GetShape(eid string) []Point {
	return g.shapes[eid].Line
}

func (g *GeomCache) GetShapeInfo(eid string) ShapeInfo {
	return g.shapes[eid]
}

// AddShape adds a Shape to the geometry cache.
func (g *GeomCache) AddShape(eid string, shape tl.Shape) {
	if !shape.Geometry.Valid {
		return
	}
	sl := make([]Point, shape.Geometry.NumCoords())
	dists := make([]float64, shape.Geometry.NumCoords())
	for i, c := range shape.Geometry.Coords() {
		sl[i] = Point{c[0], c[1]}
		dists[i] = c[2]
	}
	g.AddShapeGeom(eid, sl, dists)
}

func (g *GeomCache) AddShapeGeom(eid string, line []Point, dists []float64) {
	// Check if already exists, re-use slice to reduce mem
	for _, s := range g.shapes {
		if PointSliceEqual(line, s.Line) {
			line = s.Line
			dists = s.Dists
		}
	}
	// Create shapeInfo
	si := ShapeInfo{
		Line:   line,
		Length: LengthHaversine(line),
	}
	// Validate ShapeDistTraveled values
	if len(dists) > 0 && len(dists) == len(line) && dists[len(dists)-1]-dists[0] > 0 {
		// Use supplied ShapeDistTraveled values
		si.Dists = dists
		si.DistLength = dists[len(dists)-1]
	} else {
		// Calculate our own ShapeDistTraveled values
		si.Dists = make([]float64, len(line))
		for i := 1; i < len(line); i++ {
			si.Dists[i] = si.Dists[i-1] + DistanceHaversinePoint(line[i-1], line[i])
		}
		si.DistLength = dists[len(dists)-1]
	}
	g.shapes[eid] = si
}

// MakeShape returns geometry for the given stops.
func (g *GeomCache) MakeShape(stopids ...string) ([]Point, []float64, error) {
	var line []Point
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
		dists[i] = dists[i-1] + DistanceHaversinePoint(line[i-1], line[i])
	}
	return line, dists, nil
}

// InterpolateStopTimes uses the cached geometries to interpolate StopTimes.
func (g *GeomCache) InterpolateStopTimes(trip tl.Trip) ([]tl.StopTime, error) {
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
		if err := g.setStopTimeDists(trip.ShapeID.Val, trip.StopPatternID, sts); err != nil {
			return sts, err
		}
	}

	// Interpolate stops using the given or assigned ShapeDistTraveled values
	return InterpolateStopTimes(sts)
}

func (g *GeomCache) setStopTimeDists(shapeId string, patternId int, sts []tl.StopTime) error {
	// Check cache
	length := 0.0
	stopPositionsKey := fmt.Sprintf("%s-%d", shapeId, patternId)
	stopPositions, ok := g.stopPositions[stopPositionsKey]
	if !ok {
		// Generate the stop-to-stop geometry as fallback
		stopLine := make([]Point, len(sts))
		for i := 0; i < len(sts); i++ {
			point, ok := g.stops[sts[i].StopID]
			if !ok {
				return fmt.Errorf("stop '%s' not in cache", sts[i].StopID)
			}
			stopLine[i] = point
		}

		var shapeLine []Point
		if si, ok := g.shapes[shapeId]; ok {
			shapeLine = si.Line
			length = si.DistLength
		} else {
			shapeLine = stopLine
			length = LengthHaversine(stopLine)
		}

		// Calculate positions
		stopPositions = LineRelativePositions(shapeLine, stopLine)

		// Check for simple or fallback positions
		if !arePositionsSorted(stopPositions) || len(stopLine) == 0 {
			// log.Debugf("positions %f not increasing, falling back to stop positions; shapeline %f stopLine %f", positions, shapeline, stopLine)
			stopPositions = LineRelativePositionsFallback(stopLine)
			if !arePositionsSorted(stopPositions) {
				return errors.New("fallback positions not sorted")
			}
		}
		g.stopPositions[stopPositionsKey] = stopPositions
	}
	if len(sts) != len(stopPositions) {
		return errors.New("unequal stoptimes and positions")
	}
	// Set ShapeDistTraveled values
	for i := 0; i < len(sts); i++ {
		sts[i].ShapeDistTraveled = tt.NewFloat(stopPositions[i] * length)
	}
	return nil
}
