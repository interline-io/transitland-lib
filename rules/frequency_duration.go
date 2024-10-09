package rules

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

// FrequencyDurationCheck reports when a frequencies.txt entry has (start_time,end_time) less than a full headway.
type FrequencyDurationCheck struct{}

// Validate .
func (e *FrequencyDurationCheck) Validate(ent tl.Entity) []error {
	if v, ok := ent.(*tl.Frequency); ok {
		var errs []error
		st, et := v.StartTime.Int(), v.EndTime.Int()
		if st != 0 && et != 0 {
			if st == et {
				errs = append(errs, causes.NewValidationWarning("end_time", "end_time is equal to start_time"))
			} else if et > st && (et-st) < v.HeadwaySecs {
				errs = append(errs, causes.NewValidationWarning("end_time", "end_time is less than start_time + headway_secs"))
			}
		}
		return errs
	}
	return nil
}
