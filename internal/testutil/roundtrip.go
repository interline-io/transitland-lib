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
	Reader             gotransit.Reader
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
			if _, ok := ids[k]; !ok {
				t.Errorf("did not find expected entity '%s'", k)
			}
		}
	})
	t.Run("Routes", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Routes() {
			ids[ent.RouteID]++
		}
		if s, exp := msisum(ids), fe.RouteCount; s != exp {
			t.Errorf("got %d expected %d", len(ids), exp)
		}
		for _, k := range fe.ExpectRouteIDs {
			if _, ok := ids[k]; !ok {
				t.Errorf("did not find expected entity '%s'", k)
			}
		}
	})
	t.Run("Trips", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Trips() {
			ids[ent.TripID]++
		}
		if s, exp := msisum(ids), fe.TripCount; s != exp {
			t.Errorf("got %d expected %d", len(ids), exp)
		}
		for _, k := range fe.ExpectTripIDs {
			if _, ok := ids[k]; !ok {
				t.Errorf("did not find expected entity '%s'", k)
			}
		}
	})
	t.Run("Stops", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Stops() {
			ids[ent.StopID]++
		}
		if s, exp := msisum(ids), fe.StopCount; s != exp {
			t.Errorf("got %d expected %d", len(ids), exp)
		}
		for _, k := range fe.ExpectStopIDs {
			if _, ok := ids[k]; !ok {
				t.Errorf("did not find expected entity '%s'", k)
			}
		}
	})
	t.Run("Shapes", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Shapes() {
			ids[ent.ShapeID]++
		}
		if s, exp := msisum(ids), fe.ShapeCount; s != exp {
			t.Errorf("got %d expected %d", len(ids), exp)
		}
		for _, k := range fe.ExpectShapeIDs {
			if _, ok := ids[k]; !ok {
				t.Errorf("did not find expected entity '%s'", k)
			}
		}
	})
	t.Run("Calendars", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Calendars() {
			ids[ent.ServiceID]++
		}
		if s, exp := msisum(ids), fe.CalendarCount; s != exp {
			t.Errorf("got %d expected %d", len(ids), exp)
		}
		for _, k := range fe.ExpectCalendarIDs {
			if _, ok := ids[k]; !ok {
				t.Errorf("did not find expected entity '%s'", k)
			}
		}
	})
	t.Run("CalendarDates", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.CalendarDates() {
			ids[ent.ServiceID]++
		}
		if s, exp := msisum(ids), fe.CalendarDateCount; s != exp {
			t.Errorf("got %d expected %d", len(ids), exp)
		}
		for _, k := range fe.ExpectCalendarIDs {
			if _, ok := ids[k]; !ok {
				t.Errorf("did not find expected entity '%s'", k)
			}
		}
	})
	t.Run("StopTimes", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.StopTimes() {
			ids[ent.TripID]++
		}
		if s, exp := msisum(ids), fe.StopTimeCount; s != exp {
			t.Errorf("got %d expected %d", len(ids), exp)
		}
	})
	t.Run("FareAttributes", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.FareAttributes() {
			ids[ent.FareID]++
		}
		if s, exp := msisum(ids), fe.FareAttributeCount; s != exp {
			t.Errorf("got %d expected %d", len(ids), exp)
		}
		for _, k := range fe.ExpectFareIDs {
			if _, ok := ids[k]; !ok {
				t.Errorf("did not find expected entity '%s'", k)
			}
		}
	})
	t.Run("FareRules", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.FareRules() {
			ids[ent.FareID]++
		}
		if s, exp := msisum(ids), fe.FareRuleCount; s != exp {
			t.Errorf("got %d expected %d", len(ids), exp)
		}
	})
	t.Run("Frequencies", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Frequencies() {
			ids[ent.TripID]++
		}
		if s, exp := msisum(ids), fe.FrequencyCount; s != exp {
			t.Errorf("got %d expected %d", len(ids), exp)
		}
	})
	t.Run("Transfers", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Transfers() {
			ids[ent.FromStopID]++
		}
		if s, exp := msisum(ids), fe.TransferCount; s != exp {
			t.Errorf("got %d expected %d", len(ids), exp)
		}
	})
	t.Run("FeedInnfos", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.FeedInfos() {
			ids[ent.FeedVersion]++
		}
		if s, exp := msisum(ids), fe.FeedInfoCount; s != exp {
			t.Errorf("got %d expected %d", len(ids), exp)
		}
	})
}
