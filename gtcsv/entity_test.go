package gtcsv

import (
	"testing"

	"github.com/interline-io/gotransit/internal/testutil"
)

func TestEntities_Agency(t *testing.T) {
	r, _ := NewReader("../testdata/bad-entities")
	for ent := range r.Agencies() {
		testutil.CheckEntityErrors(&ent, t)
	}
}

func TestEntities_Routes(t *testing.T) {
	r, _ := NewReader("../testdata/bad-entities")
	for ent := range r.Routes() {
		testutil.CheckEntityErrors(&ent, t)
	}
}

func TestEntities_Stop(t *testing.T) {
	r, _ := NewReader("../testdata/bad-entities")
	for ent := range r.Stops() {
		testutil.CheckEntityErrors(&ent, t)
	}
}

func TestEntities_Trip(t *testing.T) {
	r, _ := NewReader("../testdata/bad-entities")
	for ent := range r.Trips() {
		testutil.CheckEntityErrors(&ent, t)
	}
}

func TestEntities_StopTime(t *testing.T) {
	r, _ := NewReader("../testdata/bad-entities")
	for ent := range r.StopTimes() {
		testutil.CheckEntityErrors(&ent, t)
	}
}

func TestEntities_Calendar(t *testing.T) {
	r, _ := NewReader("../testdata/bad-entities")
	for ent := range r.Calendars() {
		testutil.CheckEntityErrors(&ent, t)
	}
}

func TestEntities_CalendarDate(t *testing.T) {
	r, _ := NewReader("../testdata/bad-entities")
	for ent := range r.CalendarDates() {
		testutil.CheckEntityErrors(&ent, t)
	}
}

func TestEntities_FareAttribute(t *testing.T) {
	r, _ := NewReader("../testdata/bad-entities")
	for ent := range r.FareAttributes() {
		testutil.CheckEntityErrors(&ent, t)
	}
}

func TestEntities_FareRule(t *testing.T) {
	r, _ := NewReader("../testdata/bad-entities")
	for ent := range r.FareRules() {
		testutil.CheckEntityErrors(&ent, t)
	}
}

func TestEntities_FeedInfo(t *testing.T) {
	r, _ := NewReader("../testdata/bad-entities")
	for ent := range r.FeedInfos() {
		testutil.CheckEntityErrors(&ent, t)
	}
}

func TestEntities_Shape(t *testing.T) {
	r, _ := NewReader("../testdata/bad-entities")
	for ent := range r.Shapes() {
		testutil.CheckEntityErrors(&ent, t)
	}
}

func TestEntities_Transfer(t *testing.T) {
	r, _ := NewReader("../testdata/bad-entities")
	for ent := range r.Transfers() {
		testutil.CheckEntityErrors(&ent, t)
	}
}

func TestEntities_Frequency(t *testing.T) {
	r, _ := NewReader("../testdata/bad-entities")
	for ent := range r.Frequencies() {
		testutil.CheckEntityErrors(&ent, t)
	}
}
