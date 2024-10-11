package tl

import "github.com/interline-io/transitland-lib/tl/tt"

// Agency agency.txt
type Agency struct {
	AgencyID       string
	AgencyName     string `csv:",required"`
	AgencyURL      string `csv:",required"`
	AgencyTimezone string `csv:",required"`
	AgencyLang     string
	AgencyPhone    string
	AgencyFareURL  string
	AgencyEmail    string
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
	errs = append(errs, tt.CheckPresent("agency_name", ent.AgencyName)...)
	errs = append(errs, tt.CheckPresent("agency_url", ent.AgencyURL)...)
	errs = append(errs, tt.CheckPresent("agency_timezone", ent.AgencyTimezone)...)
	errs = append(errs, tt.CheckTimezone("agency_timezone", ent.AgencyTimezone)...)
	errs = append(errs, tt.CheckURL("agency_url", ent.AgencyURL)...)
	errs = append(errs, tt.CheckURL("agency_fare_url", ent.AgencyFareURL)...)
	errs = append(errs, tt.CheckLanguage("agency_lang", ent.AgencyLang)...)
	errs = append(errs, tt.CheckEmail("agency_email", ent.AgencyEmail)...)
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
