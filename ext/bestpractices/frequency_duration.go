package bestpractices

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// FrequencyDurationCheck reports when a frequencies.txt entry has (start_time,end_time) less than a full headway.
type FrequencyDurationCheck struct{}

// Validate .
func (e *FrequencyDurationCheck) Validate(ent tt.Entity) []error {
	if v, ok := ent.(*gtfs.Frequency); ok {
		var errs []error
		st, et := v.StartTime.Int(), v.EndTime.Int()
		if st != 0 && et != 0 {
			if st == et {
				errs = append(errs, causes.NewValidationWarning("end_time", "end_time is equal to start_time"))
			} else if et > st && (et-st) < v.HeadwaySecs.Int() {
				errs = append(errs, causes.NewValidationWarning("end_time", "end_time is less than start_time + headway_secs"))
			}
		}
		return errs
	}
	return nil
}
