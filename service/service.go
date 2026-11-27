// This package is for compound entities, e.g. calendar services and shape lines
package service

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// ymd for use as a map key
type ymd struct {
	year  int32
	month int8
	day   int8
}

func newYMD(t time.Time) ymd {
	y, m, d := t.Date()
	return ymd{int32(y), int8(m), int8(d)}
}

func (d *ymd) Before(other ymd) bool {
	return d.num() < other.num()
}

func (d *ymd) After(other ymd) bool {
	return d.num() > other.num()
}

// num is the date as yyyymmdd.
func (d ymd) num() int {
	return int(d.year)*10000 + int(d.month)*100 + int(d.day)
}

func (d *ymd) Time() time.Time {
	return time.Date(int(d.year), time.Month(d.month), int(d.day), 0, 0, 0, 0, time.UTC)
}

func (d *ymd) IsZero() bool {
	return d.year <= 1 && d.month <= 1 && d.day <= 1
}

func (d *ymd) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.year, d.month, d.day)
}

// NewServicesFromReader returns the reader's Calendar and CalendarDate entities as a Service
func NewServicesFromReader(reader adapters.Reader) []*Service {
	ret := []*Service{}
	cds := map[string][]gtfs.CalendarDate{}
	for cd := range reader.CalendarDates() {
		sid := cd.ServiceID.Val
		cds[sid] = append(cds[sid], cd)
	}
	for c := range reader.Calendars() {
		sid := c.ServiceID.Val
		s := NewService(c, cds[sid]...)
		ret = append(ret, s)
		// remove from list of calendar.txt entries
		delete(cds, sid)
	}
	for k, v := range cds {
		// remaining calendar dates without calendar.txt
		s := NewService(gtfs.Calendar{ServiceID: tt.NewString(k)}, v...)
		a, b := s.ServicePeriod()
		s.StartDate.Set(a)
		s.EndDate.Set(b)
		s.Generated.Set(true)
		s.Monday.OrSet(0)
		s.Tuesday.OrSet(0)
		s.Wednesday.OrSet(0)
		s.Thursday.OrSet(0)
		s.Friday.OrSet(0)
		s.Saturday.OrSet(0)
		s.Sunday.OrSet(0)
		ret = append(ret, s)
	}
	return ret
}

// Service is a Calendar / CalendarDate union.
type Service struct {
	added   []ymd // calendar dates with exception type 1
	removed []ymd // calendar dates with exception type 2
	gtfs.Calendar
}

// NewService returns a new Service. It leaves out a calendar date whose
// exception type is neither 1 nor 2.
func NewService(c gtfs.Calendar, cds ...gtfs.CalendarDate) *Service {
	// A Service keeps its dates as added and removed, so it drops the calendar
	// dates the copier attached, which are many times their size.
	c.CalendarDates = nil
	s := Service{Calendar: c}
	for _, cd := range cds {
		s.AddCalendarDate(cd)
	}
	return &s
}

// Reset resets calendars, except ServiceID
func (s *Service) Reset() {
	sid := s.ServiceID
	s.Calendar = gtfs.Calendar{ServiceID: sid}
	s.added, s.removed = nil, nil
}

// AddCalendarDate adds a service exception, of exception type 1 or 2.
func (s *Service) AddCalendarDate(cd gtfs.CalendarDate) error {
	switch d := newYMD(cd.Date.Val); cd.ExceptionType.Int() {
	case 1:
		s.added = append(s.added, d)
	case 2:
		s.removed = append(s.removed, d)
	default:
		return fmt.Errorf("invalid exception_type %d", cd.ExceptionType.Int())
	}
	return nil
}

// CalendarDates returns CalendarDates for this service: the added dates, then
// the removed ones.
func (s *Service) CalendarDates() []gtfs.CalendarDate {
	ret := make([]gtfs.CalendarDate, 0, len(s.added)+len(s.removed))
	for _, d := range s.added {
		ret = append(ret, s.calendarDate(d, 1))
	}
	for _, d := range s.removed {
		ret = append(ret, s.calendarDate(d, 2))
	}
	return ret
}

func (s *Service) calendarDate(d ymd, exceptionType int) gtfs.CalendarDate {
	return gtfs.CalendarDate{
		ServiceID:     tt.NewKey(s.EntityID()),
		Date:          tt.NewDate(d.Time()),
		ExceptionType: tt.NewInt(exceptionType),
	}
}

// GetWeekday returns the value fo the day of week.
func (s *Service) GetWeekday(dow int) (int, error) {
	var v tt.Int
	switch dow {
	case 0:
		v = s.Sunday
	case 1:
		v = s.Monday
	case 2:
		v = s.Tuesday
	case 3:
		v = s.Wednesday
	case 4:
		v = s.Thursday
	case 5:
		v = s.Friday
	case 6:
		v = s.Saturday
	default:
		return 0, errors.New("unknown weekday")
	}
	return v.Int(), nil
}

// SetWeekday sets the day of week.
func (s *Service) SetWeekday(dow int, value int) error {
	if value < 0 || value > 1 {
		return errors.New("only 0,1 allowed")
	}
	v := int64(value)
	switch dow {
	case 0:
		s.Sunday.Set(v)
	case 1:
		s.Monday.Set(v)
	case 2:
		s.Tuesday.Set(v)
	case 3:
		s.Wednesday.Set(v)
	case 4:
		s.Thursday.Set(v)
	case 5:
		s.Friday.Set(v)
	case 6:
		s.Saturday.Set(v)
	default:
		return errors.New("unknown weekday")
	}
	return nil
}

