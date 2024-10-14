package gtfs

import (
	"fmt"
	"time"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// Calendar calendars.txt
type Calendar struct {
	ServiceID string    `csv:",required"`
	Monday    int       `csv:",required"`
	Tuesday   int       `csv:",required"`
	Wednesday int       `csv:",required"`
	Thursday  int       `csv:",required"`
	Friday    int       `csv:",required"`
	Saturday  int       `csv:",required"`
	Sunday    int       `csv:",required"`
	StartDate time.Time `csv:",required"`
	EndDate   time.Time `csv:",required"`
	Generated bool      `csv:"-" db:"generated"`
	tt.BaseEntity
}

// EntityID returns the ID or ServiceID.
func (ent *Calendar) EntityID() string {
	return entID(ent.ID, ent.ServiceID)
}

// EntityKey returns the GTFS identifier.
func (ent *Calendar) EntityKey() string {
	return ent.ServiceID
}

// Errors for this Entity.
func (ent *Calendar) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.LoadErrors()...)
	errs = append(errs, tt.CheckPresent("service_id", ent.ServiceID)...)
	errs = append(errs, tt.CheckInsideRangeInt("monday", ent.Monday, 0, 1)...)
	errs = append(errs, tt.CheckInsideRangeInt("tuesday", ent.Tuesday, 0, 1)...)
	errs = append(errs, tt.CheckInsideRangeInt("wednesday", ent.Wednesday, 0, 1)...)
	errs = append(errs, tt.CheckInsideRangeInt("thursday", ent.Thursday, 0, 1)...)
	errs = append(errs, tt.CheckInsideRangeInt("friday", ent.Friday, 0, 1)...)
	errs = append(errs, tt.CheckInsideRangeInt("saturday", ent.Saturday, 0, 1)...)
	errs = append(errs, tt.CheckInsideRangeInt("sunday", ent.Sunday, 0, 1)...)
	if ent.StartDate.IsZero() {
		errs = append(errs, causes.NewInvalidFieldError("start_date", ent.StartDate.String(), fmt.Errorf("start_date is empty")))
	}
	if ent.EndDate.IsZero() {
		errs = append(errs, causes.NewInvalidFieldError("end_date", ent.EndDate.String(), fmt.Errorf("end_date is empty")))
	} else if ent.EndDate.Before(ent.StartDate) {
		errs = append(errs, causes.NewInvalidFieldError("end_date", ent.EndDate.String(), fmt.Errorf("end_date '%s' must come after start_date '%s'", ent.EndDate, ent.StartDate)))
	}
	return errs
}

// Filename calendar.txt
func (ent *Calendar) Filename() string {
	return "calendar.txt"
}

// TableName gtfs_calendars
func (ent *Calendar) TableName() string {
	return "gtfs_calendars"
}
