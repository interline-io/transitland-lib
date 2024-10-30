package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// CalendarDate calendar_dates.txt
type CalendarDate struct {
	ServiceID     tt.Key  `csv:",required" target:"calendar.txt"`
	Date          tt.Date `csv:",required"`
	ExceptionType tt.Int  `csv:",required" enum:"1,2"`
	tt.BaseEntity
}

// Errors for this Entity.
func (ent *CalendarDate) Errors() (errs []error) {
	errs = append(errs, tt.CheckPresent("service_id", ent.ServiceID.Val)...)
	errs = append(errs, tt.CheckInsideRangeInt("exception_type", ent.ExceptionType.Val, 1, 2)...)
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
func (ent *CalendarDate) UpdateKeys(emap *tt.EntityMap) error {
	if serviceID, ok := emap.GetEntity(&Calendar{ServiceID: ent.ServiceID.Val}); ok {
		ent.ServiceID.Set(serviceID)
	} else {
		return causes.NewInvalidReferenceError("service_id", ent.ServiceID.Val)
	}
	return nil
}
