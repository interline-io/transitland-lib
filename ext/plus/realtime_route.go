package plus

import (
	tl "github.com/interline-io/transitland-lib"
	"github.com/interline-io/transitland-lib/causes"
)

// RealtimeRoute realtime_routes.txt
type RealtimeRoute struct {
	RouteID         string `csv:"route_id"`
	RealtimeEnabled int    `csv:"realtime_enabled"`
	tl.BaseEntity
}

// Filename realtime_routes.txt
func (ent *RealtimeRoute) Filename() string {
	return "realtime_routes.txt"
}

// TableName ext_plus_realtime_routes
func (ent *RealtimeRoute) TableName() string {
	return "ext_plus_realtime_routes"
}

// UpdateKeys updates Entity references.
func (ent *RealtimeRoute) UpdateKeys(emap *tl.EntityMap) error {
	if fkid, ok := emap.GetEntity(&tl.Route{RouteID: ent.RouteID}); ok {
		ent.RouteID = fkid
	} else {
		return causes.NewInvalidReferenceError("route_id", ent.RouteID)
	}
	return nil
}
