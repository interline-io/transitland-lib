package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/internal/xy"
	"github.com/interline-io/transitland-lib/tl"
)

// FastTravelError reports when reasonable maximum speeds have been exceeded for at least 30 seconds.
type FastTravelError struct {
	TripID       string
	StopSequence int
	FromStopID   string
	ToStopID     string
	Distance     float64
	Time         int
	Speed        float64
	SpeedLimit   float64
	bc
}

func newFastTravelError(trip string, seq int, from string, to string, t int, distance float64, speed float64, limit float64) *FastTravelError {
	return &FastTravelError{
		TripID:     trip,
		FromStopID: from,
		ToStopID:   to,
		Time:       t,
		Distance:   distance,
		Speed:      speed,
		SpeedLimit: limit,
	}
}

func (e *FastTravelError) Error() string {
	return fmt.Sprintf(
		"trip '%s' stop_sequence %d traveled from stop '%s' to stop '%s' in %d seconds, a distance of %0.2f m and speed of %0.2f km/h where %0.2f km/h is the assumed maximum for this route type",
		e.TripID,
		e.StopSequence,
		e.FromStopID,
		e.ToStopID,
		e.Time,
		e.Distance,
		e.Speed,
		e.SpeedLimit,
	)
}

var maxSpeeds = map[int]float64{
	0:  200, // tram
	1:  200, // metro
	2:  500, // rail
	3:  200, // bus
	4:  100, // ferry
	5:  100, // cable car
	6:  100, // gondola
	7:  100, // funicular
	11: 100, // trolleybus
	12: 100, // monorail
}

// StopTimeFastTravelCheck checks for FastTravelErrors.
type StopTimeFastTravelCheck struct {
	routeTypes map[string]int     // keep track of route_types
	stopDist   map[string]float64 // cache stop-to-stop distances
	geomCache  *xy.GeomCache      // share with copier
}

// SetGeomCache sets a shared geometry cache.
func (e *StopTimeFastTravelCheck) SetGeomCache(g *xy.GeomCache) {
	e.geomCache = g
}

// Validate .
func (e *StopTimeFastTravelCheck) Validate(ent tl.Entity) []error {
	if v, ok := ent.(*tl.Route); ok {
		if e.routeTypes == nil {
			e.routeTypes = map[string]int{}
		}
		e.routeTypes[v.RouteID] = v.RouteType
	}
	// Use stop to stop distances, shape_dist_traveled is not reliable.
	trip, ok := ent.(*tl.Trip)
	if !ok || len(trip.StopTimes) < 2 {
		return nil
	}
	if e.stopDist == nil {
		e.stopDist = map[string]float64{}
	}
	maxspeed := 200.0 // default max speed
	if rtype, ok := e.routeTypes[trip.RouteID]; ok {
		if m, ok := maxSpeeds[rtype]; ok {
			maxspeed = m
		}
	}
	// todo: cache for trip pattern?
	var errs []error
	s1 := trip.StopTimes[0].StopID
	t := trip.StopTimes[0].DepartureTime
	for i := 1; i < len(trip.StopTimes); i++ {
		s2 := trip.StopTimes[i].StopID
		key := s1 + ":" + s2 // todo: use a real separator...
		dx, ok := e.stopDist[key]
		if !ok {
			g1, g2 := e.geomCache.GetStop(s1), e.geomCache.GetStop(s2)
			dx = 0
			// Only consider this edge if valid geoms.
			if (g1[0] != 0 && g1[1] != 0) && (g2[0] != 0 && g2[1] != 0) {
				dx = xy.DistanceHaversine(g1[0], g1[1], g2[0], g2[1])
			}
			e.stopDist[key] = dx
			e.stopDist[s2+":"+s1] = dx
		}
		dt := trip.StopTimes[i].ArrivalTime - t
		speed := (dx / 1000.0) / (float64(dt) / 3600.0)
		if dt > 30 && speed > maxspeed {
			errs = append(errs, newFastTravelError(trip.TripID, trip.StopTimes[i].StopSequence, s1, s2, dt, dx, speed, maxspeed))
		}
		s1 = s2
		t = trip.StopTimes[i].DepartureTime
	}
	return errs
}
