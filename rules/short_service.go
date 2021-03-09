package rules

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

// ShortServiceCheck checks that a calendar.txt entry has (start_date,end_date) of more than 1 day.
type ShortServiceCheck struct{}

// Validate .
func (e *ShortServiceCheck) Validate(ent tl.Entity) []error {
	// Note: Calendar/CalendarDates are validated as Services.
	if v, ok := ent.(*tl.Service); ok {
		if diff := v.EndDate.Sub(v.StartDate).Hours(); diff >= 0 && diff <= 24 {
			return []error{causes.NewValidationWarning("end_date", "covers one day or less")}
		}
	}
	return nil
}
