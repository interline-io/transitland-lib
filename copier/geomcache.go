package copier

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/interline-io/gotransit"
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

// geomCache helps speed up StopTime interpolating by caching various results
type geomCache struct {
	positions map[string][]float64
	stops     map[string][2]float64
	shapes    map[string][][2]float64
	lengths   map[string]float64
}

// newGeomCache returns an initialized geomCache
func newGeomCache() *geomCache {
	return &geomCache{
		positions: map[string][]float64{},
		stops:     map[string][2]float64{},
		shapes:    map[string][][2]float64{},
		lengths:   map[string]float64{},
	}
}

// AddStop adds a Stop to the geometry cache.
func (g *geomCache) AddStop(eid string, stop gotransit.Stop) {
	c := stop.Geometry.FlatCoords()
	g.stops[eid] = [2]float64{c[0], c[1]}
}

// AddShape adds a Shape to the geometry cache.
func (g *geomCache) AddShape(eid string, shape gotransit.Shape) {
	if shape.Geometry == nil {
		return
	}
	sl := make([][2]float64, shape.Geometry.NumCoords())
	for i, c := range shape.Geometry.Coords() {
		sl[i] = [2]float64{c[0], c[1]}
	}
	g.shapes[eid] = sl
}

// MakeShape returns geometry for the given stops.
func (g *geomCache) MakeShape(stopids ...string) (gotransit.Shape, error) {
	shape := gotransit.Shape{}
	stopline := []float64{} // flatcoords
	for _, stopid := range stopids {
		if geom, ok := g.stops[stopid]; ok {
			stopline = append(stopline, geom[0], geom[1], 0.0)
		} else {
			return shape, fmt.Errorf("stop '%s' not in cache", stopid)
		}
	}
	shape.Geometry = gotransit.NewLineStringFromFlatCoords(stopline)
	shape.Generated = true
	return shape, nil
}

// InterpolateStopTimes uses the cached geometries to interpolate StopTimes.
func (g *geomCache) InterpolateStopTimes(trip gotransit.Trip, stoptimes []gotransit.StopTime) ([]gotransit.StopTime, error) {
	// Check cache; make stopline
	stopline := make([][2]float64, len(stoptimes))
	shapeid := trip.ShapeID
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
		positions = linePositions(shapeline, stopline)
		length := lengthHaversine(shapeline)
		// Check for simple or fallback positions
		if !arePositionsSorted(positions) || len(shapeline) == 0 {
			// log.Debug("positions %f not increasing, falling back to stop positions; shapeline %f stopline %f", positions, shapeline, stopline)
			positions = linePositionsFallback(stopline)
			if !arePositionsSorted(positions) {
				return stoptimes, errors.New("fallback positions not sorted")
			}
			length = lengthHaversine(stopline)
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
		stoptimes[i].ShapeDistTraveled = positions[i] * length
	}
	return InterpolateStopTimes(stoptimes)
}
