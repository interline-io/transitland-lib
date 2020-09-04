package tl

import (
	"time"
)

// Service is a Calendar / CalendarDate union.
type Service struct {
	AddedDates  []time.Time
	ExceptDates []time.Time
	Calendar
}

// NewService returns a new Service.
func NewService(c Calendar, cds ...CalendarDate) *Service {
	s := Service{Calendar: c}
	s.AddedDates = []time.Time{}
	s.ExceptDates = []time.Time{}
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
	if cd.ExceptionType == 1 {
		s.AddedDates = append(s.AddedDates, cd.Date)
	} else if cd.ExceptionType == 2 {
		s.ExceptDates = append(s.ExceptDates, cd.Date)
	}
}

// ServicePeriod returns the widest possible range of days with transit service, including service exceptions.
func (s *Service) ServicePeriod() (time.Time, time.Time) {
	start, end := s.StartDate, s.EndDate
	for _, d := range s.AddedDates {
		if d.Before(start) {
			start = d
		}
		if d.After(end) {
			end = d
		}
	}
	return start, end
}

// IsActive returns if this Service period is active on a specified date.
func (s *Service) IsActive(t time.Time) bool {
	y1, m1, d1 := t.Date()
	for _, cd := range s.AddedDates {
		y2, m2, d2 := cd.Date()
		if y1 == y2 && m1 == m2 && d1 == d2 {
			return true
		}
	}
	for _, cd := range s.ExceptDates {
		y2, m2, d2 := cd.Date()
		if y1 == y2 && m1 == m2 && d1 == d2 {
			return false
		}
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
