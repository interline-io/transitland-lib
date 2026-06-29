package gtfs

import "github.com/interline-io/transitland-lib/tt"

// Agency agency.txt
type Agency struct {
	AgencyID       tt.String   `standardized_sort:"1"`
	AgencyName     tt.String   `csv:",required"`
	AgencyURL      tt.Url      // Required by the GTFS spec, but a missing value is a warning, not an error; see Warnings.
	AgencyTimezone tt.Timezone `csv:",required"`
	AgencyLang     tt.Language
	AgencyPhone    tt.String
	AgencyFareURL  tt.Url
	AgencyEmail    tt.Email
	CEMVSupport    tt.Int `enum:"0,1,2"`
	tt.BaseEntity
}

// EntityID returns the ID or AgencyID.
func (ent *Agency) EntityID() string {
	return entID(ent.ID, ent.AgencyID.Val)
}

// EntityKey returns the GTFS identifier.
func (ent *Agency) EntityKey() string {
	return ent.AgencyID.Val
}

// Warnings returns warnings for this entity.
//
// agency_url is required by the GTFS spec, but treating a missing value as a
// hard error causes the agency to be dropped on import, which cascades: routes
// then reference an unknown agency_id and are dropped, trips reference the
// now-missing routes and are dropped, and the feed fails the required-minimum
// entities check. Demote the missing value to a warning so the rest of the feed
// still imports. A malformed (non-empty) agency_url remains an error via Url.Check.
func (ent *Agency) Warnings() (warns []error) {
	warns = append(warns, tt.CheckPresent("agency_url", ent.AgencyURL.Val)...)
	return warns
}

// Filename agency.txt
func (ent *Agency) Filename() string {
	return "agency.txt"
}

// TableName gtfs_agencies
func (ent *Agency) TableName() string {
	return "gtfs_agencies"
}
