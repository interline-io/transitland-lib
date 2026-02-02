package bestpractices

import (
	"fmt"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
	geom "github.com/twpayne/go-geom"
)

// LevelTooFarFromStationError reports when a level geometry centroid is too far
// from stops that reference it.
type LevelTooFarFromStationError struct {
	LevelID  string
	StopID   string
	Distance float64
	bc
}

func (e *LevelTooFarFromStationError) Error() string {
	return fmt.Sprintf(
		"level '%s' geometry centroid is %0.0fm from stop '%s' which references it",
		e.LevelID,
		e.Distance,
		e.StopID,
	)
}

// LevelTooFarFromStationCheck validates that level geometry is near the stops that reference it.
// It implements Filter (to cache parent stations by level_id) and Validator (to check Level entities).
// Uses the shared GeomCache for stop coordinates to avoid duplicating data.
type LevelTooFarFromStationCheck struct {
	maxDist          float64
	geomCache        tlxy.GeomCache      // shared stop geometry cache
	stationsByLevel  map[string][]string // level_id -> parent station IDs (deduplicated)
	checkedLevels    map[string]bool     // track which levels have been checked
	checkedStations  map[string]bool     // track station/level pairs already added
}

// SetGeomCache sets a shared geometry cache for stop coordinates.
func (e *LevelTooFarFromStationCheck) SetGeomCache(g tlxy.GeomCache) {
	e.geomCache = g
}

// Filter caches parent station IDs indexed by level_id.
// Only stores parent station references, not coordinates (uses shared GeomCache for that).
func (e *LevelTooFarFromStationCheck) Filter(ent tt.Entity, emap *tt.EntityMap) error {
	stop, ok := ent.(*gtfs.Stop)
	if !ok {
		return nil
	}

	// Only care about stops that have both a level_id and parent_station
	if stop.LevelID.Val == "" || stop.ParentStation.Val == "" {
		return nil
	}

	// Initialize maps if needed
	if e.stationsByLevel == nil {
		e.stationsByLevel = make(map[string][]string)
	}
	if e.checkedStations == nil {
		e.checkedStations = make(map[string]bool)
	}

	// Deduplicate: only add each station once per level
	key := stop.LevelID.Val + ":" + stop.ParentStation.Val
	if e.checkedStations[key] {
		return nil
	}
	e.checkedStations[key] = true

	e.stationsByLevel[stop.LevelID.Val] = append(e.stationsByLevel[stop.LevelID.Val], stop.ParentStation.Val)
	return nil
}

// Validate checks if a Level's geometry centroid is too far from its parent stations.
func (e *LevelTooFarFromStationCheck) Validate(ent tt.Entity) []error {
	level, ok := ent.(*gtfs.Level)
	if !ok {
		return nil
	}

	// Only check levels with geometry
	if !level.Geometry.Valid {
		return nil
	}

	// Need geomCache to look up station coordinates
	if e.geomCache == nil {
		return nil
	}

	// Initialize tracking map
	if e.checkedLevels == nil {
		e.checkedLevels = make(map[string]bool)
	}

	// Don't check the same level twice
	if e.checkedLevels[level.LevelID.Val] {
		return nil
	}
	e.checkedLevels[level.LevelID.Val] = true

	// Set default max distance (500m)
	if e.maxDist == 0 {
		e.maxDist = 500.0
	}

	// Get parent stations that reference this level
	stations := e.stationsByLevel[level.LevelID.Val]
	if len(stations) == 0 {
		// No stations reference this level, nothing to check
		return nil
	}

	// Calculate the centroid of the level geometry
	centroid, ok := polygonCentroid(level.Geometry)
	if !ok {
		return nil
	}

	var errs []error

	// Check each parent station that has stops referencing this level
	for _, stationID := range stations {
		stationPt := e.geomCache.GetStop(stationID)
		if stationPt.Lon == 0 && stationPt.Lat == 0 {
			continue // Station not in cache
		}

		distance := tlxy.DistanceHaversine(centroid, stationPt)
		if distance > e.maxDist {
			errs = append(errs, &LevelTooFarFromStationError{
				LevelID:  level.LevelID.Val,
				StopID:   stationID,
				Distance: distance,
			})
			// Only report once per level
			break
		}
	}

	return errs
}

// polygonCentroid calculates the centroid of a Polygon or MultiPolygon geometry.
// For MultiPolygon, it returns the centroid of the first polygon.
// Returns the centroid point and whether calculation was successful.
func polygonCentroid(geometry tt.Geometry) (tlxy.Point, bool) {
	if !geometry.Valid || geometry.Val == nil {
		return tlxy.Point{}, false
	}

	var ring []geom.Coord

	switch g := geometry.Val.(type) {
	case *geom.Polygon:
		if g.NumLinearRings() == 0 {
			return tlxy.Point{}, false
		}
		ring = g.LinearRing(0).Coords()
	case *geom.MultiPolygon:
		if g.NumPolygons() == 0 {
			return tlxy.Point{}, false
		}
		poly := g.Polygon(0)
		if poly.NumLinearRings() == 0 {
			return tlxy.Point{}, false
		}
		ring = poly.LinearRing(0).Coords()
	default:
		return tlxy.Point{}, false
	}

	if len(ring) == 0 {
		return tlxy.Point{}, false
	}

	// Calculate centroid as average of exterior ring coordinates
	// (This is a simple approximation; for complex polygons, area-weighted
	// centroid would be more accurate, but this is sufficient for our use case)
	var sumLon, sumLat float64
	for _, coord := range ring {
		sumLon += coord.X()
		sumLat += coord.Y()
	}

	n := float64(len(ring))
	return tlxy.Point{
		Lon: sumLon / n,
		Lat: sumLat / n,
	}, true
}
