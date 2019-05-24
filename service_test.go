package gotransit

import (
	"testing"
	"time"
)

func newTestService() *Service {
	start, _ := time.Parse("20060102", "20190101")
	end, _ := time.Parse("20060102", "20190131")
	except, _ := time.Parse("20060102", "20190102")
	added, _ := time.Parse("20060102", "20190105")
	s := Service{
		Calendar: Calendar{
			StartDate: start,
			EndDate:   end,
			Monday:    1,
			Tuesday:   1,
			Wednesday: 1,
			Thursday:  1,
			Friday:    1,
			Saturday:  0,
			Sunday:    0,
		},
		AddedDates:  []time.Time{added},
		ExceptDates: []time.Time{except},
	}
	return &s
}

func TestService_IsActive(t *testing.T) {
	s := newTestService()
	dates := []struct {
		day   string
		value bool
	}{
		{"20190106", false}, // sunday
		{"20190107", true},  // monday
		{"20190108", true},  // tuesday
		{"20190109", true},  // wednesday
		{"20190110", true},  // thursday
		{"20190111", true},  // friday
		{"20190112", false}, // saturday
		{"20190204", false}, // first monday in february
		{"20190102", false}, // except day
		{"20190105", true},  // saturday added
	}
	for _, exp := range dates {
		day, _ := time.Parse("20060102", exp.day)
		result := s.IsActive(day)
		if result != exp.value {
			t.Errorf("day %s got %t expect %t", exp.day, result, exp.value)
		}
	}

}
