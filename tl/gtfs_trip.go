package tl

import (
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/enum"
)

// Trip trips.txt
type Trip struct {
	RouteID              string               `csv:"route_id" required:"true"`
	ServiceID            string               `csv:"service_id" required:"true"`
	TripID               string               `csv:"trip_id" required:"true"`
	TripHeadsign         string               `csv:"trip_headsign"`
	TripShortName        string               `csv:"trip_short_name"`
	DirectionID          int                  `csv:"direction_id"`
	BlockID              string               `csv:"block_id"`
	ShapeID              OptionalRelationship `csv:"shape_id"`
	WheelchairAccessible int                  `csv:"wheelchair_accessible"`
	BikesAllowed         int                  `csv:"bikes_allowed"`
	StopPatternID        int
	JourneyPatternID     int
	JourneyPatternOffset int
	BaseEntity
}

// EntityID returns the ID or TripID.
func (ent *Trip) EntityID() string {
	return entID(ent.ID, ent.TripID)
}

// EntityKey returns the GTFS identifier.
func (ent *Trip) EntityKey() string {
	return ent.TripID
}

// Errors for this Entity.
func (ent *Trip) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, enum.CheckPresent("route_id", ent.RouteID)...)
	errs = append(errs, enum.CheckPresent("service_id", ent.ServiceID)...)
	errs = append(errs, enum.CheckPresent("trip_id", ent.TripID)...)
	errs = append(errs, enum.CheckInsideRangeInt("direction_id", ent.DirectionID, 0, 1)...)
	errs = append(errs, enum.CheckInsideRangeInt("wheelchair_accessible", ent.WheelchairAccessible, 0, 2)...)
	errs = append(errs, enum.CheckInsideRangeInt("bikes_allowed", ent.BikesAllowed, 0, 2)...)
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
	if serviceID, ok := emap.GetEntity(&Calendar{ServiceID: ent.ServiceID}); ok {
		ent.ServiceID = serviceID
	} else {
		return causes.NewInvalidReferenceError("service_id", ent.ServiceID)
	}
	// Adjust RouteID
	if routeID, ok := emap.GetEntity(&Route{RouteID: ent.RouteID}); ok {
		ent.RouteID = routeID
	} else {
		return causes.NewInvalidReferenceError("route_id", ent.RouteID)
	}
	// Adjust ShapeID
	if len(ent.ShapeID.Key) > 0 {
		if shapeID, ok := emap.GetEntity(&Shape{ShapeID: ent.ShapeID.Key}); ok {
			ent.ShapeID.Key = shapeID
			ent.ShapeID.Valid = true
		} else {
			return causes.NewInvalidReferenceError("shape_id", ent.ShapeID.Key)
		}
	}
	return nil
}
