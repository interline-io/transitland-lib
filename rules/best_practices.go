package rules

import (
	"strconv"

	"github.com/interline-io/transitland-lib/internal/xy"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/mmcloughlin/geohash"
)

///////////////////

// NoScheduledServiceCheck checks that a service contains at least one scheduled day, otherwise returns a warning.
type NoScheduledServiceCheck struct{}

// ValidateEntity .
func (e *NoScheduledServiceCheck) ValidateEntity(ent tl.Entity) ([]error, []error) {
	v, ok := ent.(*tl.Service)
	if !ok {
		return nil, nil
	}
	if v.HasAtLeastOneDay() {
		return nil, nil
	}
	return nil, []error{&causes.NoScheduledServiceError{}}
}

///////////////////

// StopTooFarCheck checks if two related stops are >1km away.
type StopTooFarCheck struct {
	geoms   map[string]*tl.Point // regularize and use copier geomCache?
	maxdist float64
}

// ValidateEntity .
func (e *StopTooFarCheck) ValidateEntity(ent tl.Entity) ([]error, []error) {
	e.maxdist = 1000.0
	if e.geoms == nil {
		e.geoms = map[string]*tl.Point{}
	}
	v, ok := ent.(*tl.Stop)
	if !ok {
		return nil, nil
	}
	var errs []error
	coords := v.Geometry.Coords()
	newp := tl.NewPoint(coords[0], coords[1]) // copy
	e.geoms[v.StopID] = &newp
	if v.ParentStation.Key == "" {
		return nil, nil
	}
	// Check if parent stop is >1km
	if pgeom, ok := e.geoms[v.ParentStation.Key]; ok {
		// if not ok, then it's a parent error and out of scope for this check
		d := xy.DistanceHaversinePoint(coords, pgeom.Coords())
		if d > e.maxdist {
			errs = append(errs, causes.NewStopTooFarError())
		}
	}
	return nil, errs
}

///////////////////

type stopPoint struct {
	id  string
	lat float64
	lon float64
}

// StopTooCloseCheck checks if two stops are within 1m
type StopTooCloseCheck struct {
	geoms   map[string][]*stopPoint
	maxdist float64
}

// ValidateEntity .
func (e *StopTooCloseCheck) ValidateEntity(ent tl.Entity) ([]error, []error) {
	e.maxdist = 1.0
	if e.geoms == nil {
		e.geoms = map[string][]*stopPoint{}
	}
	v, ok := ent.(*tl.Stop)
	// This only checks location_type == 0 and no parent
	if !ok || v.ParentStation.Key != "" || v.LocationType != 0 || !v.Geometry.Valid {
		return nil, nil
	}
	// Use geohash for fast neighbor search; precision = 9 is approx 5m x 5m at the equator.
	coords := v.Geometry.Coords()
	if len(coords) < 2 {
		return nil, nil
	}
	var errs []error
	gh := geohash.EncodeWithPrecision(coords[0], coords[1], 9)
	neighbors := geohash.Neighbors(gh)
	neighbors = append(neighbors, gh)
	g := stopPoint{id: v.StopID, lat: coords[0], lon: coords[1]}
	for _, neighbor := range neighbors {
		if hits, ok := e.geoms[neighbor]; ok {
			for _, hit := range hits {
				d := xy.DistanceHaversine(g.lon, g.lat, hit.lon, hit.lat)
				if d < e.maxdist {
					errs = append(errs, causes.NewStopTooCloseError(hit.id, d))
				}
			}
		}
	}
	// add to index
	e.geoms[gh] = append(e.geoms[gh], &g)
	return nil, errs
}

///////////////////

// StopTooFarFromShapeCheck checks if a stop is more than 100m from an associated shape.
type StopTooFarFromShapeCheck struct {
	maxdist   float64
	geomCache *xy.GeomCache // share stop/shape geometry cache with copier
	checked   map[string]map[string]bool
}

// SetGeomCache sets a shared geometry cache.
func (e *StopTooFarFromShapeCheck) SetGeomCache(g *xy.GeomCache) {
	e.geomCache = g
}

