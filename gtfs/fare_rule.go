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

// // Errors for this Entity.
func (ent *FareRule) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.LoadErrors()...)
	errs = append(errs, tt.CheckPresent("fare_id", ent.FareID.Val)...)
	return errs
}

// Filename fare_rules.txt
func (ent *FareRule) Filename() string {
	return "fare_rules.txt"
}

// TableName gtfs_fare_Rules
func (ent *FareRule) TableName() string {
	return "gtfs_fare_rules"
}
