package gtfs

import (
	"github.com/interline-io/transitland-lib/tt"
)

// FareRule fare_rules.txt
type FareRule struct {
	FareID        tt.String `csv:",required" target:"fare_attributes.txt"`
	RouteID       tt.Key    `target:"routes.txt"`
	OriginID      tt.String
	DestinationID tt.String
	ContainsID    tt.String
	tt.BaseEntity
}

// Filename fare_rules.txt
func (ent *FareRule) Filename() string {
	return "fare_rules.txt"
}

// TableName gtfs_fare_Rules
func (ent *FareRule) TableName() string {
	return "gtfs_fare_rules"
}
