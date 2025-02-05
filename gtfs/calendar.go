package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// Calendar calendar.txt
type Calendar struct {
	ServiceID     tt.String      `csv:",required"`
	Monday        tt.Int         `csv:",required" enum:"0,1"`
	Tuesday       tt.Int         `csv:",required" enum:"0,1"`
	Wednesday     tt.Int         `csv:",required" enum:"0,1"`
	Thursday      tt.Int         `csv:",required" enum:"0,1"`
	Friday        tt.Int         `csv:",required" enum:"0,1"`
	Saturday      tt.Int         `csv:",required" enum:"0,1"`
	Sunday        tt.Int         `csv:",required" enum:"0,1"`
	StartDate     tt.Date        `csv:",required"`
	EndDate       tt.Date        `csv:",required"`
	Generated     tt.Bool        `csv:"-" db:"generated"`
	CalendarDates []CalendarDate `csv:"-" db:"-"` // for validation
	tt.BaseEntity
}

// EntityID returns the ID or ServiceID.
func (ent *Calendar) EntityID() string {
	return entID(ent.ID, ent.ServiceID.Val)
}

// EntityKey returns the GTFS identifier.
func (ent *Calendar) EntityKey() string {
	return ent.ServiceID.Val
}

// Errors for this Entity.
func (ent *Calendar) ConditionalErrors() []error {
	var errs []error
	if !ent.StartDate.IsZero() && !ent.EndDate.IsZero() && ent.EndDate.Before(ent.StartDate) {
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
