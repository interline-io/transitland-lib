package gotransit

import (
	"github.com/interline-io/gotransit/causes"
)

// FareRule fare_rules.txt
type FareRule struct {
	FareID        string `csv:"fare_id" required:"true" gorm:"type:int;index;not null"`
	RouteID       string `csv:"route_id" gorm:"type:int"`
	OriginID      string `csv:"origin_id"`
	DestinationID string `csv:"destination_id"`
	ContainsID    string `csv:"contains_id"`
	BaseEntity
}

// EntityID returns nothing.
func (ent *FareRule) EntityID() string {
	return ""
}

// Warnings for this Entity.
func (ent *FareRule) Warnings() (errs []error) {
	return errs
}

// Errors for this Entity.
func (ent *FareRule) Errors() (errs []error) {
	errs = ValidateTags(ent)
	errs = append(errs, ent.BaseEntity.loadErrors...)
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
	if fareID, ok := emap.Get(&FareAttribute{FareID: ent.FareID}); ok {
		ent.FareID = fareID
	} else {
		return causes.NewInvalidReferenceError("fare_id", ent.FareID)
	}
	if len(ent.RouteID) > 0 {
		if routeID, ok := emap.Get(&Route{RouteID: ent.RouteID}); ok {
			ent.RouteID = routeID
		} else {
			return causes.NewInvalidReferenceError("route_id", ent.RouteID)
		}
	}
	return nil
}
