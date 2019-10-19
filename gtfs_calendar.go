package gotransit

import (
	"fmt"
	"time"

	"github.com/interline-io/gotransit/causes"
)

// Calendar calendars.txt
type Calendar struct {
	ServiceID string    `csv:"service_id" required:"true"`
	Monday    int       `csv:"monday" required:"true" min:"0" max:"1"`
	Tuesday   int       `csv:"tuesday" required:"true" min:"0" max:"1"`
	Wednesday int       `csv:"wednesday" required:"true" min:"0" max:"1"`
	Thursday  int       `csv:"thursday" required:"true" min:"0" max:"1"`
	Friday    int       `csv:"friday" required:"true" min:"0" max:"1"`
	Saturday  int       `csv:"saturday" required:"true" min:"0" max:"1"`
	Sunday    int       `csv:"sunday" required:"true" min:"0" max:"1"`
	StartDate time.Time `csv:"start_date" required:"true" min:"0" max:"1"`
	EndDate   time.Time `csv:"end_date" required:"true" min:"0" max:"1"`
	Generated bool      `db:"generated"`
	BaseEntity
}

// EntityID returns the ID or ServiceID.
func (ent *Calendar) EntityID() string {
	return entID(ent.ID, ent.ServiceID)
}

// Warnings for this Entity.
func (ent *Calendar) Warnings() (errs []error) {
	diff := ent.EndDate.Sub(ent.StartDate).Hours()
	if ent.Monday == 0 && ent.Tuesday == 0 && ent.Wednesday == 0 && ent.Thursday == 0 && ent.Friday == 0 && ent.Saturday == 0 && ent.Sunday == 0 {
		errs = append(errs, causes.NewValidationWarning("", "all days are empty"))
	}
	if diff <= 24 {
		errs = append(errs, causes.NewValidationWarning("", "only covers a single day"))
	}
	return errs
}

// Errors for this Entity.
func (ent *Calendar) Errors() (errs []error) {
	errs = ValidateTags(ent)
	errs = append(errs, ent.BaseEntity.loadErrors...)
	if ent.EndDate.Before(ent.StartDate) {
		errs = append(errs, causes.NewInvalidFieldError("end_date", "", fmt.Errorf("end_date '%s' must come after start_date '%s'", ent.EndDate, ent.StartDate)))
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
