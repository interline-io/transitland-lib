package tl

import (
	"errors"
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

// Reset resets calendars, except ServiceID
func (s *Service) Reset() {
	sid := s.ServiceID
	s.Calendar = Calendar{ServiceID: sid}
	s.exceptions = map[ymd]int{}
}

// AddCalendarDate adds a service exception.
func (s *Service) AddCalendarDate(cd CalendarDate) {
	s.exceptions[newYMD(cd.Date)] = cd.ExceptionType
}

// CalendarDates returns CalendarDates for this service.
func (s *Service) CalendarDates() []CalendarDate {
	ret := []CalendarDate{}
	for ymd, v := range s.exceptions {
		ret = append(ret, CalendarDate{
			ServiceID:     s.EntityID(),
			Date:          ymd.Time(),
			ExceptionType: v,
		})
	}
	return ret
}

// GetWeekday returns the value fo the day of week.
func (s *Service) GetWeekday(dow int) (int, error) {
	v := 0
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
	return v, nil
}

// SetWeekday sets the day of week.
func (s *Service) SetWeekday(dow int, value int) error {
	if value < 0 || value > 1 {
		return errors.New("only 0,1 allowed")
	}
	switch dow {
	case 0:
		s.Sunday = value
	case 1:
		s.Monday = value
	case 2:
		s.Tuesday = value
	case 3:
		s.Wednesday = value
	case 4:
		s.Thursday = value
	case 5:
		s.Friday = value
	case 6:
		s.Saturday = value
	default:
		return errors.New("unknown weekday")
	}
	return nil
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
	v, err := s.GetWeekday(int(t.Weekday()))
	if err != nil {
		return false
	}
	return v == 1
}

// Exception returns if a calendar exception exists on the given day.
func (s *Service) Exception(t time.Time) (int, bool) {
	a, ok := s.exceptions[newYMD(t)]
	return a, ok
}

// HasAtLeastOneDay checks if the Service is active for at least one day.
func (s *Service) HasAtLeastOneDay() bool {
	// Quick checks before iterating through each day.
	// quick check that we have at least a week of service,
	// otherwise fall back to full check...
	duration := s.EndDate.Sub(s.StartDate).Hours() / 24
	days := s.Monday + s.Tuesday + s.Wednesday + s.Thursday + s.Friday + s.Saturday + s.Sunday
	if duration >= 7 && days > 0 && len(s.exceptions) == 0 {
		return true
	}
	add, remove := 0, 0
	for _, v := range s.exceptions {
		if v == 1 {
			add++
		} else if v == 2 {
			remove++
		}
	}
	// By definition, at least one day
	if add > 0 {
		return true
	}
	// By definition, no days
	if days == 0 && add == 0 {
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
	inputServiceStart, inputServiceEnd := s.StartDate, s.EndDate
	if s.Generated || s.StartDate.IsZero() || s.EndDate.IsZero() {
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
		etype := s.exceptions[newYMD(start)]
		if etype == 1 {
			addedCount[dow]++
		} else if etype == 2 {
			removedCount[dow]++
		}
		start = start.AddDate(0, 0, 1)
	}

	// Set weekdays based on dow counts
	ret := NewService(Calendar{ServiceID: s.ServiceID, Generated: s.Generated, StartDate: inputServiceStart, EndDate: inputServiceEnd})
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
			ret.AddCalendarDate(CalendarDate{Date: start, ExceptionType: 1})
		} else if !a && b {
			// existing is inactive, new is active
			ret.AddCalendarDate(CalendarDate{Date: start, ExceptionType: 2})
		}
		start = start.AddDate(0, 0, 1)
	}
	return ret, nil
}
