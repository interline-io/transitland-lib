package testutil

import (
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tt"
)

// AllEntities iterates through all Reader entities, calling the specified callback.
func AllEntities(reader adapters.Reader, cb func(tt.Entity)) {
	for ent := range reader.Agencies() {
		cb(&ent)
	}
	for ent := range reader.Routes() {
		cb(&ent)
	}

	// stops
	for ent := range reader.Stops() {
		if ent.LocationType.Val == 1 {
			cb(&ent)
		}
	}
	for ent := range reader.Stops() {
		if ent.LocationType.Val == 0 || ent.LocationType.Val == 2 || ent.LocationType.Val == 3 {
			cb(&ent)
		}
	}
	for ent := range reader.Stops() {
		if ent.LocationType.Val == 4 {
			cb(&ent)
		}
	}

	// shapes
	for ent := range reader.Shapes() {
		cb(&ent)
	}

	// services
	svcs := service.NewServicesFromReader(reader)
	for _, svc := range svcs {
		cb(svc)
	}
	for cd := range reader.CalendarDates() {
		cb(&cd)
	}

	// trips and stop times
	for ent := range reader.Trips() {
		cb(&ent)
	}
	for ent := range reader.StopTimes() {
		cb(&ent)
	}

	// other entities
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
	for ent := range reader.FareMedia() {
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
