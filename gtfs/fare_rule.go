package gtfs

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// FareRule fare_rules.txt
type FareRule struct {
	FareID        string `csv:",required"`
	RouteID       tt.Key
	OriginID      string
	DestinationID string
	ContainsID    string
	tt.BaseEntity
}

// Errors for this Entity.
func (ent *FareRule) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, tt.CheckPresent("fare_id", ent.FareID)...)
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

// UpdateKeys updates Entity references.
func (ent *FareRule) UpdateKeys(emap *EntityMap) error {
	if fareID, ok := emap.GetEntity(&FareAttribute{FareID: ent.FareID}); ok {
		ent.FareID = fareID
	} else {
		return causes.NewInvalidReferenceError("fare_id", ent.FareID)
	}
	if v := ent.RouteID.Val; v != "" {
		if routeID, ok := emap.GetEntity(&Route{RouteID: v}); ok {
			ent.RouteID.Val = routeID
			ent.RouteID.Valid = true
		} else {
			return causes.NewInvalidReferenceError("route_id", v)
		}
	}
	return nil
}
