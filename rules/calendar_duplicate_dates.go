package rules

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

type CalendarDuplicateDates struct {
}

func (e *CalendarDuplicateDates) Validate(ent tl.Entity) []error {
	svc, ok := ent.(*tl.Service)
	if !ok {
		return nil
	}
	var errs []error
	hits := map[string]bool{}
	for _, cd := range svc.CalendarDates() {
		k := cd.Date.Format("20060102")
		if _, ok := hits[k]; ok {
			errs = append(errs, causes.NewDuplicateServiceExceptionError(svc.ServiceID, cd.Date))
		}
		hits[k] = true
	}
	return errs
}
