package tl

import (
	"fmt"
	"time"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/enums"
)

// CalendarDate calendar_dates.txt
type CalendarDate struct {
	ServiceID     string    `csv:"service_id" required:"true"`
	Date          time.Time `csv:"date" required:"true"`
	ExceptionType int       `csv:"exception_type" required:"true"`
	BaseEntity
}

// EntityID returns nothing, CalendarDates are not unique.
func (ent *CalendarDate) EntityID() string {
	return ""
}

// Errors for this Entity.
func (ent *CalendarDate) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, enums.CheckPresent("service_id", ent.ServiceID)...)
	errs = append(errs, enums.CheckInsideRangeInt("exception_type", ent.ExceptionType, 1, 2)...)
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
