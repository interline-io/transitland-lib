package xy

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/interline-io/transitland-lib/tl"
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

// GeomCache helps speed up StopTime interpolating by caching various results
type GeomCache struct {
	positions map[string][]float64
	stops     map[string]Point
	shapes    map[string][]Point
	lengths   map[string]float64
}

// NewGeomCache returns an initialized geomCache
func NewGeomCache() *GeomCache {
	return &GeomCache{
		positions: map[string][]float64{},
		stops:     map[string]Point{},
		shapes:    map[string][]Point{},
		lengths:   map[string]float64{},
	}
}

// AddStop adds a Stop to the geometry cache.
func (g *GeomCache) AddStop(eid string, stop tl.Stop) {
	c := stop.Geometry.FlatCoords()
	g.stops[eid] = Point{c[0], c[1]}
}

// GetStop returns the coordinates for the cached stop.
func (g *GeomCache) GetStop(eid string) Point {
	return g.stops[eid]
}

// GetShape returns the coordinates for the cached shape.
func (g *GeomCache) GetShape(eid string) []Point {
	return g.shapes[eid]
}

// AddShape adds a Shape to the geometry cache.
func (g *GeomCache) AddShape(eid string, shape tl.Shape) {
	if !shape.Geometry.Valid {
		return
	}
	sl := make([]Point, shape.Geometry.NumCoords())
	for i, c := range shape.Geometry.Coords() {
		sl[i] = Point{c[0], c[1]}
	}
	g.shapes[eid] = sl
}

// MakeShape returns geometry for the given stops.
func (g *GeomCache) MakeShape(stopids ...string) (tl.Shape, error) {
	shape := tl.Shape{}
	stopline := []float64{} // flatcoords
	prevPoint := Point{}
	for i, stopid := range stopids {
		if newPoint, ok := g.stops[stopid]; ok {
			if i > 0 {
				if d := Distance2d(prevPoint, newPoint); d > 10.0 {
					return shape, fmt.Errorf("distance from (%f,%f) to (%f,%f) is %f decimal degrees", prevPoint.Lon, prevPoint.Lat, newPoint.Lon, newPoint.Lat, d)
				}
			}
			stopline = append(stopline, newPoint.Lon, newPoint.Lat, 0.0)
			prevPoint = newPoint
		} else {
			return shape, fmt.Errorf("stop '%s' not in cache", stopid)
		}
	}
	shape.Geometry = tl.NewLineStringFromFlatCoords(stopline)
	shape.Generated = true
	return shape, nil
}

// InterpolateStopTimes uses the cached geometries to interpolate StopTimes.
func (g *GeomCache) InterpolateStopTimes(trip tl.Trip) ([]tl.StopTime, error) {
	// Check cache; make stopline
	stoptimes := trip.StopTimes
	if len(stoptimes) == 0 {
		return stoptimes, nil
	}
	stopline := make([]Point, len(stoptimes))
	shapeid := trip.ShapeID.Key
	k := strings.Join([]string{shapeid, strconv.Itoa(trip.StopPatternID)}, "|")
	for i := 0; i < len(stoptimes); i++ {
		point, ok := g.stops[stoptimes[i].StopID]
		if !ok {
			return stoptimes, fmt.Errorf("stop '%s' not in cache", stoptimes[i].StopID)
		}
		stopline[i] = point
	}
	shapeline := g.shapes[shapeid]
	// Check cache
	positions, ok := g.positions[k]
	if !ok {
		positions = LinePositions(shapeline, stopline)
		length := LengthHaversine(shapeline)
		// Check for simple or fallback positions
		if !arePositionsSorted(positions) || len(shapeline) == 0 {
			// log.Debug("positions %f not increasing, falling back to stop positions; shapeline %f stopline %f", positions, shapeline, stopline)
			positions = LinePositionsFallback(stopline)
			if !arePositionsSorted(positions) {
				return stoptimes, errors.New("fallback positions not sorted")
			}
			length = LengthHaversine(stopline)
		}
		g.positions[k] = positions
		g.lengths[k] = length
	}
	length, ok := g.lengths[k]
	if !ok {
		return stoptimes, errors.New("could not get length from cache")
	}
	if len(stoptimes) != len(positions) {
		return stoptimes, errors.New("unequal stoptimes and positions")
	}
	// Set ShapeDistTraveled
	for i := 0; i < len(stoptimes); i++ {
		stoptimes[i].ShapeDistTraveled = tl.NewOFloat(positions[i] * length)
	}
	return InterpolateStopTimes(stoptimes)
}
