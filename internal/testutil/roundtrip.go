package testutil

import (
	"testing"

	"github.com/interline-io/gotransit"
)

func msisum(m map[string]int) int {
	count := 0
	for _, v := range m {
		count += v
	}
	return count
}

type FeedExpect struct {
	URL                string
	AgencyCount        int
	RouteCount         int
	TripCount          int
	StopCount          int
	StopTimeCount      int
	ShapeCount         int
	CalendarCount      int
	CalendarDateCount  int
	FeedInfoCount      int
	FareRuleCount      int
	FareAttributeCount int
	FrequencyCount     int
	TransferCount      int
	ExpectAgencyIDs    []string
	ExpectRouteIDs     []string
	ExpectTripIDs      []string
	ExpectStopIDs      []string
	ExpectShapeIDs     []string
	ExpectCalendarIDs  []string
	ExpectFareIDs      []string
}

func TestReader(t *testing.T, fe FeedExpect, reader gotransit.Reader) {
	t.Run("Agencies", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Agencies() {
			ids[ent.AgencyID]++
		}
		if s, exp := msisum(ids), fe.AgencyCount; s != exp {
			t.Errorf("got %d expected %d", s, exp)
		}
		for _, k := range fe.ExpectAgencyIDs {
			t.Errorf("did not find expected entity '%s'", k)
		}
	})
	t.Run("Routes", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Routes() {
			ids[ent.RouteID]++
		}
		if exp := fe.RouteCount; len(ids) != exp {
			t.Errorf("got %d expected %d", len(ids), exp)
		}
		for _, k := range fe.ExpectRouteIDs {
			t.Errorf("did not find expected entity '%s'", k)
		}
	})
	t.Run("Trips", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Trips() {
			ids[ent.TripID]++
		}
		if exp := fe.TripCount; len(ids) != exp {
			t.Errorf("got %d expected %d", len(ids), exp)
		}
		for _, k := range fe.ExpectTripIDs {
			t.Errorf("did not find expected entity '%s'", k)
		}
	})
	t.Run("Stops", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Stops() {
			ids[ent.StopID]++
		}
		if exp := fe.StopCount; len(ids) != exp {
			t.Errorf("got %d expected %d", len(ids), exp)
		}
		for _, k := range fe.ExpectTripIDs {
			t.Errorf("did not find expected entity '%s'", k)
		}
	})
	t.Run("Shapes", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Shapes() {
			ids[ent.ShapeID]++
		}
		if exp := fe.ShapeCount; len(ids) != exp {
			t.Errorf("got %d expected %d", len(ids), exp)
		}
		for _, k := range fe.ExpectShapeIDs {
			t.Errorf("did not find expected entity '%s'", k)
		}
	})
	t.Run("Calendars", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Calendars() {
			ids[ent.ServiceID]++
		}
		if exp := fe.CalendarCount; len(ids) != exp {
			t.Errorf("got %d expected %d", len(ids), exp)
		}
		for _, k := range fe.ExpectCalendarIDs {
			t.Errorf("did not find expected entity '%s'", k)
		}
	})
	t.Run("FareAttributes", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.FareAttributes() {
			ids[ent.FareID]++
		}
		if exp := fe.FareAttributeCount; len(ids) != exp {
			t.Errorf("got %d expected %d", len(ids), exp)
		}
		for _, k := range fe.ExpectFareIDs {
			t.Errorf("did not find expected entity '%s'", k)
		}
	})
	// t.Run("FareRules", func(t *testing.T) {
	// 	ids := map[string]int{}
	// 	for ent := range reader.FareRules() {
	// 		ids[ent.FareID]++
	// 	}
	// 	if exp := fe.FareAttributeCount; len(ids) != exp {
	// 		t.Errorf("got %d expected %d", len(ids), exp)
	// 	}
	// 	for _, k := range fe.ExpectFareIDs {
	// 		t.Errorf("did not find expected entity '%s'", k)
	// 	}
	// })
}
