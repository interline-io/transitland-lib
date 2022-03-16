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
	StartDate           time.Time
	StartDays           int
	TargetDate          time.Time
	TargetDays          int
	RemoveOutsideWindow bool
}

func NewRedateFilter(startDate, targetDate time.Time, startDays, targetDays int) *RedateFilter {
	return &RedateFilter{
		StartDate:  startDate,
		StartDays:  startDays,
		TargetDate: targetDate,
		TargetDays: targetDays,
	}
}

func (tf *RedateFilter) Filter(ent tl.Entity, emap *tl.EntityMap) error {
	v, ok := ent.(*tl.Service)
	if !ok {
		return nil
	}
	// Copy active service days in window into new calendar
	active := false
	startDate := tf.StartDate
	targetDate := tf.TargetDate
	newSvc := tl.NewService(tl.Calendar{ServiceID: v.ServiceID, StartDate: targetDate})
	newSvc.ID = v.ID
	fmt.Printf("%#v\n", v.Calendar)
	for i := 1; i < tf.TargetDays; i++ {
		a := false
		if v.IsActive(startDate) {
			newSvc.AddCalendarDate(tl.CalendarDate{Date: targetDate, ExceptionType: 1})
			active = true
			a = true
		}
		fmt.Println(
			"svcId:", newSvc.ServiceID,
			"startDate:", startDate,
			startDate.Weekday().String(),
			"targetDate:", targetDate,
			targetDate.Weekday().String(),
			"i:", i,
			"a:", a,
		)
		startDate = tf.StartDate.AddDate(0, 0, i%tf.StartDays) // startDate.AddDate(0, 0, 1)
		targetDate = tf.TargetDate.AddDate(0, 0, i)
	}
	newSvc.EndDate = targetDate
	if !active && tf.RemoveOutsideWindow {
		return fmt.Errorf("service not in window")
	}
	// Simplify back to regular calendar
	newSvc, err := newSvc.Simplify()
	if err != nil {
		return err
	}
	newSvc.Generated = false
	// Reset and update in place
	v.Reset()
	v.Calendar = newSvc.Calendar
	for _, cd := range newSvc.CalendarDates() {
		v.AddCalendarDate(cd)
	}
	return nil
}
