package gtfs

import (
	"fmt"
	"time"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// CalendarDate calendar_dates.txt
type CalendarDate struct {
	ServiceID     string    `csv:",required"`
	Date          time.Time `csv:",required"`
	ExceptionType int       `csv:",required"`
	tt.BaseEntity
}

// Errors for this Entity.
func (ent *CalendarDate) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, tt.CheckPresent("service_id", ent.ServiceID)...)
	errs = append(errs, tt.CheckInsideRangeInt("exception_type", ent.ExceptionType, 1, 2)...)
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
