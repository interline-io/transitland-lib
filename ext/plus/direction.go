package plus

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// Direction directions.txt
type Direction struct {
	RouteID     string `csv:"route_id"`
	DirectionID string `csv:"direction_id"`
	Direction   string `csv:"direction"`
	tt.BaseEntity
}

// Filename directions.txt
func (ent *Direction) Filename() string {
	return "directions.txt"
}

// TableName ext_fare_attributes
func (ent *Direction) TableName() string {
	return "ext_plus_directions"
}

// UpdateKeys updates Entity references.
func (ent *Direction) UpdateKeys(emap *tt.EntityMap) error {
	if routeID, ok := emap.GetEntity(&gtfs.Route{RouteID: tt.NewString(ent.RouteID)}); ok {
		ent.RouteID = routeID
	} else {
		return causes.NewInvalidReferenceError("route_id", ent.RouteID)
	}
	return nil
}
