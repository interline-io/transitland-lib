package tl

import (
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

// Trip trips.txt
type Trip struct {
	RouteID              string `csv:",required"`
	ServiceID            string `csv:",required"`
	TripID               string `csv:",required"`
	TripHeadsign         string
	TripShortName        string
	DirectionID          int
	BlockID              string
	ShapeID              tt.Key
	WheelchairAccessible int
	BikesAllowed         int
	StopTimes            []StopTime `csv:"-" db:"-"` // for validation methods
	StopPatternID        int        `csv:"-"`
	JourneyPatternID     string     `csv:"-"`
	JourneyPatternOffset int        `csv:"-"`
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
	errs = append(errs, tt.CheckPresent("route_id", ent.RouteID)...)
	errs = append(errs, tt.CheckPresent("service_id", ent.ServiceID)...)
	errs = append(errs, tt.CheckPresent("trip_id", ent.TripID)...)
	errs = append(errs, tt.CheckInsideRangeInt("direction_id", ent.DirectionID, 0, 1)...)
	errs = append(errs, tt.CheckInsideRangeInt("wheelchair_accessible", ent.WheelchairAccessible, 0, 2)...)
	errs = append(errs, tt.CheckInsideRangeInt("bikes_allowed", ent.BikesAllowed, 0, 2)...)
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
	if len(ent.ShapeID.Val) > 0 {
		if shapeID, ok := emap.GetEntity(&Shape{ShapeID: ent.ShapeID.Val}); ok {
			ent.ShapeID.Val = shapeID
			ent.ShapeID.Valid = true
		} else {
			return causes.NewInvalidReferenceError("shape_id", ent.ShapeID.Val)
		}
	}
	return nil
}
