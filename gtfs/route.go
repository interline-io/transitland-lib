package gtfs

import (
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

// Route routes.txt
type Route struct {
	RouteID           string `csv:",required"`
	AgencyID          string
	RouteShortName    string
	RouteLongName     string
	RouteDesc         string
	RouteType         int `csv:",required"`
	RouteURL          string
	RouteColor        string
	RouteTextColor    string
	RouteSortOrder    int
	ContinuousPickup  tt.Int
	ContinuousDropOff tt.Int
	NetworkID         tt.String
	AsRoute           tt.Int
	Geometry          tt.Geometry `csv:"-" db:"-"`
	tt.BaseEntity
}

// EntityID returns ID or RouteID.
func (ent *Route) EntityID() string {
	return entID(ent.ID, ent.RouteID)
}

// EntityKey returns the GTFS identifier.
func (ent *Route) EntityKey() string {
	return ent.RouteID
}

// Errors for this Entity.
func (ent *Route) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, tt.CheckPresent("route_id", ent.RouteID)...)
	errs = append(errs, tt.CheckURL("route_url", ent.RouteURL)...)
	errs = append(errs, tt.CheckColor("route_color", ent.RouteColor)...)
	errs = append(errs, tt.CheckColor("route_text_color", ent.RouteTextColor)...)
	errs = append(errs, tt.CheckPositiveInt("route_sort_order", ent.RouteSortOrder)...)
	errs = append(errs, tt.CheckInArrayInt("continuous_pickup", ent.ContinuousPickup.Val, 0, 1, 2, 3)...)
	errs = append(errs, tt.CheckInArrayInt("continuous_drop_off", ent.ContinuousDropOff.Val, 0, 1, 2, 3)...)
	if len(ent.RouteShortName) == 0 && len(ent.RouteLongName) == 0 {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("route_short_name"))
	}
	if _, ok := tt.GetRouteType(ent.RouteType); !ok {
		errs = append(errs, causes.NewInvalidFieldError("route_type", tt.TryCsv(ent.RouteType), nil))
	}
	return errs
}

// Filename routes.txt
func (ent *Route) Filename() string {
	return "routes.txt"
}

// TableName gtfs_routes
func (ent *Route) TableName() string {
	return "gtfs_routes"
}

// UpdateKeys updates Entity references.
func (ent *Route) UpdateKeys(emap *EntityMap) error {
	aid := ent.AgencyID
	if agencyID, ok := emap.GetEntity(&Agency{AgencyID: ent.AgencyID}); ok {
		ent.AgencyID = agencyID
	} else if aid == "" {
		// best practice warning, handled elsewhere
	} else {
		return causes.NewInvalidReferenceError("agency_id", ent.AgencyID)
	}
	return nil
}
