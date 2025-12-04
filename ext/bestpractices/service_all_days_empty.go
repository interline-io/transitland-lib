package bestpractices

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// ServiceAllDaysEmptyCheck checks if a calendar.txt entry, non-generated, has at least one day of week marked as 1.
type ServiceAllDaysEmptyCheck struct{}

// Validate .
func (e *ServiceAllDaysEmptyCheck) Validate(ent tt.Entity) []error {
	// Note: Calendar/CalendarDates are validated as Services.
	if v, ok := ent.(*gtfs.Calendar); ok && !v.Generated.Val {
		days := v.Monday.Val + v.Tuesday.Val + v.Wednesday.Val + v.Thursday.Val + v.Friday.Val + v.Saturday.Val + v.Sunday.Val
		if days == 0 {
			return []error{causes.NewValidationWarning("monday", "at least one day of the week should be set")}
		}
	}
	return nil
}
