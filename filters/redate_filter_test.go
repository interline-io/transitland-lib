package filters

import (
	"fmt"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/adapters/tlcsv"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tlutil"
	"github.com/stretchr/testify/assert"
)

var tft = "2006-01-02"

func ptime(v string) time.Time {
	t, _ := time.Parse(tft, v)
	return t
}

func TestRedateFilter(t *testing.T) {
	tcs := []struct {
		Name        string
		StartDate   string
		EndDate     string
		StartDays   int
		EndDays     int
		ExpectError bool
		ExpectSvc   tlutil.Service
	}{
		{"7/7 days", "2018-06-03", "2022-01-02", 7, 7, false, tlutil.Service{Calendar: tl.Calendar{ServiceID: "WKDY", StartDate: ptime("2022-01-02"), EndDate: ptime("2022-01-08"), Monday: 1, Tuesday: 1, Wednesday: 1, Thursday: 1, Friday: 1}}},
		{"7/28 days", "2018-06-03", "2022-01-02", 7, 28, false, tlutil.Service{Calendar: tl.Calendar{ServiceID: "WKDY", StartDate: ptime("2022-01-02"), EndDate: ptime("2022-01-29"), Monday: 1, Tuesday: 1, Wednesday: 1, Thursday: 1, Friday: 1}}},
		{"7/28 days sunday", "2018-06-03", "2022-01-02", 7, 28, false, tlutil.Service{Calendar: tl.Calendar{ServiceID: "SUN", StartDate: ptime("2022-01-02"), EndDate: ptime("2022-01-29"), Sunday: 1}}},
		{"7/28 days monday holiday", "2018-05-27", "2022-01-02", 7, 28, false, tlutil.Service{Calendar: tl.Calendar{ServiceID: "WKDY", StartDate: ptime("2022-01-02"), EndDate: ptime("2022-01-29"), Monday: 0, Tuesday: 1, Wednesday: 1, Thursday: 1, Friday: 1}}},
		{"7/365 days", "2018-06-03", "2022-01-02", 7, 365, false, tlutil.Service{Calendar: tl.Calendar{ServiceID: "WKDY", StartDate: ptime("2022-01-02"), EndDate: ptime("2023-01-01"), Monday: 1, Tuesday: 1, Wednesday: 1, Thursday: 1, Friday: 1}}},
		{"7/365 days monday holiday", "2018-05-27", "2022-01-02", 7, 365, false, tlutil.Service{Calendar: tl.Calendar{ServiceID: "WKDY", StartDate: ptime("2022-01-02"), EndDate: ptime("2023-01-01"), Monday: 0, Tuesday: 1, Wednesday: 1, Thursday: 1, Friday: 1}}},
		{"7/1 days monday start", "2018-06-04", "2022-01-03", 7, 1, false, tlutil.Service{Calendar: tl.Calendar{ServiceID: "WKDY", StartDate: ptime("2022-01-03"), EndDate: ptime("2022-01-03"), Monday: 1}}},
		{"1/1 days monday start", "2018-06-04", "2022-01-03", 1, 1, false, tlutil.Service{Calendar: tl.Calendar{ServiceID: "WKDY", StartDate: ptime("2022-01-03"), EndDate: ptime("2022-01-03"), Monday: 1}}},
		{"1/7 days monday start", "2018-06-04", "2022-01-03", 1, 7, false, tlutil.Service{Calendar: tl.Calendar{ServiceID: "WKDY", StartDate: ptime("2022-01-03"), EndDate: ptime("2022-01-09"), Sunday: 1, Saturday: 1, Monday: 1, Tuesday: 1, Wednesday: 1, Thursday: 1, Friday: 1}}},
		{"no start date", "", "2022-01-02", 7, 7, true, tlutil.Service{}},
		{"no end date", "2018-06-04", "", 7, 7, true, tlutil.Service{}},
		{"no source days", "2018-06-04", "2022-01-03", 0, 7, true, tlutil.Service{}},
		{"no target days", "2018-06-04", "2022-01-03", 7, 0, true, tlutil.Service{}},
		{"different weekday", "2018-06-04", "2022-01-04", 7, 7, true, tlutil.Service{}},
	}
	reader, err := tlcsv.NewReader(testutil.ExampleFeedBART.URL)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			rf, err := NewRedateFilter(ptime(tc.StartDate), ptime(tc.EndDate), tc.StartDays, tc.EndDays, false)
			if err != nil && tc.ExpectError {
				// ok
				return
			} else if err != nil && !tc.ExpectError {
				t.Fatalf("got unexpected error: %s", err.Error())
			} else if err == nil && tc.ExpectError {
				t.Fatalf("expected error, got none")
			}
			// rf.AllowInactive = true
			w := direct.NewWriter()
			cp, err := testutil.NewDirectCopier(reader, w, testutil.DirectCopierOptions{})
			if err != nil {
				t.Fatal(err)
			}
			cp.AddFilter(rf)
			if err := cp.Copy(); err != nil {
				t.Fatal(err)
			}
			wr, err := w.NewReader()
			if err != nil {
				t.Fatal(err)
			}
			svcs := tlutil.NewServicesFromReader(wr)
			found := false
			v := tc.ExpectSvc
			for _, svc := range svcs {
				if v.ServiceID != svc.ServiceID {
					continue
				}
				found = true
				assert.Equal(t, v.StartDate.Format(tft), svc.StartDate.Format(tft))
				assert.Equal(t, v.EndDate.Format(tft), svc.EndDate.Format(tft))
				startDate := v.StartDate
				for startDate.Before(v.EndDate) {
					a := v.IsActive(startDate)
					b := svc.IsActive(startDate)
					assert.Equalf(t, a, b, "expected active %t got %t on date %s", a, b, startDate.Format(tft))
					startDate = startDate.AddDate(0, 0, 1)
				}
			}
			if !found {
				t.Errorf("did not find expected output service %s", v.ServiceID)
			}
		})
	}
}

func Test_daysBetween(t *testing.T) {
	tc := []struct {
		d1     string
		d2     string
		expect int
	}{
		{
			d1:     "2023-05-15",
			d2:     "2023-05-15",
			expect: 0,
		},
		{
			d1:     "2023-05-15",
			d2:     "2023-05-16",
			expect: 1,
		},
		{
			d1:     "2023-05-15",
			d2:     "2024-05-15",
			expect: 366,
		},
		{
			d1:     "2023-05-15",
			d2:     "2030-05-15",
			expect: 2557,
		},
		{
			d1:     "2023-05-15",
			d2:     "2023-05-14",
			expect: -1,
		},
		{
			d1:     "2023-05-15",
			d2:     "2023-05-01",
			expect: -14,
		},
		{
			d1:     "1970-01-01",
			d2:     "2023-05-15",
			expect: 19492,
		},
	}
	for _, tc := range tc {
		t.Run(fmt.Sprintf("%s:%s", tc.d1, tc.d2), func(t *testing.T) {
			t1, err := time.Parse("2006-01-02", tc.d1)
			if err != nil {
				t.Fatal(err)
			}
			t2, err := time.Parse("2006-01-02", tc.d2)
			if err != nil {
				t.Fatal(err)
			}
			days := daysBetween(t1, t2)
			assert.Equal(t, tc.expect, days, "days between")
		})
	}
}
