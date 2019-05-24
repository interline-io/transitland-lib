package testutil

import (
	"fmt"
	"testing"

	"github.com/interline-io/gotransit"
)

func compareMap(t *testing.T, result map[string]int, expect map[string]int) {
	for k, v := range expect {
		if i := result[k]; v != i {
			t.Error("expeced", k, "=", i)
		}
	}
}

func compareMapLen(t *testing.T, result map[string]int, expect int) {
	if len(result) != expect {
		t.Error("expected", expect, "keys, got", len(result))
	}
}

// ReaderTester checks implementations of Reader interface against the Example Feed.
func ReaderTester(reader gotransit.Reader, t *testing.T) {
	t.Run("StopTimesByTripID", func(t *testing.T) { readerTesterStopTimesByTripID(reader, t) })
	t.Run("ShapesByShapeID", func(t *testing.T) { readerTesterShapesByShapeID(reader, t) })
	t.Run("ShapeLinesByShapeID", func(t *testing.T) { readerTesterShapeLinesByShapeID(reader, t) })
	t.Run("Stops", func(t *testing.T) { readerTesterStops(reader, t) })
	t.Run("StopTimes", func(t *testing.T) { readerTesterStopTimes(reader, t) })
	t.Run("Agencies", func(t *testing.T) { readerTesterAgencies(reader, t) })
	t.Run("Calendars", func(t *testing.T) { readerTesterCalendars(reader, t) })
	t.Run("CalendarDates", func(t *testing.T) { readerTesterCalendarDates(reader, t) })
	t.Run("FareAttributes", func(t *testing.T) { readerTesterFareAttributes(reader, t) })
	t.Run("FareRules", func(t *testing.T) { readerTesterFareRules(reader, t) })
	t.Run("FeedInfos", func(t *testing.T) { readerTesterFeedInfos(reader, t) })
	t.Run("Frequencies", func(t *testing.T) { readerTesterFrequencies(reader, t) })
	t.Run("Routes", func(t *testing.T) { readerTesterRoutes(reader, t) })
	t.Run("Trips", func(t *testing.T) { readerTesterTrips(reader, t) })
	t.Run("Transfers", func(t *testing.T) { readerTesterTransfers(reader, t) })
	t.Run("Shapes", func(t *testing.T) { readerTesterShapes(reader, t) })
	t.Run("ReadEntities", func(t *testing.T) { readerTesterReadEntities(reader, t) })
}

func readerTesterReadEntities(reader gotransit.Reader, t *testing.T) {
	out := make(chan gotransit.Route, 1000)
	reader.ReadEntities(out)
	count := map[string]int{}
	for ent := range out {
		count[ent.EntityID()]++
	}
	expect := 5
	if c := len(count); c != expect {
		t.Errorf("expected %d entities, got %d", c, expect)
	}

}

func readerTesterStopTimesByTripID(reader gotransit.Reader, t *testing.T) {
	result := map[string]int{}
	expectgroups := 11
	expectcount := 28
	count := 0
	for ents := range reader.StopTimesByTripID() {
		if len(ents) > 0 {
			result[ents[0].TripID] = len(ents)
			count = count + len(ents)
		}
	}
	if count != expectcount {
		t.Error(fmt.Sprintf("expected %d total, got %d", expectcount, count))
	}
	if z := len(result); z != expectgroups {
		t.Error(fmt.Sprintf("expected %d groups, got %d", expectgroups, z))
	}
}

func readerTesterShapesByShapeID(reader gotransit.Reader, t *testing.T) {
	result := map[string]int{}
	expectgroups := 3
	count := 0
	for ents := range reader.ShapesByShapeID() {
		result[ents[0].ShapeID] = len(ents)
		count = count + len(ents)
	}
	if z := len(result); z != expectgroups {
		t.Error(fmt.Sprintf("expected %d groups, got %d", expectgroups, z))
	}
}

func readerTesterShapeLinesByShapeID(reader gotransit.Reader, t *testing.T) {
	result := map[string]int{}
	total := 0
	expectgroups := 3
	totalexpect := 9
	for ent := range reader.ShapeLinesByShapeID() {
		result[ent.ShapeID]++
		total += ent.Geometry.NumCoords()
	}
	if z := len(result); z != expectgroups {
		t.Error(fmt.Sprintf("expected %d groups, got %d", expectgroups, z))
	}
	if total != totalexpect {
		t.Errorf("expected %d got %d", totalexpect, total)
	}
}

///////////////////////////
// Entity reader tests
///////////////////////////

func readerTesterStops(reader gotransit.Reader, t *testing.T) {
	m := map[string]int{}
	for ent := range reader.Stops() {
		m[ent.StopID]++
	}
	compareMapLen(t, m, 9)
	expect := map[string]int{"NADAV": 1, "AMV": 1, "BULLFROG": 1, "DADAN": 1}
	compareMap(t, m, expect)
}

