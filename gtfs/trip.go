package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
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
	// GTFS-Flex: safe duration fields (google/transit#598)
	// See: https://github.com/google/transit/pull/598
	SafeDurationFactor tt.Float
	SafeDurationOffset tt.Float
	JourneyPatternID   tt.String `csv:"-"`
	JourneyPatternOffset tt.Int     `csv:"-"`
	StopPatternID        tt.Int     `csv:"-"`
	StopTimes            []StopTime `csv:"-" db:"-"` // for validation
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

// ConditionalErrors validates GTFS-Flex safe duration fields.
func (ent *Trip) ConditionalErrors() (errs []error) {
	// safe_duration_factor and safe_duration_offset must both be present or both absent
	if ent.SafeDurationFactor.Valid && !ent.SafeDurationOffset.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("safe_duration_offset"))
	}
	if ent.SafeDurationOffset.Valid && !ent.SafeDurationFactor.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("safe_duration_factor"))
	}
	// safe_duration_factor must be positive
	if ent.SafeDurationFactor.Valid && ent.SafeDurationFactor.Val <= 0 {
		errs = append(errs, causes.NewInvalidFieldError(
			"safe_duration_factor",
			fmt.Sprintf("%f", ent.SafeDurationFactor.Val),
			fmt.Errorf("must be positive"),
		))
	}
	return errs
}
