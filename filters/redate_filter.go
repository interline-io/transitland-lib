package filters

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
)

type RedateFilter struct {
	SourceDate    time.Time
	SourceDays    int
	TargetDate    time.Time
	TargetDays    int
	DOWAlign      bool
	AllowInactive bool
	excluded      map[string]bool
}

func NewRedateFilter(sourceDate, targetDate time.Time, SourceDays, targetDays int, dowAlign bool) (*RedateFilter, error) {
	r := RedateFilter{
		SourceDate: sourceDate,
		SourceDays: SourceDays,
		TargetDate: targetDate,
		TargetDays: targetDays,
		DOWAlign:   dowAlign,
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
	if !r.DOWAlign && (r.SourceDate.Weekday() != r.TargetDate.Weekday()) {
		return nil, errors.New("SourceDate and TargetDate must be same day of week, or DOWAlign must be true")
	}
	return &r, nil
}

func newRedateFilterFromJson(args string) (*RedateFilter, error) {
	type redateOptions struct {
		SourceDate    tt.Date
		SourceEndDate tt.Date
		SourceDays    json.Number
		TargetDate    tt.Date
		TargetEndDate tt.Date
		TargetDays    json.Number
		DOWAlign      tt.Bool
		AllowInactive bool
	}
	opts := &redateOptions{}
	if err := json.Unmarshal([]byte(args), opts); err != nil {
		return nil, err
	}
	a, _ := opts.SourceDays.Int64()
	if opts.SourceEndDate.Valid {
		a = int64(daysBetween(opts.SourceDate.Val, opts.SourceEndDate.Val) + 1)
	}
	b, _ := opts.TargetDays.Int64()
	if opts.TargetEndDate.Valid {
		b = int64(daysBetween(opts.TargetDate.Val, opts.TargetEndDate.Val) + 1)
	}
	return NewRedateFilter(opts.SourceDate.Val, opts.TargetDate.Val, int(a), int(b), opts.DOWAlign.Val)
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

		// Align days of week
		alignDays := 0
		if tf.DOWAlign {
			for {
				if sourceDate.Weekday() != targetDate.Weekday() {
					log.Trace().
						Str("source_date", sourceDate.Format("2006-01-02")).
						Str("source_dow", sourceDate.Weekday().String()).
						Str("target_date", targetDate.Format("2006-01-02")).
						Str("target_dow", targetDate.Weekday().String()).
						Int("align_days", alignDays).
						Str("service_id", v.ServiceID).
						Msg("weekday mismatch; shifting source_date forward 1 day")
					sourceDate = sourceDate.AddDate(0, 0, 1)
					alignDays += 1
					continue
				}
				break
			}
		}

		newSvc := tl.NewService(tl.Calendar{ServiceID: v.ServiceID, StartDate: targetDate})
		newSvc.ID = v.ID
		for i := 1; i <= tf.TargetDays; i++ {
			if v.IsActive(sourceDate) {
				newSvc.AddCalendarDate(tl.CalendarDate{Date: targetDate, ExceptionType: 1})
				active = true
			}
			log.Trace().
				Str("source_date", sourceDate.Format("2006-01-02")).
				Str("source_dow", sourceDate.Weekday().String()).
				Str("target_date", targetDate.Format("2006-01-02")).
				Str("target_dow", targetDate.Weekday().String()).
				Int("i", i).
				Int("align_days", alignDays).
				Str("service_id", v.ServiceID).
				Bool("active", active).
				Msg("redate")
			sourceDate = tf.SourceDate.AddDate(0, 0, (alignDays+i)%tf.SourceDays)
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

func daysBetween(t1 time.Time, t2 time.Time) int {
	days := 0
	flip := 1
	if t2.Before(t1) {
		t1, t2 = t2, t1
		flip = -1
	}
	for {
		if t2.Equal(t1) || t2.Before(t1) {
			break
		}
		t1 = t1.AddDate(0, 0, 1)
		days += 1
	}
	return days * flip
}
