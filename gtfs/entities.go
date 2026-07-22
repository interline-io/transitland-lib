package gtfs

import "github.com/interline-io/transitland-lib/tt"

// AllEntities returns a fresh zero-value instance of every GTFS entity type
// defined in this package, in roughly reference.md order. It is the canonical
// enumeration of GTFS entities, intended as a single source of truth for
// schema-drift checks and any future field-level code generation.
//
// Note: fare_leg_join_rules.txt has no model yet and is intentionally absent.
func AllEntities() []tt.Entity {
	return []tt.Entity{
		&Agency{},
		&Stop{},
		&Route{},
		&Trip{},
		&StopTime{},
		&Calendar{},
		&CalendarDate{},
		&FareAttribute{},
		&FareRule{},
		&Timeframe{},
		&RiderCategory{},
		&FareMedia{},
		&FareProduct{},
		&FareLegRule{},
		&FareLegJoinRule{},
		&FareTransferRule{},
		&Area{},
		&StopArea{},
		&Network{},
		&RouteNetwork{},
		&Shape{},
		&Frequency{},
		&Transfer{},
		&Pathway{},
		&Level{},
		&LocationGroup{},
		&LocationGroupStop{},
		&Location{},
		&BookingRule{},
		&Translation{},
		&FeedInfo{},
		&Attribution{},
	}
}
