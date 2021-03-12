package tl

import (
	"fmt"
	"time"

	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/enum"
)

// Calendar calendars.txt
type Calendar struct {
	ServiceID string    `csv:"service_id,required" required:"true"`
	Monday    int       `csv:"monday,required" required:"true"`
	Tuesday   int       `csv:"tuesday,required" required:"true"`
	Wednesday int       `csv:"wednesday,required" required:"true"`
	Thursday  int       `csv:"thursday,required" required:"true"`
	Friday    int       `csv:"friday,required" required:"true"`
	Saturday  int       `csv:"saturday,required" required:"true"`
	Sunday    int       `csv:"sunday,required" required:"true"`
	StartDate time.Time `csv:"start_date,required" required:"true"`
	EndDate   time.Time `csv:"end_date,required" required:"true"`
	Generated bool      `csv:"-" db:"generated"`
	BaseEntity
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
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, enum.CheckPresent("service_id", ent.ServiceID)...)
	errs = append(errs, enum.CheckInsideRangeInt("monday", ent.Monday, 0, 1)...)
	errs = append(errs, enum.CheckInsideRangeInt("tuesday", ent.Tuesday, 0, 1)...)
	errs = append(errs, enum.CheckInsideRangeInt("wednesday", ent.Wednesday, 0, 1)...)
	errs = append(errs, enum.CheckInsideRangeInt("thursday", ent.Thursday, 0, 1)...)
	errs = append(errs, enum.CheckInsideRangeInt("friday", ent.Friday, 0, 1)...)
	errs = append(errs, enum.CheckInsideRangeInt("saturday", ent.Saturday, 0, 1)...)
	errs = append(errs, enum.CheckInsideRangeInt("sunday", ent.Sunday, 0, 1)...)
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
