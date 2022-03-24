package tests

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/stretchr/testify/assert"
)

func tparse(v string) time.Time {
	a, err := time.Parse("20060102", v)
	if err != nil {
		panic(err)
	}
	return a
}

func newTestService() *tl.Service {
	s := tl.NewService(
		tl.Calendar{
			StartDate: tparse("20190101"),
			EndDate:   tparse("20190131"),
			Monday:    1,
			Tuesday:   1,
			Wednesday: 1,
			Thursday:  1,
			Friday:    1,
			Saturday:  0,
			Sunday:    0,
		},
		tl.CalendarDate{Date: tparse("20190102"), ExceptionType: 1},
		tl.CalendarDate{Date: tparse("20190105"), ExceptionType: 2},
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
		result := s.IsActive(tparse(exp.day))
		if result != exp.value {
			t.Errorf("day %s got %t expect %t", exp.day, result, exp.value)
		}
	}
}

func TestService_Equal(t *testing.T) {
	type testcase struct {
		name   string
		a      *tl.Service
		b      *tl.Service
		expect bool
	}
	var tcs []testcase
	s := newTestService()
	tcs = append(tcs, testcase{"basic", s, s, true})
	//
	s2 := newTestService()
	s2.StartDate = s2.StartDate.AddDate(0, 0, 1)
	tcs = append(tcs, testcase{"start date diff", s, s2, false})
	//
	s3 := newTestService()
	s3.EndDate = s2.EndDate.AddDate(0, 0, 1)
	tcs = append(tcs, testcase{"end date diff", s, s3, false})
	//
	s4 := newTestService()
	s4.Monday = 0
	tcs = append(tcs, testcase{"dow diff", s, s4, false})
	//
	s5 := newTestService()
	s5.AddCalendarDate(tl.CalendarDate{ExceptionType: 2, Date: s5.StartDate})
	tcs = append(tcs, testcase{"removed diff", s, s5, false})
	//
	s6 := newTestService()
	s6.AddCalendarDate(tl.CalendarDate{ExceptionType: 2, Date: s5.StartDate.AddDate(0, 0, -1)})
	tcs = append(tcs, testcase{"removed diff outside window", s, s6, false})
	//
	s7 := newTestService()
	s7.Reset()
	tcs = append(tcs, testcase{"after reset", s, s7, false})
	// this test expects equal
	s8 := newTestService()
	s8.AddCalendarDate(tl.CalendarDate{ExceptionType: 1, Date: tparse("20190111")})
	tcs = append(tcs, testcase{"added equal inside window", s, s8, true})
	//
	s9 := newTestService()
	s9.AddCalendarDate(tl.CalendarDate{ExceptionType: 1, Date: s8.StartDate.AddDate(0, 0, -1)})
	tcs = append(tcs, testcase{"added diff outside window", s, s9, false})
	// run
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, tc.a.Equal(tc.b))
		})
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
		for _, svc := range tl.NewServicesFromReader(reader) {
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
			// fmt.Println("input:", s.StartDate.String()[0:10], "end:", s.EndDate.String()[0:10], "Days:", s.Sunday, s.Monday, s.Tuesday, s.Wednesday, s.Thursday, s.Friday, s.Saturday, "calendar_date count:", len(s.CalendarDates()))
			// for _, cd := range s.CalendarDates() {
			// 	fmt.Printf("\tdate: %s dow: %d type: %d\n", cd.Date.Format("2006-01-02"), cd.Date.Weekday(), cd.ExceptionType)
			// }
			// fmt.Println("ret  :", ret.StartDate.String()[0:10], "end:", ret.EndDate.String()[0:10], "Days:", ret.Sunday, ret.Monday, ret.Tuesday, ret.Wednesday, ret.Thursday, ret.Friday, ret.Saturday, "calendar_date count:", len(ret.CalendarDates()))
			// for _, cd := range ret.CalendarDates() {
			// 	fmt.Printf("\tdate: %s dow: %d type: %d\n", cd.Date.Format("2006-01-02"), cd.Date.Weekday(), cd.ExceptionType)
			// }
			// if a, b := len(s.CalendarDates()), len(ret.CalendarDates()); b > a {
			// 	fmt.Printf("calendar_dates increased: %d -> %d\n", a, b)
			// } else if b < a {
			// 	fmt.Printf("ok; calendar_dates decreased: %d -> %d\n", a, b)
			// }
			start, end := s.ServicePeriod()
			for start.Before(end) || start.Equal(end) {
				a := s.IsActive(start)
				b := ret.IsActive(start)
				// fmt.Printf("\tchecking %s dow: %d a: %t b: %t\n", start.Format("2006-01-02"), start.Weekday(), a, b)
				if a != b {
					t.Errorf("got %t on day %s, expected %t", b, start.Format("2006-01-02"), a)
				}
				start = start.AddDate(0, 0, 1)
			}

		})
	}
}
