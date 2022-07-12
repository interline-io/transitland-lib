package tl

import (
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

type Attribution struct {
	OrganizationName String
	AgencyID         Key
	RouteID          Key
	TripID           Key
	IsProducer       Int
	IsOperator       Int
	IsAuthority      Int
	AttributionID    String
	AttributionURL   String
	AttributionEmail String
	AttributionPhone String
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
	errs = append(errs, tt.CheckPresent("organization_name", ent.OrganizationName.String)...)
	errs = append(errs, tt.CheckURL("attribution_url", ent.AttributionURL.String)...)
	errs = append(errs, tt.CheckInsideRangeInt("is_producer", ent.IsProducer.Int, 0, 1)...)
	errs = append(errs, tt.CheckInsideRangeInt("is_operator", ent.IsOperator.Int, 0, 1)...)
	errs = append(errs, tt.CheckInsideRangeInt("is_authority", ent.IsAuthority.Int, 0, 1)...)
	errs = append(errs, tt.CheckEmail("attribution_email", ent.AttributionEmail.String)...)
	// At least one must be present
	if ent.IsProducer.Int == 0 && ent.IsOperator.Int == 0 && ent.IsAuthority.Int == 0 {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("is_producer"))
	}
	// Mutually exclusive fields
	if ent.AgencyID.Key != "" {
		if ent.RouteID.Key != "" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("route_id", "route_id cannot be set if agency_id is present"))
		}
		if ent.TripID.Key != "" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("trip_id", "trip_id cannot be set if agency_id is present"))
		}
	} else if ent.RouteID.Key != "" {
		if ent.TripID.Key != "" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("trip_id", "trip_id cannot be set if route_id is present"))
		}
	}
	return errs
}

// UpdateKeys updates Entity references.
func (ent *Attribution) UpdateKeys(emap *EntityMap) error {
	// Adjust AgencyID
	if ent.AgencyID.Key != "" {
		if eid, ok := emap.GetEntity(&Agency{AgencyID: ent.AgencyID.Key}); ok {
			ent.AgencyID = NewKey(eid)
		} else {
			return causes.NewInvalidReferenceError("agency_id", ent.AgencyID.Key)
		}
	}
	// Adjust RouteID
	if ent.RouteID.Key != "" {
		if eid, ok := emap.GetEntity(&Route{RouteID: ent.RouteID.Key}); ok {
			ent.RouteID = NewKey(eid)
		} else {
			return causes.NewInvalidReferenceError("route_id", ent.RouteID.Key)
		}
	}
	// Adjust TripID
	if ent.TripID.Key != "" {
		if eid, ok := emap.GetEntity(&Trip{TripID: ent.TripID.Key}); ok {
			ent.TripID = NewKey(eid)
		} else {
			return causes.NewInvalidReferenceError("trip_id", ent.TripID.Key)
		}
	}
	return nil
}
