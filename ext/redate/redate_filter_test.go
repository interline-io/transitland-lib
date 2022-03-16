package redate

import (
	"fmt"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/mock"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/stretchr/testify/assert"
)

var tft = "2006-01-02"

func ptime(v string) time.Time {
	t, _ := time.Parse(tft, v)
	return t
}

func TestRedateFilter(t *testing.T) {
	tcs := []struct {
		Name      string
		StartDate string
		EndDate   string
		StartDays int
		EndDays   int
		ExpectSvc tl.Service
	}{
		{"7/7 days", "2018-06-03", "2022-01-02", 7, 7, tl.Service{Calendar: tl.Calendar{ServiceID: "WKDY", StartDate: ptime("2022-01-02"), EndDate: ptime("2022-01-08"), Monday: 1, Tuesday: 1, Wednesday: 1, Thursday: 1, Friday: 1}}},
		{"7/28 days", "2018-06-03", "2022-01-02", 7, 28, tl.Service{Calendar: tl.Calendar{ServiceID: "WKDY", StartDate: ptime("2022-01-02"), EndDate: ptime("2022-01-29"), Monday: 1, Tuesday: 1, Wednesday: 1, Thursday: 1, Friday: 1}}},
		{"7/28 days sunday", "2018-06-03", "2022-01-02", 7, 28, tl.Service{Calendar: tl.Calendar{ServiceID: "SUN", StartDate: ptime("2022-01-02"), EndDate: ptime("2022-01-29"), Sunday: 1}}},
		{"7/28 days monday holiday", "2018-05-27", "2022-01-02", 7, 28, tl.Service{Calendar: tl.Calendar{ServiceID: "WKDY", StartDate: ptime("2022-01-02"), EndDate: ptime("2022-01-29"), Monday: 0, Tuesday: 1, Wednesday: 1, Thursday: 1, Friday: 1}}},
		{"7/365 days", "2018-06-03", "2022-01-02", 7, 365, tl.Service{Calendar: tl.Calendar{ServiceID: "WKDY", StartDate: ptime("2022-01-02"), EndDate: ptime("2023-01-01"), Monday: 1, Tuesday: 1, Wednesday: 1, Thursday: 1, Friday: 1}}},
		{"7/1 days", "2018-06-03", "2022-01-02", 7, 1, tl.Service{Calendar: tl.Calendar{ServiceID: "WKDY", StartDate: ptime("2022-01-02"), EndDate: ptime("2022-01-02"), Monday: 1, Tuesday: 1, Wednesday: 1, Thursday: 1, Friday: 1}}},
		{"1/1 days", "2018-06-03", "2022-01-02", 1, 1, tl.Service{Calendar: tl.Calendar{ServiceID: "WKDY", StartDate: ptime("2022-01-02"), EndDate: ptime("2022-01-02"), Monday: 1, Tuesday: 1, Wednesday: 1, Thursday: 1, Friday: 1}}},
		{"1/7 days monday start", "2018-06-04", "2022-01-03", 1, 7, tl.Service{Calendar: tl.Calendar{ServiceID: "WKDY", StartDate: ptime("2022-01-03"), EndDate: ptime("2022-01-09"), Sunday: 1, Saturday: 1, Monday: 1, Tuesday: 1, Wednesday: 1, Thursday: 1, Friday: 1}}},
	}
	reader, err := tlcsv.NewReader(testutil.ExampleFeedBART.URL)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range tcs {
		name := fmt.Sprintf("%s-%s:%d:%d", tc.StartDate, tc.EndDate, tc.StartDays, tc.EndDays)
		t.Run(name, func(t *testing.T) {
			rf := NewRedateFilter(ptime(tc.StartDate), ptime(tc.EndDate), tc.StartDays, tc.EndDays)
			w := mock.Writer{}
			cp, err := copier.NewCopier(reader, &w, copier.Options{})
			if err != nil {
				t.Fatal(err)
			}
			cp.AddExtension(rf)
			cp.Copy()
			wr, err := w.NewReader()
			if err != nil {
				t.Fatal(err)
			}
			svcs := tl.NewServicesFromReader(wr)
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
				t.Errorf("did not match expected output service %s", v.ServiceID)
			}
		})
	}
}
