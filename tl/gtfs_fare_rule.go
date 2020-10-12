package tl

import (
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/enum"
)

// FareRule fare_rules.txt
type FareRule struct {
	FareID        string               `csv:"fare_id" required:"true"`
	RouteID       OptionalRelationship `csv:"route_id" `
	OriginID      string               `csv:"origin_id"`
	DestinationID string               `csv:"destination_id"`
	ContainsID    string               `csv:"contains_id"`
	BaseEntity
}

// EntityID returns nothing.
func (ent *FareRule) EntityID() string {
	return ""
}

// Errors for this Entity.
func (ent *FareRule) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, enum.CheckPresent("fare_id", ent.FareID)...)
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
	if v := ent.RouteID.Key; v != "" {
		if routeID, ok := emap.GetEntity(&Route{RouteID: v}); ok {
			ent.RouteID.Key = routeID
			ent.RouteID.Valid = true
		} else {
			return causes.NewInvalidReferenceError("route_id", v)
		}
	}
	return nil
}
