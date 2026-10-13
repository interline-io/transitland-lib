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
		// A period that ends at or before its start is an error, which the
		// frequency reports itself.
		st, et := v.StartTime.Int(), v.EndTime.Int()
		if v.StartTime.Valid && v.EndTime.Valid && et > st && et-st < v.HeadwaySecs.Int() {
			errs = append(errs, causes.NewValidationWarning("end_time", "end_time is less than start_time + headway_secs"))
		}
		return errs
	}
	return nil
}
