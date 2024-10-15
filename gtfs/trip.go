package gtfs

import (
	"github.com/interline-io/transitland-lib/tt"
)

// Trip trips.txt
type Trip struct {
	RouteID              tt.Key    `csv:",required" target:"routes.txt"`
	ServiceID            tt.Key    `csv:",required" target:"calendar.txt"`
	TripID               tt.String `csv:",required"`
	TripHeadsign         tt.String
	TripShortName        tt.String
	DirectionID          tt.Int `enum:"0,1"`
	BlockID              tt.String
	ShapeID              tt.Key     `target:"shapes.txt"`
	WheelchairAccessible tt.Int     `enum:"0,1,2"`
	BikesAllowed         tt.Int     `enum:"0,1,2"`
	StopTimes            []StopTime `csv:"-" db:"-"` // for validation methods
	JourneyPatternID     tt.String  `csv:"-"`
	JourneyPatternOffset tt.Int     `csv:"-"`
	StopPatternID        tt.Int     `csv:"-"`
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