func readerTesterStopTimes(reader gotransit.Reader, t *testing.T) {
	result := map[string]int{}
	expectgroups := 9
	expectcount := 28
	count := 0
	for ent := range reader.StopTimes() {
		result[ent.StopID]++
		count++
	}
	if count != expectcount {
		t.Error(fmt.Sprintf("expected %d total, got %d", expectcount, count))
	}
	if z := len(result); z != expectgroups {
		t.Error(fmt.Sprintf("expected %d groups, got %d", expectgroups, z))
	}
}

func readerTesterAgencies(reader gotransit.Reader, t *testing.T) {
	m := map[string]int{}
	for stop := range reader.Agencies() {
		m[stop.AgencyID]++
	}
	if _, ok := m["DTA"]; !ok {
		t.Error("expected key DTA")
	}
}

func readerTesterCalendars(reader gotransit.Reader, t *testing.T) {
	m := map[string]int{}
	for e := range reader.Calendars() {
		m[e.ServiceID]++
	}
	expected := map[string]int{"FULLW": 1}
	for k, v := range expected {
		if i := m[k]; v != i {
			t.Error("expeced", k, "=", i)
		}
	}
}

func readerTesterCalendarDates(reader gotransit.Reader, t *testing.T) {
	m := map[string]int{}
	for e := range reader.CalendarDates() {
		m[e.ServiceID]++
	}
	// handle noralization
	serviceids := map[string]string{}
	for e := range reader.Calendars() {
		serviceids[e.ServiceID] = e.EntityID()
	}
	expected := map[string]int{"FULLW": 1}
	for k, v := range expected {
		sid := serviceids[k]
		if i := m[sid]; v != i {
			t.Error("expeced", k, "=", i, "got", v)
		}
	}
}

func readerTesterFareAttributes(reader gotransit.Reader, t *testing.T) {
	m := map[string]int{}
	for e := range reader.FareAttributes() {
		m[e.FareID]++
	}
	expected := map[string]int{"p": 1, "a": 1}
	for k, v := range expected {
		if i := m[k]; v != i {
			t.Error("expeced", k, "=", i)
		}
	}
}

func readerTesterFareRules(reader gotransit.Reader, t *testing.T) {
	m := map[string]int{}
	for e := range reader.FareRules() {
		m[e.FareID]++
	}
	// handle noralization
	fareids := map[string]string{}
	for e := range reader.FareAttributes() {
		fareids[e.FareID] = e.EntityID()
	}
	expected := map[string]int{"p": 3, "a": 1}
	for k, v := range expected {
		fid := fareids[k]
		if i := m[fid]; v != i {
			t.Error("expeced", k, "=", i)
		}
	}
}

func readerTesterFeedInfos(reader gotransit.Reader, t *testing.T) {
	m := map[string]int{}
	for e := range reader.FeedInfos() {
		m[e.FeedPublisherName]++
	}
	expected := map[string]int{"Google": 1}
	for k, v := range expected {
		if i := m[k]; v != i {
			t.Error("expeced", k, "=", i)
		}
	}
}
func readerTesterFrequencies(reader gotransit.Reader, t *testing.T) {
	result := map[string]int{}
	expectgroups := 3
	expectcount := 11
	count := 0
	for ent := range reader.Frequencies() {
		result[ent.TripID]++
		count++
	}
	if count != expectcount {
		t.Error(fmt.Sprintf("expected %d total, got %d", expectcount, count))
	}
	if z := len(result); z != expectgroups {
		t.Error(fmt.Sprintf("expected %d groups, got %d", expectgroups, z))
	}
}

func readerTesterRoutes(reader gotransit.Reader, t *testing.T) {
	m := map[string]int{}
	for e := range reader.Routes() {
		m[e.RouteID]++
	}
	expected := map[string]int{"BFC": 1, "STBA": 1, "AAMV": 1}
	for k, v := range expected {
		if i := m[k]; v != i {
			t.Error("expeced", k, "=", i)
		}
	}
}

func readerTesterTrips(reader gotransit.Reader, t *testing.T) {
	m := map[string]int{}
	for e := range reader.Trips() {
		m[e.TripID]++
	}
	expected := map[string]int{"AB1": 1, "STBA": 1, "AAMV1": 1}
	for k, v := range expected {
		if i := m[k]; v != i {
			t.Error("expeced", k, "=", i)
		}
	}
}

// TODO: example feed needs transfers
func readerTesterTransfers(reader gotransit.Reader, t *testing.T) {}

// TODO: example feed needs shapes
func readerTesterShapes(reader gotransit.Reader, t *testing.T) {}
