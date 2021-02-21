package tl

import (
	"time"
)

// ymd for use as a map key
type ymd struct {
	year  int
	month int
	day   int
}

func newYMD(t time.Time) ymd {
	y, m, d := t.Date()
	return ymd{y, int(m), d}
}

func (d *ymd) Before(other ymd) bool {
	return (d.year*10000)+(d.month*100)+d.day < (other.year*10000)+(other.month*100)+(other.day)
}

func (d *ymd) After(other ymd) bool {
	return (d.year*10000)+(d.month*100)+d.day > (other.year*10000)+(other.month*100)+(other.day)
}

func (d *ymd) Time() time.Time {
	return time.Date(d.year, time.Month(d.month), d.day, 0, 0, 0, 0, time.UTC)
}

func (d *ymd) IsZero() bool {
	return d.year <= 1 && d.month <= 1 && d.day <= 1
}

// Service is a Calendar / CalendarDate union.
type Service struct {
	exceptions map[ymd]int
	Calendar
}

// NewService returns a new Service.
func NewService(c Calendar, cds ...CalendarDate) *Service {
	s := Service{Calendar: c}
	s.exceptions = map[ymd]int{}
	for _, cd := range cds {
		s.AddCalendarDate(cd)
	}
	return &s
}

// NewServicesFromReader returns
func NewServicesFromReader(reader Reader) []*Service {
	ret := []*Service{}
	cds := map[string][]CalendarDate{}
	for cd := range reader.CalendarDates() {
		sid := cd.ServiceID
		cds[sid] = append(cds[sid], cd)
	}
	for c := range reader.Calendars() {
		sid := c.ServiceID
		s := NewService(c, cds[sid]...)
		ret = append(ret, s)
		delete(cds, sid)
	}
	for k, v := range cds {
		s := NewService(Calendar{ServiceID: k}, v...)
		ret = append(ret, s)
	}
	return ret
}

// AddCalendarDate adds a service exception.
func (s *Service) AddCalendarDate(cd CalendarDate) {
	s.exceptions[newYMD(cd.Date)] = cd.ExceptionType
}

// ServicePeriod returns the widest possible range of days with transit service, including service exceptions.
func (s *Service) ServicePeriod() (time.Time, time.Time) {
	start, end := newYMD(s.StartDate), newYMD(s.EndDate)
	for d := range s.exceptions {
		if start.IsZero() || d.Before(start) {
			start = d
		}
		if end.IsZero() || d.After(end) {
			end = d
		}
	}
	return start.Time(), end.Time()
}

// IsActive returns if this Service period is active on a specified date.
func (s *Service) IsActive(t time.Time) bool {
	if etype, ok := s.exceptions[newYMD(t)]; ok {
		if etype == 1 {
			return true
		}
		return false
	}
	if t.Before(s.StartDate) {
		return false
	}
	if t.After(s.EndDate) {
		return false
	}
	switch dow := t.Weekday(); dow {
	case 0:
		return s.Sunday == 1
	case 1:
		return s.Monday == 1
	case 2:
		return s.Tuesday == 1
	case 3:
		return s.Wednesday == 1
	case 4:
		return s.Thursday == 1
	case 5:
		return s.Friday == 1
	case 6:
		return s.Saturday == 1
	}
	return false
}
