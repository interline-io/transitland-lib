package tl

import (
	"errors"
	"fmt"
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

// CalendarDates returns CalendarDates for this service.
func (s *Service) CalendarDates() []CalendarDate {
	ret := []CalendarDate{}
	for ymd, v := range s.exceptions {
		ret = append(ret, CalendarDate{
			ServiceID:     s.ServiceID,
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

func (s *Service) Exception(t time.Time) (int, bool) {
	a, ok := s.exceptions[newYMD(t)]
	return a, ok
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
	for dow, total := range totalCount {
		if total == 0 {
			continue
		}
		active := activeCount[dow]
		willBeAdded := active           // if 0, then add
		willBeRemoved := total - active // if 1, then remove
		// r := float64(active) / float64(total)
		// _ = r
		// added := addedCount[dow]
		// removed := removedCount[dow]
		// fmt.Println("dow:", dow, "total:", total, "active:", active, "added:", added, "removed:", removed, "willBeAdded:", willBeAdded, "willBeRemoved:", willBeRemoved)
		if total == active || willBeAdded >= willBeRemoved {
			// fmt.Println("setting active:", dow)
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
			// fmt.Println("adding:", start, "dow:", start.Weekday())
			ret.AddCalendarDate(CalendarDate{Date: start, ExceptionType: 1})
		} else if !a && b {
			// existing is inactive, new is active
			// fmt.Println("removing:", start, "dow:", start.Weekday())
			ret.AddCalendarDate(CalendarDate{Date: start, ExceptionType: 2})
		}
		start = start.AddDate(0, 0, 1)
	}
	fmt.Println("input:", s.StartDate.String()[0:10], "end:", s.EndDate.String()[0:10], "Days:", s.Sunday, s.Monday, s.Tuesday, s.Wednesday, s.Thursday, s.Friday, s.Saturday, "calendar_date count:", len(s.CalendarDates()))
	fmt.Println("ret  :", ret.StartDate.String()[0:10], "end:", ret.EndDate.String()[0:10], "Days:", ret.Sunday, ret.Monday, ret.Tuesday, ret.Wednesday, ret.Thursday, ret.Friday, ret.Saturday, "calendar_date count:", len(ret.CalendarDates()))
	if a, b := len(s.CalendarDates()), len(ret.CalendarDates()); b > a {
		fmt.Printf("calendar_dates increased: %d -> %d\n", a, b)
	} else if b < a {
		fmt.Printf("ok; calendar_dates decreased: %d -> %d\n", a, b)
	}

	return ret, nil
}
