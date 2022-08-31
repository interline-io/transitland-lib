package plus

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

type RouteAttributes struct {
	RouteID     tt.Key
	Category    tt.Int
	Subcategory tt.Int
	RunningWay  tt.Int
	tl.BaseEntity
}

func (ent *RouteAttributes) Filename() string {
	return "route_attributes.txt"
}

func (ent *RouteAttributes) TableName() string {
	return "ext_plus_route_attributes"
}

// UpdateKeys updates Entity references.
func (ent *RouteAttributes) UpdateKeys(emap *tl.EntityMap) error {
	if routeID, ok := emap.GetEntity(&tl.Route{RouteID: ent.RouteID.Val}); ok {
		ent.RouteID = tt.NewKey(routeID)
	} else {
		return causes.NewInvalidReferenceError("route_id", ent.RouteID.Val)
	}
	return nil
}
