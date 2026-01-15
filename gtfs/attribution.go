package gtfs

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

type Attribution struct {
	OrganizationName tt.String `csv:",required" standardized_sort:"2"`
	AgencyID         tt.Key    `target:"agency.txt"`
	RouteID          tt.Key    `target:"routes.txt"`
	TripID           tt.Key    `target:"trips.txt"`
	IsProducer       tt.Int    `enum:"0,1"`
	IsOperator       tt.Int    `enum:"0,1"`
	IsAuthority      tt.Int    `enum:"0,1"`
	AttributionID    tt.String `standardized_sort:"1"`
	AttributionURL   tt.Url
	AttributionEmail tt.Email
	AttributionPhone tt.String
	tt.BaseEntity
}

func (ent *Attribution) Filename() string {
	return "attributions.txt"
}

func (ent *Attribution) TableName() string {
	return "gtfs_attributions"
}

// Errors for this Entity.
func (ent *Attribution) ConditionalErrors() (errs []error) {
	// At least one must be present
	if ent.IsProducer.Val == 0 && ent.IsOperator.Val == 0 && ent.IsAuthority.Val == 0 {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("is_producer"))
	}
	// Mutually exclusive fields
	if ent.AgencyID.Val != "" {
		if ent.RouteID.Val != "" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("route_id", ent.RouteID.Val, "route_id cannot be set if agency_id is present"))
		}
		if ent.TripID.Val != "" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("trip_id", ent.TripID.Val, "trip_id cannot be set if agency_id is present"))
		}
	} else if ent.RouteID.Val != "" {
		if ent.TripID.Val != "" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("trip_id", ent.RouteID.Val, "trip_id cannot be set if route_id is present"))
		}
	}
	return errs
}
