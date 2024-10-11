package tl

import (
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

type Attribution struct {
	OrganizationName tt.String
	AgencyID         tt.Key
	RouteID          tt.Key
	TripID           tt.Key
	IsProducer       tt.Int
	IsOperator       tt.Int
	IsAuthority      tt.Int
	AttributionID    tt.String
	AttributionURL   tt.String
	AttributionEmail tt.String
	AttributionPhone tt.String
	BaseEntity
}

func (ent *Attribution) Filename() string {
	return "attributions.txt"
}

func (ent *Attribution) TableName() string {
	return "gtfs_attributions"
}

// Errors for this Entity.
func (ent *Attribution) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, tt.CheckPresent("organization_name", ent.OrganizationName.Val)...)
	errs = append(errs, tt.CheckURL("attribution_url", ent.AttributionURL.Val)...)
	errs = append(errs, tt.CheckInsideRangeInt("is_producer", ent.IsProducer.Val, 0, 1)...)
	errs = append(errs, tt.CheckInsideRangeInt("is_operator", ent.IsOperator.Val, 0, 1)...)
	errs = append(errs, tt.CheckInsideRangeInt("is_authority", ent.IsAuthority.Val, 0, 1)...)
	errs = append(errs, tt.CheckEmail("attribution_email", ent.AttributionEmail.Val)...)
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

// UpdateKeys updates Entity references.
func (ent *Attribution) UpdateKeys(emap *EntityMap) error {
	// Adjust AgencyID
	if ent.AgencyID.Val != "" {
		if eid, ok := emap.GetEntity(&Agency{AgencyID: ent.AgencyID.Val}); ok {
			ent.AgencyID = tt.NewKey(eid)
		} else {
			return causes.NewInvalidReferenceError("agency_id", ent.AgencyID.Val)
		}
	}
	// Adjust RouteID
	if ent.RouteID.Val != "" {
		if eid, ok := emap.GetEntity(&Route{RouteID: ent.RouteID.Val}); ok {
			ent.RouteID = tt.NewKey(eid)
		} else {
			return causes.NewInvalidReferenceError("route_id", ent.RouteID.Val)
		}
	}
	// Adjust TripID
	if ent.TripID.Val != "" {
		if eid, ok := emap.GetEntity(&Trip{TripID: ent.TripID.Val}); ok {
			ent.TripID = tt.NewKey(eid)
		} else {
			return causes.NewInvalidReferenceError("trip_id", ent.TripID.Val)
		}
	}
	return nil
}
