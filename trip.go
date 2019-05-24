package gotransit

import (
	"github.com/interline-io/gotransit/causes"
)

// Trip trips.txt
type Trip struct {
	RouteID              string `csv:"route_id" required:"true" gorm:"type:int;index;not null"`
	ServiceID            string `csv:"service_id" required:"true" gorm:"type:int;index;not null"`
	TripID               string `csv:"trip_id" required:"true" gorm:"index:idx_trips_trip_id;index;not null"`
	TripHeadsign         string `csv:"trip_headsign"`
	TripShortName        string `csv:"trip_short_name"`
	DirectionID          int    `csv:"direction_id" min:"0" max:"1"`
	BlockID              string `csv:"block_id"`
	ShapeID              string `csv:"shape_id" gorm:"type:int;index"`
	WheelchairAccessible int    `csv:"wheelchair_accessible" min:"0" max:"2"`
	BikesAllowed         int    `csv:"bikes_allowed" min:"0" max:"2"`
	StopPatternID        int    `gorm:"index"`
	BaseEntity
}

// EntityID returns the ID or TripID.
func (ent *Trip) EntityID() string {
	return entID(ent.ID, ent.TripID)
}

// Warnings for this Entity.
func (ent *Trip) Warnings() (errs []error) {
	return errs
}

// Errors for this Entity.
func (ent *Trip) Errors() (errs []error) {
	errs = ValidateTags(ent)
	errs = append(errs, ent.BaseEntity.loadErrors...)
	return errs
}

// Filename trips.txt
func (ent *Trip) Filename() string {
	return "trips.txt"
}

// TableName gtfs_trips
func (ent *Trip) TableName() string {
	return "gtfs_trips"
}

// UpdateKeys updates Entity references.
func (ent *Trip) UpdateKeys(emap *EntityMap) error {
	if serviceID, ok := emap.Get(&Calendar{ServiceID: ent.ServiceID}); ok {
		ent.ServiceID = serviceID
	} else {
		return causes.NewInvalidReferenceError("service_id", ent.ServiceID)
	}
	// Adjust RouteID
	if routeID, ok := emap.Get(&Route{RouteID: ent.RouteID}); ok {
		ent.RouteID = routeID
	} else {
		return causes.NewInvalidReferenceError("route_id", ent.RouteID)
	}
	// Adjust ShapeID
	if len(ent.ShapeID) > 0 {
		if shapeID, ok := emap.Get(&Shape{ShapeID: ent.ShapeID}); ok {
			ent.ShapeID = shapeID
		} else {
			return causes.NewInvalidReferenceError("shape_id", ent.ShapeID)
		}
	}
	return nil
}
