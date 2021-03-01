package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlcsv"
)

func newTestService() *tl.Service {
	start, _ := time.Parse("20060102", "20190101")
	end, _ := time.Parse("20060102", "20190131")
	except, _ := time.Parse("20060102", "20190102")
	added, _ := time.Parse("20060102", "20190105")
	s := tl.NewService(
		tl.Calendar{
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
		tl.CalendarDate{Date: added, ExceptionType: 1},
		tl.CalendarDate{Date: except, ExceptionType: 2},
	)
	return s
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

func TestService_Simplify(t *testing.T) {
	type testcase struct {
		name    string
		service *tl.Service
	}
	testcases := []testcase{
		{"TestService", newTestService()},
	}
	// get more examples from feeds
	feedchecks := []string{
		"../../test/data/example",
		"../../test/data/external/caltrain.zip",
		"../../test/data/external/bart.zip",
		"../../test/data/external/mbta.zip",
		"../../test/data/external/cdmx.zip",
	}
	for _, path := range feedchecks {
		reader, err := tlcsv.NewReader(path)
		if err != nil {
			panic(err)
		}
		if err := reader.Open(); err != nil {
			panic(err)
		}
		for _, svc := range tl.NewServicesFromReader(reader) {
			testcases = append(testcases, testcase{fmt.Sprintf("%s:%s", path, svc.ServiceID), svc})
		}
	}
	for _, tc := range testcases {
		if len(tc.service.CalendarDates()) == 0 {
			// No need to test services without exceptions...
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			s := tc.service
			ret, err := s.Simplify()
			if err != nil {
				t.Error(err)
				return
			}
			// Verify all IsActive values match
			start, end := s.ServicePeriod()
			for start.Before(end) || start.Equal(end) {
				a := s.IsActive(start)
				b := ret.IsActive(start)
				if a != b {
					t.Errorf("got %t on day %s, expected %t", b, start.Format("2006-01-02"), a)
				}
				start = start.AddDate(0, 0, 1)
			}
		})
	}
}
