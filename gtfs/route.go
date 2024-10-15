package gtfs

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// Route routes.txt
type Route struct {
	RouteID           tt.String `csv:",required"`
	AgencyID          tt.Key    `csv:",required" target:"agency.txt"`
	RouteShortName    tt.String
	RouteLongName     tt.String
	RouteDesc         tt.String
	RouteType         tt.Int `csv:",required"`
	RouteURL          tt.Url
	RouteColor        tt.Color
	RouteTextColor    tt.Color
	RouteSortOrder    tt.Int
	ContinuousPickup  tt.Int
	ContinuousDropOff tt.Int
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

// Errors for this Entity.
func (ent *Route) Errors() (errs []error) {
	errs = append(errs, tt.CheckPresent("route_id", ent.RouteID.Val)...)
	errs = append(errs, tt.CheckURL("route_url", ent.RouteURL.Val)...)
	errs = append(errs, tt.CheckColor("route_color", ent.RouteColor.Val)...)
	errs = append(errs, tt.CheckColor("route_text_color", ent.RouteTextColor.Val)...)
	errs = append(errs, tt.CheckPositiveInt("route_sort_order", ent.RouteSortOrder.Val)...)
	errs = append(errs, tt.CheckInArrayInt("continuous_pickup", ent.ContinuousPickup.Val, 0, 1, 2, 3)...)
	errs = append(errs, tt.CheckInArrayInt("continuous_drop_off", ent.ContinuousDropOff.Val, 0, 1, 2, 3)...)
	return errs
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

// UpdateKeys updates Entity references.
func (ent *Route) UpdateKeys(emap *EntityMap) error {
	aid := ent.AgencyID.Val
	if agencyID, ok := emap.GetEntity(&Agency{AgencyID: tt.NewString(aid)}); ok {
		ent.AgencyID.Set(agencyID)
	} else if aid == "" {
		// best practice warning, handled elsewhere
	} else {
		return causes.NewInvalidReferenceError("agency_id", aid)
	}
	return nil
}
