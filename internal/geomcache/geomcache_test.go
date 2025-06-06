package geomcache

import (
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tlcsv"
)

func TestGeomCache(t *testing.T) {
	r, err := tlcsv.NewReader(testutil.ExampleDir.URL)
	if err != nil {
		t.Error(err)
	}
	r.Open()
	defer r.Close()
	trips := map[string]gtfs.Trip{}
	count := 1
	for trip := range r.Trips() {
		trip := trip
		trip.StopPatternID.SetInt(count)
		trips[trip.TripID.Val] = trip
		count++
	}
	cache := NewGeomCache()
	for shapeEnts := range r.ShapesByShapeID() {
		e := service.NewShapeLineFromShapes(shapeEnts)
		lm := e.Geometry.ToLineM()
		cache.AddShapeGeom(e.ShapeID.Val, lm.Coords, lm.Data)
	}
	for e := range r.Stops() {
		cache.AddStopGeom(e.StopID.Val, e.ToPoint())
	}
	for stoptimes := range r.StopTimesByTripID() {
		trip := trips[stoptimes[0].TripID.Val]
		trip.StopTimes = stoptimes
		stoptimes2, err := cache.InterpolateStopTimes(&trip)
		if err != nil {
			t.Error(err)
		}
		if len(stoptimes) != len(stoptimes2) {
			t.Error("unequal length")
		}
	}
	// check that we had cache hits
	if x := len(cache.stopPositions); x < 9 {
		t.Errorf("expected at least %d cached trip journeys, got %d", 9, x)
	}
}
