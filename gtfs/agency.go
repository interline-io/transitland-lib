package gtfs

import "github.com/interline-io/transitland-lib/tt"

// Agency agency.txt
type Agency struct {
	AgencyID       tt.String   `standardized_sort:"1"`
	AgencyName     tt.String   `csv:",required"`
	AgencyURL      tt.Url      `csv:",required"`
	AgencyTimezone tt.Timezone `csv:",required"`
	AgencyLang     tt.Language
	AgencyPhone    tt.String
	AgencyFareURL  tt.Url
	AgencyEmail    tt.Email
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

// Filename agency.txt
func (ent *Agency) Filename() string {
	return "agency.txt"
}

// TableName gtfs_agencies
func (ent *Agency) TableName() string {
	return "gtfs_agencies"
}
