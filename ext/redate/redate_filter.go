package redate

import (
	"fmt"
	"time"

	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/tl"
)

func init() {
	e := func() ext.Extension { return &RedateFilter{} }
	ext.RegisterExtension("redate", e)
}

type RedateFilter struct {
	StartDate  time.Time
	TargetDate time.Time
	TargetDays int
}

func (tf *RedateFilter) Filter(ent tl.Entity, emap *tl.EntityMap) error {
	v, ok := ent.(*tl.Service)
	if !ok {
		return nil
	}
	// Copy active service days in window into new calendar
	startDate := tf.StartDate
	targetDate := tf.TargetDate
	targetDays := 31
	newSvc := tl.NewService(tl.Calendar{ServiceID: v.ServiceID, StartDate: targetDate})
	active := false
	for i := 0; i < targetDays-1; i++ {
		if v.IsActive(startDate) {
			newSvc.AddCalendarDate(tl.CalendarDate{Date: targetDate, ExceptionType: 1})
			active = true
		}
		startDate = startDate.AddDate(0, 0, 1)
		targetDate = targetDate.AddDate(0, 0, 1)
	}
	newSvc.EndDate = targetDate
	if !active {
		return fmt.Errorf("service not in window")
	}
	// Simplify back to regular calendar
	newSvc, err := newSvc.Simplify()
	if err != nil {
		panic(err)
	}
	// Reset and update in place
	v.Reset()
	v.Calendar = newSvc.Calendar
	for _, cd := range newSvc.CalendarDates() {
		v.AddCalendarDate(cd)
	}
	return nil
}
