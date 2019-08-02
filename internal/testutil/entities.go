package testutil

import (
	"testing"

	"github.com/interline-io/gotransit"
)

// TestEntityErrors checks that all expected Entity errors are present.
func TestEntityErrors(t *testing.T, r gotransit.Reader) {
	t.Run("Agencies", func(t *testing.T) {
		for ent := range r.Agencies() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("Stops", func(t *testing.T) {
		for ent := range r.Stops() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("Routes", func(t *testing.T) {
		for ent := range r.Routes() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("Trips", func(t *testing.T) {
		for ent := range r.Trips() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("StopTimes", func(t *testing.T) {
		for ent := range r.StopTimes() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("Calendars", func(t *testing.T) {
		for ent := range r.Calendars() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("CalendarDates", func(t *testing.T) {
		for ent := range r.CalendarDates() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("FareAttributes", func(t *testing.T) {
		for ent := range r.FareAttributes() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("FareRules", func(t *testing.T) {
		for ent := range r.FareRules() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("FeedInfos", func(t *testing.T) {
		for ent := range r.FeedInfos() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("Shapes", func(t *testing.T) {
		for ent := range r.Shapes() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("Transfer", func(t *testing.T) {
		for ent := range r.Transfers() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("Frequency", func(t *testing.T) {
		for ent := range r.Frequencies() {
			CheckEntityErrors(&ent, t)
		}
	})
}
