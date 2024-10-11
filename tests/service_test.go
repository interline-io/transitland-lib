package tests

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tlcsv"
)

func newTestService() *service.Service {
	start, _ := time.Parse("20060102", "20190101")
	end, _ := time.Parse("20060102", "20190131")
	except, _ := time.Parse("20060102", "20190102")
	added, _ := time.Parse("20060102", "20190105")
	s := service.NewService(
		gtfs.Calendar{
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
		gtfs.CalendarDate{Date: added, ExceptionType: 1},
		gtfs.CalendarDate{Date: except, ExceptionType: 2},
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
		service *service.Service
	}
	testcases := []testcase{
		{"TestService", newTestService()},
	}
	// get more examples from feeds
	feedchecks := []string{}
	for _, v := range testutil.ExternalTestFeeds {
		feedchecks = append(feedchecks, v.URL)
	}
	for _, path := range feedchecks {
		reader, err := tlcsv.NewReader(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := reader.Open(); err != nil {
			t.Fatal(err)
		}
		for _, svc := range service.NewServicesFromReader(reader) {
			testcases = append(testcases, testcase{fmt.Sprintf("%s:%s", filepath.Base(path), svc.ServiceID), svc})
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
			if a, b := len(s.CalendarDates()), len(ret.CalendarDates()); b > a {
				t.Errorf("got %d calendar dates for simplified service, which is more than the input of %d calendar dates", b, a)
			}
			// Verify all IsActive values match through entire service period, not just StartDate, EndDate
			// debugging....
			// t.Log("input:", s.StartDate.String()[0:10], "end:", s.EndDate.String()[0:10], "Days:", s.Sunday, s.Monday, s.Tuesday, s.Wednesday, s.Thursday, s.Friday, s.Saturday, "calendar_date count:", len(s.CalendarDates()))
			// for _, cd := range s.CalendarDates() {
			// 	t.Logf("\tdate: %s dow: %d type: %d\n", cd.Date.Format("2006-01-02"), cd.Date.Weekday(), cd.ExceptionType)
			// }
			// t.Log("ret  :", ret.StartDate.String()[0:10], "end:", ret.EndDate.String()[0:10], "Days:", ret.Sunday, ret.Monday, ret.Tuesday, ret.Wednesday, ret.Thursday, ret.Friday, ret.Saturday, "calendar_date count:", len(ret.CalendarDates()))
			// for _, cd := range ret.CalendarDates() {
			// 	t.Logf("\tdate: %s dow: %d type: %d\n", cd.Date.Format("2006-01-02"), cd.Date.Weekday(), cd.ExceptionType)
			// }
			// if a, b := len(s.CalendarDates()), len(ret.CalendarDates()); b > a {
			// 	t.Log("calendar_dates increased: %d -> %d\n", a, b)
			// } else if b < a {
			// 	t.Log("ok; calendar_dates decreased: %d -> %d\n", a, b)
			// }
			start, end := s.ServicePeriod()
			for start.Before(end) || start.Equal(end) {
				a := s.IsActive(start)
				b := ret.IsActive(start)
				// t.Logf("\tchecking %s dow: %d a: %t b: %t\n", start.Format("2006-01-02"), start.Weekday(), a, b)
				if a != b {
					t.Errorf("got %t on day %s, expected %t", b, start.Format("2006-01-02"), a)
				}
				start = start.AddDate(0, 0, 1)
			}

		})
	}
}
