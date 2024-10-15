package gtfs

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// Route routes.txt
type Route struct {
	RouteID           tt.String `csv:",required"`
	AgencyID          tt.Key    `target:"agency.txt"`
	RouteShortName    tt.String
	RouteLongName     tt.String
	RouteDesc         tt.String
	RouteType         tt.Int `csv:",required"`
	RouteURL          tt.Url
	RouteColor        tt.Color
	RouteTextColor    tt.Color
	RouteSortOrder    tt.Int `range:"0,"`
	ContinuousPickup  tt.Int `enum:"0,1,2,3"`
	ContinuousDropOff tt.Int `enum:"0,1,2,3"`
	NetworkID         tt.String
	AsRoute           tt.Int
	Geometry          tt.Geometry `csv:"-" db:"-"`
	tt.BaseEntity
}

// EntityID returns ID or RouteID.
func (ent *Route) EntityID() string {
	return entID(ent.ID, ent.RouteID.Val)
}

// EntityKey returns the GTFS identifier.
func (ent *Route) EntityKey() string {
	return ent.RouteID.Val
}

// Filename routes.txt
func (ent *Route) Filename() string {
	return "routes.txt"
}

// TableName gtfs_routes
func (ent *Route) TableName() string {
	return "gtfs_routes"
}

func (ent *Route) ConditionalErrors() []error {
	var errs []error
	if !ent.RouteShortName.Valid && !ent.RouteLongName.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("route_short_name"))
	}
	if _, ok := tt.GetRouteType(ent.RouteType.Int()); !ok {
		errs = append(errs, causes.NewInvalidFieldError("route_type", ent.RouteType.String(), nil))
	}
	return errs
}
