package gotransit

import (
	"time"

	"github.com/interline-io/gotransit/causes"
)

// CalendarDate calendar_dates.txt
type CalendarDate struct {
	ServiceID     string    `csv:"service_id" required:"true" gorm:"type:int;index;not null"`
	Date          time.Time `csv:"date" required:"true" gorm:"index;not null"`
	ExceptionType int       `csv:"exception_type" required:"true" min:"1" max:"2" gorm:"index;not null"`
	BaseEntity
}

// EntityID returns nothing, CalendarDates are not unique.
func (ent *CalendarDate) EntityID() string {
	return ""
}

// Warnings for this Entity.
func (ent *CalendarDate) Warnings() (errs []error) {
	return errs
}

// Errors for this Entity.
func (ent *CalendarDate) Errors() (errs []error) {
	errs = ValidateTags(ent)
	errs = append(errs, ent.BaseEntity.loadErrors...)
	return errs
}

// Filename calendar_dates.txt
func (ent *CalendarDate) Filename() string {
	return "calendar_dates.txt"
}

// TableName gtfs_calendar_dates
func (ent *CalendarDate) TableName() string {
	return "gtfs_calendar_dates"
}

// UpdateKeys updates Entity references.
func (ent *CalendarDate) UpdateKeys(emap *EntityMap) error {
	if serviceID, ok := emap.Get(&Calendar{ServiceID: ent.ServiceID}); ok {
		ent.ServiceID = serviceID
	} else {
		return causes.NewInvalidReferenceError("service_id", ent.ServiceID)
	}
	return nil
}
