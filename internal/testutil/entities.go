package testutil

import (
	"github.com/interline-io/transitland-lib/tl"
)

// AllEntities iterates through all Reader entities, calling the specified callback.
func AllEntities(reader tl.Reader, cb func(tl.Entity)) {
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
	for ent := range reader.Attributions() {
		cb(&ent)
	}
	for ent := range reader.Translations() {
		cb(&ent)
	}
	for ent := range reader.Areas() {
		cb(&ent)
	}
	for ent := range reader.StopAreas() {
		cb(&ent)
	}
	for ent := range reader.FareContainers() {
		cb(&ent)
	}
	for ent := range reader.RiderCategories() {
		cb(&ent)
	}
	for ent := range reader.FareProducts() {
		cb(&ent)
	}
	for ent := range reader.FareLegRules() {
		cb(&ent)
	}
	for ent := range reader.FareTransferRules() {
		cb(&ent)
	}
}
