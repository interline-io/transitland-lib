package testutil

import (
	"github.com/interline-io/gotransit"
)

// AllEntities iterates through all Reader entities, calling the specified callback.
func AllEntities(reader gotransit.Reader, cb func(gotransit.Entity)) {
	for ent := range reader.Agencies() {
		cb(&ent)
	}
	for ent := range reader.Routes() {
		cb(&ent)
	}
	for ent := range reader.Stops() {
		if ent.LocationType == 1 {
			cb(&ent)
		}
	}
	for ent := range reader.Stops() {
		if ent.LocationType != 1 {
			cb(&ent)
		}
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
	for ent := range reader.Trips() {
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
