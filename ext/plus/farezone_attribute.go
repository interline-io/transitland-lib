package plus

import "github.com/interline-io/transitland-lib/tt"

// FarezoneAttribute farezone_attributes.txt
type FarezoneAttribute struct {
	ZoneID   string `csv:"zone_id"`
	ZoneName string `csv:"zone_name"`
	tt.BaseEntity
}

// Filename farezone_attributes.txt
func (ent *FarezoneAttribute) Filename() string {
	return "farezone_attributes.txt"
}

// TableName ext_plus_fare_attributes
func (ent *FarezoneAttribute) TableName() string {
	return "ext_plus_farezone_attributes"
}