// ValidateEntity .
func (e *StopTooFarFromShapeCheck) ValidateEntity(ent tl.Entity) ([]error, []error) {
	// An initial approach used geohashes to check shape <-> stop as an initial filter, but it turns
	// out in practice that just checking directly is almost exactly the same speed.
	// Even the largest feeds are only a few tens of thousands of comparisons. Just keep track
	// of comparisons that have already been made and it's fine.
	e.maxdist = 100.0
	v, ok := ent.(*tl.Trip)
	if !ok {
		return nil, nil
	}
	if e.checked == nil {
		e.checked = map[string]map[string]bool{}
	}
	shapeid := v.ShapeID.Key
	if shapeid == "" || len(v.StopTimes) == 0 {
		return nil, nil
	}
	if e.checked[shapeid] == nil {
		e.checked[shapeid] = map[string]bool{}
	}
	var errs []error
	for _, st := range v.StopTimes {
		// Check the cache
		if e.checked[shapeid][st.StopID] {
			continue
		}
		e.checked[shapeid][st.StopID] = true
		g := e.geomCache.GetStop(st.StopID)
		sgeom := e.geomCache.GetShape(shapeid)
		nearest, _ := xy.LineClosestPoint(sgeom, g)
		distance := xy.DistanceHaversine(g[0], g[1], nearest[0], nearest[1])
		if distance > e.maxdist {
			errs = append(errs, causes.NewStopTooFarFromShapeError(st.StopID, shapeid, distance))
		}
	}
	return nil, errs
}

///////////////////

// DuplicateRouteNameCheck checks for routes of the same agency with identical route_long_names.
type DuplicateRouteNameCheck struct {
	names map[string]int
}

// ValidateEntity .
func (e *DuplicateRouteNameCheck) ValidateEntity(ent tl.Entity) ([]error, []error) {
	v, ok := ent.(*tl.Route)
	if !ok {
		return nil, nil
	}
	if e.names == nil {
		e.names = map[string]int{}
	}
	key := v.AgencyID + ":" + strconv.Itoa(v.RouteType) + ":" + v.RouteLongName // todo: use a real separator
	if _, ok := e.names[key]; ok {
		return nil, []error{causes.NewValidationWarning("route_long_name", "duplicate route_long_name in same agency_id,route_type")}
	}
	e.names[key]++
	return nil, nil
}

///////////////////

// DuplicateFareRuleCheck checks for fare_rules that are effectively identical.
type DuplicateFareRuleCheck struct {
	rules map[string]int
}

// ValidateEntity .
func (e *DuplicateFareRuleCheck) ValidateEntity(ent tl.Entity) ([]error, []error) {
	v, ok := ent.(*tl.FareRule)
	if !ok {
		return nil, nil
	}
	if e.rules == nil {
		e.rules = map[string]int{}
	}
	key := v.RouteID.Key + ":" + v.OriginID + ":" + v.DestinationID + ":" + v.ContainsID
	if _, ok := e.rules[key]; ok {
		return nil, []error{causes.NewValidationWarning("origin_id", "duplicate fare_rule")}
	}
	e.rules[key]++
	return nil, nil
}

////////////////////

type freqValue struct {
	start int
	end   int
}

// FrequencyOverlapCheck checks that frequencies for the same trip do not overlap.
type FrequencyOverlapCheck struct {
	freqs map[string][]*freqValue
}

// ValidateEntity .
func (e *FrequencyOverlapCheck) ValidateEntity(ent tl.Entity) ([]error, []error) {
	v, ok := ent.(*tl.Frequency)
	if !ok {
		return nil, nil
	}
	if e.freqs == nil {
		e.freqs = map[string][]*freqValue{}
	}
	var errs []error
	tf := freqValue{
		start: v.StartTime.Seconds,
		end:   v.EndTime.Seconds,
	}
	for _, hit := range e.freqs[v.TripID] {
		if !(tf.start >= hit.end || tf.end <= hit.start) {
			errs = append(errs, causes.NewValidationWarning("start_time", "overlaps with another frequency for same trip"))
		}
	}
	e.freqs[v.TripID] = append(e.freqs[v.TripID], &tf)
	return nil, errs
}
