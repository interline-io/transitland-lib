package tl

import (
	"fmt"
	"time"

	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/enum"
)

// Calendar calendars.txt
type Calendar struct {
	ServiceID string    `csv:"service_id" required:"true"`
	Monday    int       `csv:"monday" required:"true"`
	Tuesday   int       `csv:"tuesday" required:"true"`
	Wednesday int       `csv:"wednesday" required:"true"`
	Thursday  int       `csv:"thursday" required:"true"`
	Friday    int       `csv:"friday" required:"true"`
	Saturday  int       `csv:"saturday" required:"true"`
	Sunday    int       `csv:"sunday" required:"true"`
	StartDate time.Time `csv:"start_date" required:"true"`
	EndDate   time.Time `csv:"end_date" required:"true"`
	Generated bool      `db:"generated"`
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

// Warnings for this Entity.
func (ent *Calendar) Warnings() (errs []error) {
	errs = append(errs, ent.loadWarnings...)
	// Are all days empty?
	if ent.Monday == 0 && ent.Tuesday == 0 && ent.Wednesday == 0 && ent.Thursday == 0 && ent.Friday == 0 && ent.Saturday == 0 && ent.Sunday == 0 {
		errs = append(errs, causes.NewValidationWarning("", "all days are empty"))
	}
	// Does this cover less than 24 hours? End before start is checked in Errors().
	if diff := ent.EndDate.Sub(ent.StartDate).Hours(); diff >= 0 && diff <= 24 {
		errs = append(errs, causes.NewValidationWarning("", "covers one day or less"))
	}
	return errs
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
