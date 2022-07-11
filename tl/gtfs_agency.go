package tl

import (
	"github.com/interline-io/transitland-lib/tl/enum"
)

// Agency agency.txt
type Agency struct {
	AgencyID       string
	AgencyName     string
	AgencyURL      string
	AgencyTimezone Timezone
	AgencyLang     Language
	AgencyPhone    String
	AgencyFareURL  String
	AgencyEmail    String
	BaseEntity
}

// EntityID returns the ID or AgencyID.
func (ent *Agency) EntityID() string {
	return entID(ent.ID, ent.AgencyID)
}

// EntityKey returns the GTFS identifier.
func (ent *Agency) EntityKey() string {
	return ent.AgencyID
}

// Errors for this Entity.
func (ent *Agency) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = enum.CheckError(errs, enum.CheckFieldPresentError("agency_timezone", &ent.AgencyTimezone))
	errs = enum.CheckError(errs, enum.CheckFieldError("agency_lang", &ent.AgencyLang))

	errs = append(errs, enum.CheckPresent("agency_name", ent.AgencyName)...)
	errs = append(errs, enum.CheckPresent("agency_url", ent.AgencyURL)...)
	errs = append(errs, enum.CheckURL("agency_url", ent.AgencyURL)...)
	errs = append(errs, enum.CheckURL("agency_fare_url", ent.AgencyFareURL.Val)...)
	errs = append(errs, enum.CheckEmail("agency_email", ent.AgencyEmail.Val)...)
	return errs
}

// Filename agency.txt
func (ent *Agency) Filename() string {
	return "agency.txt"
}

// TableName gtfs_agencies
func (ent *Agency) TableName() string {
	return "gtfs_agencies"
}
