package redate

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/tl"
)

func init() {
	e := func(args string) (ext.Extension, error) {
		opts := &redateOptions{}
		if err := json.Unmarshal([]byte(args), opts); err != nil {
			return nil, err
		}
		a, _ := opts.SourceDays.Int64()
		b, _ := opts.TargetDays.Int64()
		return NewRedateFilter(opts.SourceDate.Time, opts.TargetDate.Time, int(a), int(b))
	}
	ext.RegisterExtension("redate", e)
}

type redateOptions struct {
	SourceDate    tl.Date
	SourceDays    json.Number
	TargetDate    tl.Date
	TargetDays    json.Number
	AllowInactive bool
}

type RedateFilter struct {
	SourceDate    time.Time
	SourceDays    int
	TargetDate    time.Time
	TargetDays    int
	AllowInactive bool
}

func NewRedateFilter(sourceDate, targetDate time.Time, SourceDays, targetDays int) (*RedateFilter, error) {
	r := RedateFilter{
		SourceDate: sourceDate,
		SourceDays: SourceDays,
		TargetDate: targetDate,
		TargetDays: targetDays,
	}
	if r.SourceDate.IsZero() {
		return nil, errors.New("SourceDate not provided")
	}
	if r.TargetDate.IsZero() {
		return nil, errors.New("TargetDate not provided")
	}
	if r.SourceDays <= 0 {
		return nil, errors.New("SourceDays must be > 0")
	}
	if r.TargetDays <= 0 {
		return nil, errors.New("TargetDays must be > 0")
	}
	if r.SourceDate.Weekday() != r.TargetDate.Weekday() {
		return nil, errors.New("SourceDate and TargetDate must be same day of week")
	}
	return &r, nil
}

func (tf *RedateFilter) Filter(ent tl.Entity, emap *tl.EntityMap) error {
	v, ok := ent.(*tl.Service)
	if !ok {
		return nil
	}
	// Copy active service days in window into new calendar
	active := false
	SourceDate := tf.SourceDate
	targetDate := tf.TargetDate
	newSvc := tl.NewService(tl.Calendar{ServiceID: v.ServiceID, StartDate: targetDate})
	newSvc.ID = v.ID
	for i := 1; i <= tf.TargetDays; i++ {
		if v.IsActive(SourceDate) {
			newSvc.AddCalendarDate(tl.CalendarDate{Date: targetDate, ExceptionType: 1})
			active = true
		}
		// fmt.Println(
		// 	"svcId:", newSvc.ServiceID,
		// 	"SourceDate:", SourceDate,
		// 	SourceDate.Weekday().String(),
		// 	"targetDate:", targetDate,
		// 	targetDate.Weekday().String(),
		// 	"i:", i,
		// 	"a:", a,
		// )
		SourceDate = tf.SourceDate.AddDate(0, 0, i%tf.SourceDays)
		targetDate = tf.TargetDate.AddDate(0, 0, i)
	}
	newSvc.EndDate = tf.TargetDate.AddDate(0, 0, tf.TargetDays-1)
	if !active && !tf.AllowInactive {
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