// ServicePeriod returns the widest possible range of days with transit service, including service exceptions.
func (s *Service) ServicePeriod() (time.Time, time.Time) {
	start, end := newYMD(s.StartDate.Val), newYMD(s.EndDate.Val)
	for _, dates := range [][]ymd{s.added, s.removed} {
		for _, d := range dates {
			if start.IsZero() || d.Before(start) {
				start = d
			}
			if end.IsZero() || d.After(end) {
				end = d
			}
		}
	}
	return start.Time(), end.Time()
}

// Exception returns the exception type on t's calendar date, if the service
// has one.
func (s *Service) Exception(t time.Time) (int, bool) {
	return s.exceptionOn(newYMD(t))
}

func (s *Service) exceptionOn(d ymd) (int, bool) {
	if slices.Contains(s.added, d) {
		return 1, true
	}
	if slices.Contains(s.removed, d) {
		return 2, true
	}
	return 0, false
}

// IsActive returns if this Service period is active on a specified date: t's
// calendar date, in t's location.
func (s *Service) IsActive(t time.Time) bool {
	// StartDate and EndDate hold midnight UTC, so the bounds compare calendar
	// dates, not instants.
	day := newYMD(t)
	if v, ok := s.exceptionOn(day); ok {
		return v == 1
	}
	if day.Before(newYMD(s.StartDate.Val)) {
		return false
	}
	if day.After(newYMD(s.EndDate.Val)) {
		return false
	}
	v, err := s.GetWeekday(int(t.Weekday()))
	if err != nil {
		return false
	}
	return v == 1
}

// HasAtLeastOneDay checks if the Service is active for at least one day.
func (s *Service) HasAtLeastOneDay() bool {
	// Quick checks before iterating through each day.
	// quick check that we have at least a week of service,
	// otherwise fall back to full check...
	duration := s.EndDate.Val.Sub(s.StartDate.Val).Hours() / 24
	days := s.Monday.Val + s.Tuesday.Val + s.Wednesday.Val + s.Thursday.Val + s.Friday.Val + s.Saturday.Val + s.Sunday.Val
	if duration >= 7 && days > 0 && len(s.added)+len(s.removed) == 0 {
		return true
	}
	// By definition, at least one day
	if len(s.added) > 0 {
		return true
	}
	// By definition, no days
	if days == 0 {
		return false
	}
	// Now unfortunately have to check every day
	start, end := s.ServicePeriod()
	for start.Before(end) || start.Equal(end) {
		if s.IsActive(start) {
			return true
		}
		start = start.AddDate(0, 0, 1)
	}
	return false
}

// Simplify tries to simplify exceptions down to a basic calendar with fewer exceptions.
func (s *Service) Simplify() (*Service, error) {
	inputServiceStart, inputServiceEnd := s.StartDate.Val, s.EndDate.Val
	if s.Generated.Val || s.StartDate.IsZero() || s.EndDate.IsZero() {
		inputServiceStart, inputServiceEnd = s.ServicePeriod()
	}
	// Count the total days and active days, by day of week
	totalCount := map[int]int{}
	activeCount := map[int]int{}
	addedCount := map[int]int{}
	removedCount := map[int]int{}
	start, end := inputServiceStart, inputServiceEnd
	for start.Before(end) || start.Equal(end) {
		dow := int(start.Weekday())
		totalCount[dow]++
		if s.IsActive(start) {
			activeCount[dow]++
		}
		if etype, ok := s.Exception(start); ok {
			if etype == 1 {
				addedCount[dow]++
			} else if etype == 2 {
				removedCount[dow]++
			}
		}
		start = start.AddDate(0, 0, 1)
	}

	// Set weekdays based on dow counts
	ret := NewService(gtfs.Calendar{
		ServiceID: s.ServiceID,
		Generated: s.Generated,
		StartDate: tt.NewDate(inputServiceStart),
		EndDate:   tt.NewDate(inputServiceEnd),
		Monday:    tt.NewInt(0),
		Tuesday:   tt.NewInt(0),
		Wednesday: tt.NewInt(0),
		Thursday:  tt.NewInt(0),
		Friday:    tt.NewInt(0),
		Saturday:  tt.NewInt(0),
		Sunday:    tt.NewInt(0),
	})
	ret.ID = s.ID
	for dow, total := range totalCount {
		if total == 0 {
			continue
		}
		active := activeCount[dow]
		willBeAdded := active           // if 0, then add
		willBeRemoved := total - active // if 1, then remove
		if total == active || willBeAdded >= willBeRemoved {
			ret.SetWeekday(dow, 1)
		}
	}

	// Add exceptions
	start, end = s.ServicePeriod() // check over the entire service range
	for start.Before(end) || start.Equal(end) {
		a := s.IsActive(start)
		b := ret.IsActive(start)
		if a && b {
			// both are active
		} else if a && !b {
			// existing is active, new is not active
			ret.AddCalendarDate(gtfs.CalendarDate{
				Date:          tt.NewDate(start),
				ExceptionType: tt.NewInt(1),
			})
		} else if !a && b {
			// existing is inactive, new is active
			ret.AddCalendarDate(gtfs.CalendarDate{
				Date:          tt.NewDate(start),
				ExceptionType: tt.NewInt(2),
			})
		}
		start = start.AddDate(0, 0, 1)
	}
	return ret, nil
}
