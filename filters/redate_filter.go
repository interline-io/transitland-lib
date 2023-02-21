package filters

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/interline-io/transitland-lib/tl"
)

type RedateFilter struct {
	SourceDate    time.Time
	SourceDays    int
	TargetDate    time.Time
	TargetDays    int
	AllowInactive bool
	excluded      map[string]bool
}

func NewRedateFilter(sourceDate, targetDate time.Time, SourceDays, targetDays int) (*RedateFilter, error) {
	r := RedateFilter{
		SourceDate: sourceDate,
		SourceDays: SourceDays,
		TargetDate: targetDate,
		TargetDays: targetDays,
		excluded:   map[string]bool{},
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

func newRedateFilterFromJson(args string) (*RedateFilter, error) {
	type redateOptions struct {
		SourceDate    tl.Date
		SourceDays    json.Number
		TargetDate    tl.Date
		TargetDays    json.Number
		AllowInactive bool
	}
	opts := &redateOptions{}
	if err := json.Unmarshal([]byte(args), opts); err != nil {
		return nil, err
	}
	a, _ := opts.SourceDays.Int64()
	b, _ := opts.TargetDays.Int64()
	return NewRedateFilter(opts.SourceDate.Val, opts.TargetDate.Val, int(a), int(b))
}

func (tf *RedateFilter) Filter(ent tl.Entity, emap *tl.EntityMap) error {
	switch v := ent.(type) {
	case *tl.Trip:
		if tf.excluded[v.ServiceID] {
			return fmt.Errorf("trip service_id not in redate window")
		}
	case *tl.CalendarDate:
		if tf.excluded[v.ServiceID] {
			return fmt.Errorf("calendar date service_id not in redate window")
		}
	case *tl.Service:
		// Copy active service days in window into new calendar
		active := false
		sourceDate := tf.SourceDate
		targetDate := tf.TargetDate
		newSvc := tl.NewService(tl.Calendar{ServiceID: v.ServiceID, StartDate: targetDate})
		newSvc.ID = v.ID
		for i := 1; i <= tf.TargetDays; i++ {
			if v.IsActive(sourceDate) {
				newSvc.AddCalendarDate(tl.CalendarDate{Date: targetDate, ExceptionType: 1})
				active = true
			}
			sourceDate = tf.SourceDate.AddDate(0, 0, i%tf.SourceDays)
			targetDate = tf.TargetDate.AddDate(0, 0, i)
		}
		newSvc.EndDate = tf.TargetDate.AddDate(0, 0, tf.TargetDays-1)
		if !active && !tf.AllowInactive {
			tf.excluded[v.ServiceID] = true
			return fmt.Errorf("service not in redate window")
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
	return nil
}
