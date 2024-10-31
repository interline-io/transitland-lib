package rules

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

type CalendarDuplicateDates struct {
}

func (e *CalendarDuplicateDates) Validate(ent tt.Entity) []error {
	v, ok := ent.(*gtfs.Calendar)
	if !ok {
		return nil
	}
	var errs []error
	hits := map[string]bool{}
	for _, cd := range v.CalendarDates {
		k := cd.Date.Format("20060102")
		if _, ok := hits[k]; ok {
			errs = append(errs, causes.NewDuplicateServiceExceptionError(v.ServiceID.Val, cd.Date.Val))
		}
		hits[k] = true
	}
	return errs
}
