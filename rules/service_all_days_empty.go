package rules

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tt"
)

// ServiceAllDaysEmptyCheck checks if a calendar.txt entry, non-generated, has at least one day of week marked as 1.
type ServiceAllDaysEmptyCheck struct{}

// Validate .
func (e *ServiceAllDaysEmptyCheck) Validate(ent tt.Entity) []error {
	// Note: Calendar/CalendarDates are validated as Services.
	if v, ok := ent.(*service.Service); ok && !v.Generated {
		days := v.Monday + v.Tuesday + v.Wednesday + v.Thursday + v.Friday + v.Saturday + v.Sunday
		if days == 0 {
			return []error{causes.NewValidationWarning("monday", "at least one day of the week should be set")}
		}
	}
	return nil
}
