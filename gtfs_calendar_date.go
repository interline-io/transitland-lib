package gotransit

import (
	"fmt"
	"time"

	"github.com/interline-io/gotransit/causes"
)

// CalendarDate calendar_dates.txt
type CalendarDate struct {
	ServiceID     string    `csv:"service_id" required:"true"`
	Date          time.Time `csv:"date" required:"true"`
	ExceptionType int       `csv:"exception_type" required:"true" min:"1" max:"2"`
	BaseEntity
}

// EntityID returns nothing, CalendarDates are not unique.
func (ent *CalendarDate) EntityID() string {
	return ""
}

// Errors for this Entity.
func (ent *CalendarDate) Errors() (errs []error) {
	errs = ValidateTags(ent)
	errs = append(errs, ent.BaseEntity.loadErrors...)
	if ent.Date.IsZero() {
		errs = append(errs, causes.NewInvalidFieldError("date", "", fmt.Errorf("date is zero")))
	}
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
	if serviceID, ok := emap.GetEntity(&Calendar{ServiceID: ent.ServiceID}); ok {
		ent.ServiceID = serviceID
	} else {
		return causes.NewInvalidReferenceError("service_id", ent.ServiceID)
	}
	return nil
}
