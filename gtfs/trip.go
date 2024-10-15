package gtfs

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// Trip trips.txt
type Trip struct {
	RouteID              tt.Key    `csv:",required"`
	ServiceID            tt.Key    `csv:",required"`
	TripID               tt.String `csv:",required"`
	TripHeadsign         tt.String
	TripShortName        tt.String
	DirectionID          tt.Int
	BlockID              tt.String
	ShapeID              tt.Key
	WheelchairAccessible tt.Int
	BikesAllowed         tt.Int
	StopTimes            []StopTime `csv:"-" db:"-"` // for validation methods
	StopPatternID        tt.Int     `csv:"-"`
	JourneyPatternID     tt.String  `csv:"-"`
	JourneyPatternOffset tt.Int     `csv:"-"`
	tt.BaseEntity
}

// EntityID returns the ID or TripID.
func (ent *Trip) EntityID() string {
	return entID(ent.ID, ent.TripID.Val)
}

// EntityKey returns the GTFS identifier.
func (ent *Trip) EntityKey() string {
	return ent.TripID.Val
}

// Filename trips.txt
func (ent *Trip) Filename() string {
	return "trips.txt"
}

// TableName gtfs_trips
func (ent *Trip) TableName() string {
	return "gtfs_trips"
}

// Errors for this Entity.
func (ent *Trip) Errors() (errs []error) {
	errs = append(errs, tt.CheckPresent("route_id", ent.RouteID.Val)...)
	errs = append(errs, tt.CheckPresent("service_id", ent.ServiceID.Val)...)
	errs = append(errs, tt.CheckPresent("trip_id", ent.TripID.Val)...)
	errs = append(errs, tt.CheckInsideRangeInt("direction_id", ent.DirectionID.Val, 0, 1)...)
	errs = append(errs, tt.CheckInsideRangeInt("wheelchair_accessible", ent.WheelchairAccessible.Val, 0, 2)...)
	errs = append(errs, tt.CheckInsideRangeInt("bikes_allowed", ent.BikesAllowed.Val, 0, 2)...)
	return errs
}

// UpdateKeys updates Entity references.
func (ent *Trip) UpdateKeys(emap *EntityMap) error {
	if serviceID, ok := emap.GetEntity(&Calendar{ServiceID: ent.ServiceID.Val}); ok {
		ent.ServiceID.Set(serviceID)
	} else {
		return causes.NewInvalidReferenceError("service_id", ent.ServiceID.Val)
	}
	// Adjust RouteID
	if routeID, ok := emap.GetEntity(&Route{RouteID: ent.RouteID.Val}); ok {
		ent.RouteID.Set(routeID)
	} else {
		return causes.NewInvalidReferenceError("route_id", ent.RouteID.Val)
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
