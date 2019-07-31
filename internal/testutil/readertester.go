package testutil

import (
	"testing"

	"github.com/interline-io/gotransit"
)

func getfn(ent gotransit.Entity) string {
	return ent.Filename()
}

func msisum(m map[string]int) int {
	count := 0
	keys := []string{}
	for k, v := range m {
		keys = append(keys, k)
		count += v
	}
	return count
}

// AllEntities iterates through all Reader entities, calling the specified callback.
func AllEntities(reader gotransit.Reader, cb func(gotransit.Entity)) {
	for ent := range reader.Agencies() {
		cb(&ent)
	}
	for ent := range reader.Routes() {
		cb(&ent)
	}
	for ent := range reader.Trips() {
		cb(&ent)
	}
	for ent := range reader.Stops() {
		cb(&ent)
	}
	for ent := range reader.Shapes() {
		cb(&ent)
	}
	for ent := range reader.Calendars() {
		cb(&ent)
	}
	for ent := range reader.CalendarDates() {
		cb(&ent)
	}
	for ent := range reader.StopTimes() {
		cb(&ent)
	}
	for ent := range reader.FareAttributes() {
		cb(&ent)
	}
	for ent := range reader.FareRules() {
		cb(&ent)
	}
	for ent := range reader.Frequencies() {
		cb(&ent)
	}
	for ent := range reader.Transfers() {
		cb(&ent)
	}
	for ent := range reader.FeedInfos() {
		cb(&ent)
	}
}

// ReaderTester contains information about the number and types of identities expected in a Reader.
type ReaderTester struct {
	URL       string
	Counts    map[string]int
	EntityIDs map[string][]string
}

// Benchmark checks the Reader against the expected values and records errors.
func (fe ReaderTester) Benchmark(b *testing.B, reader gotransit.Reader) {
	ids := map[string]map[string]int{}
	add := func(ent gotransit.Entity) {
		ent.SetID(0) // TODO: This is a HORRIBLE UGLY HACK :( it sets db ID to zero value to get GTFS ID.
		m, ok := ids[ent.Filename()]
		if !ok {
			m = map[string]int{}
		}
		m[ent.EntityID()]++
		ids[ent.Filename()] = m
	}
	check := func(fn string, gotids map[string]int) {
		s := msisum(gotids)
		if exp, ok := fe.Counts[fn]; ok && s != exp {
			b.Errorf("got %d expected %d", s, exp)
		}
		for _, k := range fe.EntityIDs[fn] {
			if _, ok := gotids[k]; !ok {
				b.Errorf("did not find expected entity %s '%s'", fn, k)
			}
		}
	}
	AllEntities(reader, add)
	for k, v := range ids {
		check(k, v)
	}
}

// Test checks the Reader against the expected values and records errors.
func (fe ReaderTester) Test(t *testing.T, reader gotransit.Reader) {
	check := func(fn string, gotids map[string]int) {
		s := msisum(gotids)
		if exp, ok := fe.Counts[fn]; ok && s != exp {
			t.Errorf("got %d expected %d", s, exp)
		}
		for _, k := range fe.EntityIDs[fn] {
			if _, ok := gotids[k]; !ok {
				t.Errorf("did not find expected entity '%s'", k)
			}
		}
	}
	t.Run("Agencies", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Agencies() {
			ids[ent.AgencyID]++
		}
		check(getfn(&gotransit.Agency{}), ids)
	})
	t.Run("Routes", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Routes() {
			ids[ent.RouteID]++
		}
		check(getfn(&gotransit.Route{}), ids)
	})
	t.Run("Trips", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Trips() {
			ids[ent.TripID]++
		}
		check(getfn(&gotransit.Trip{}), ids)
	})
	t.Run("Stops", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Stops() {
			ids[ent.StopID]++
		}
		check(getfn(&gotransit.Stop{}), ids)
	})
	t.Run("Shapes", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Shapes() {
			ids[ent.ShapeID]++
		}
		check(getfn(&gotransit.Shape{}), ids)
	})
	t.Run("Calendars", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Calendars() {
			ids[ent.ServiceID]++
		}
		check(getfn(&gotransit.Calendar{}), ids)
	})
	t.Run("CalendarDates", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.CalendarDates() {
			ids[ent.ServiceID]++
		}
		check(getfn(&gotransit.CalendarDate{}), ids)
	})
	t.Run("StopTimes", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.StopTimes() {
			ids[ent.TripID]++
		}
		check(getfn(&gotransit.StopTime{}), ids)
	})
	t.Run("FareAttributes", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.FareAttributes() {
			ids[ent.FareID]++
		}
		check(getfn(&gotransit.FareAttribute{}), ids)
	})
	t.Run("FareRules", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.FareRules() {
			ids[ent.FareID]++
		}
		check(getfn(&gotransit.FareRule{}), ids)
	})
	t.Run("Frequencies", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Frequencies() {
			ids[ent.TripID]++
		}
		check(getfn(&gotransit.Frequency{}), ids)
	})
	t.Run("Transfers", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.Transfers() {
			ids[ent.FromStopID]++
		}
		check(getfn(&gotransit.Transfer{}), ids)
	})
	t.Run("FeedInfos", func(t *testing.T) {
		ids := map[string]int{}
		for ent := range reader.FeedInfos() {
			ids[ent.FeedVersion]++
		}
		check(getfn(&gotransit.FeedInfo{}), ids)
	})
}
