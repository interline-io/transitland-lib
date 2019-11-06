package plus

import (
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
)

// CalendarAttribute calendar_attributes.txt
type CalendarAttribute struct {
	ServiceID          string `csv:"service_id"`
	ServiceDescription string `csv:"service_description"`
	gotransit.BaseEntity
}

// Filename calendar_attributes.txt
func (ent *CalendarAttribute) Filename() string {
	return "calendar_attributes.txt"
}

// TableName ext_plus_fare_attributes
func (ent *CalendarAttribute) TableName() string {
	return "ext_plus_calendar_attributes"
}

// UpdateKeys updates Entity references.
func (ent *CalendarAttribute) UpdateKeys(emap *gotransit.EntityMap) error {
	if fkid, ok := emap.GetEntity(&gotransit.Calendar{ServiceID: ent.ServiceID}); ok {
		ent.ServiceID = fkid
	} else if fkid, ok := emap.GetEntity(&gotransit.CalendarDate{ServiceID: ent.ServiceID}); ok {
		ent.ServiceID = fkid
	} else {
		return causes.NewInvalidReferenceError("service_id", ent.ServiceID)
	}
	return nil
}
