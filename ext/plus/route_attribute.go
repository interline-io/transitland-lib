package plus

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

type RouteAttribute struct {
	RouteID     tt.Key
	Category    tt.Int
	Subcategory tt.Int
	RunningWay  tt.Int
	tt.BaseEntity
}

func (ent *RouteAttribute) Filename() string {
	return "route_attributes.txt"
}

func (ent *RouteAttribute) TableName() string {
	return "ext_plus_route_attributes"
}

func (ent *RouteAttribute) UpdateKeys(emap *tt.EntityMap) error {
	if routeID, ok := emap.GetEntity(&gtfs.Route{RouteID: tt.NewString(ent.RouteID.Val)}); ok {
		ent.RouteID = tt.NewKey(routeID)
	} else {
		return causes.NewInvalidReferenceError("route_id", ent.RouteID.Val)
	}
	return nil
}
