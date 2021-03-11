package xy

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/interline-io/transitland-lib/tl"
	geomxy "github.com/twpayne/go-geom/xy"
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
	stops     map[string][2]float64
	shapes    map[string][][2]float64
	lengths   map[string]float64
}

// NewGeomCache returns an initialized geomCache
func NewGeomCache() *GeomCache {
	return &GeomCache{
		positions: map[string][]float64{},
		stops:     map[string][2]float64{},
		shapes:    map[string][][2]float64{},
		lengths:   map[string]float64{},
	}
}

// AddStop adds a Stop to the geometry cache.
func (g *GeomCache) AddStop(eid string, stop tl.Stop) {
	c := stop.Geometry.FlatCoords()
	g.stops[eid] = [2]float64{c[0], c[1]}
}

// GetStop returns the coordinates for the cached stop.
func (g *GeomCache) GetStop(eid string) [2]float64 {
	return g.stops[eid]
}

// GetShape returns the coordinates for the cached shape.
func (g *GeomCache) GetShape(eid string) [][2]float64 {
	return g.shapes[eid]
}

// AddShape adds a Shape to the geometry cache.
func (g *GeomCache) AddShape(eid string, shape tl.Shape) {
	if !shape.Geometry.Valid {
		return
	}
	sl := make([][2]float64, shape.Geometry.NumCoords())
	for i, c := range shape.Geometry.Coords() {
		sl[i] = [2]float64{c[0], c[1]}
	}
	g.shapes[eid] = sl
}

// AddSimplifiedShape adds a simplified Shape to the geometry cache.
func (g *GeomCache) AddSimplifiedShape(eid string, shape tl.Shape, threshold float64) {
	if !shape.Geometry.Valid {
		return
	}
	pnts := shape.Geometry.FlatCoords()
	fmt.Println("shape input length:", eid, len(pnts))
	stride := shape.Geometry.Stride()
	ii := geomxy.SimplifyFlatCoords(pnts, threshold, stride)
	for i, j := range ii {
		if i == j*stride {
			continue
		}
		pnts[i*stride], pnts[i*stride+1] = pnts[j*stride], pnts[j*stride+1]
	}
	pnts = pnts[:len(ii)*stride]
	fmt.Println("shape output length:", eid, len(pnts))
	sl := make([][2]float64, len(pnts)/stride)
	for i := 0; i < len(pnts)-stride+1; i += stride {
		sl[i/stride] = [2]float64{pnts[i], pnts[i+1]}
	}
	g.shapes[eid] = sl
}

// MakeShape returns geometry for the given stops.
func (g *GeomCache) MakeShape(stopids ...string) (tl.Shape, error) {
	shape := tl.Shape{}
	stopline := []float64{} // flatcoords
	for _, stopid := range stopids {
		if geom, ok := g.stops[stopid]; ok {
			stopline = append(stopline, geom[0], geom[1], 0.0)
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
	stopline := make([][2]float64, len(stoptimes))
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
		stoptimes[i].ShapeDistTraveled = sql.NullFloat64{Valid: true, Float64: positions[i] * length}
	}
	return InterpolateStopTimes(stoptimes)
}
