package gotransit

import (
	"github.com/interline-io/gotransit/causes"
	"github.com/interline-io/gotransit/enums"
)

// Agency agency.txt
type Agency struct {
	AgencyID       string `csv:"agency_id"`
	AgencyName     string `csv:"agency_name" required:"true"`
	AgencyURL      string `csv:"agency_url" required:"true" validator:"url"`
	AgencyTimezone string `csv:"agency_timezone" required:"true"`
	AgencyLang     string `csv:"agency_lang" validator:"lang"`
	AgencyPhone    string `csv:"agency_phone"`
	AgencyFareURL  string `csv:"agency_fare_url" validator:"url"`
	AgencyEmail    string `csv:"agency_email" validator:"email"`
	BaseEntity
}

// EntityID returns the ID or AgencyID.
func (ent *Agency) EntityID() string {
	return entID(ent.ID, ent.AgencyID)
}

// Warnings for this Entity.
func (ent *Agency) Warnings() (errs []error) {
	errs = append(errs, ent.loadWarnings...)
	if len(ent.AgencyID) == 0 {
		errs = append(errs, causes.NewValidationWarning("agency_id", "agency_id should be set"))
	}
	if !enums.IsValidTimezone(ent.AgencyTimezone) {
		errs = append(errs, causes.NewValidationWarning("agency_timezone", "agency_timezone is not a valid timezone"))
	}
	return errs
}

// Errors for this Entity.
func (ent *Agency) Errors() (errs []error) {
	errs = ValidateTags(ent)
	errs = append(errs, ent.BaseEntity.loadErrors...)
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
