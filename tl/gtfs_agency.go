package tl

import (
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/enum"
)

// Agency agency.txt
type Agency struct {
	AgencyID       string `csv:"agency_id"`
	AgencyName     string `csv:"agency_name" required:"true"`
	AgencyURL      string `csv:"agency_url" required:"true"`
	AgencyTimezone string `csv:"agency_timezone" required:"true"`
	AgencyLang     string `csv:"agency_lang"`
	AgencyPhone    string `csv:"agency_phone"`
	AgencyFareURL  string `csv:"agency_fare_url"`
	AgencyEmail    string `csv:"agency_email"`
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

// Warnings for this Entity.
func (ent *Agency) Warnings() (errs []error) {
	errs = append(errs, ent.loadWarnings...)
	if len(ent.AgencyID) == 0 {
		errs = append(errs, causes.NewValidationWarning("agency_id", "agency_id should be set"))
	}
	if !enum.IsValidTimezone(ent.AgencyTimezone) {
		errs = append(errs, causes.NewValidationWarning("agency_timezone", "agency_timezone is not a valid timezone"))
	}
	return errs
}

// Errors for this Entity.
func (ent *Agency) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, enum.CheckPresent("agency_name", ent.AgencyName)...)
	errs = append(errs, enum.CheckPresent("agency_url", ent.AgencyURL)...)
	errs = append(errs, enum.CheckPresent("agency_timezone", ent.AgencyTimezone)...)
	errs = append(errs, enum.CheckURL("agency_url", ent.AgencyURL)...)
	errs = append(errs, enum.CheckURL("agency_fare_url", ent.AgencyFareURL)...)
	errs = append(errs, enum.CheckLanguage("agency_lang", ent.AgencyLang)...)
	errs = append(errs, enum.CheckEmail("agency_email", ent.AgencyEmail)...)
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
