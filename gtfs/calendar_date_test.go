package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestCalendarDate_Errors(t *testing.T) {
	newCalendarDate := func(fn func(*CalendarDate)) *CalendarDate {
		date, _ := tt.ParseDate("20180101")
		calendarDate := &CalendarDate{
			ServiceID:     tt.NewKey("ok"),
			Date:          date,
			ExceptionType: tt.NewInt(2),
		}
		if fn != nil {
			fn(calendarDate)
		}
		return calendarDate
	}

	tests := []struct {
		name           string
		calendarDate   *CalendarDate
		expectedErrors []ExpectError
	}{
		{
			name:           "Valid calendar date (exception_type=2)",
			calendarDate:   newCalendarDate(nil),
			expectedErrors: nil,
		},
		{
			name: "Valid calendar date (exception_type=1)",
			calendarDate: newCalendarDate(func(cd *CalendarDate) {
				date, _ := tt.ParseDate("20180102")
				cd.Date = date
				cd.ExceptionType = tt.NewInt(1)
			}),
			expectedErrors: nil,
		},
		{
			name: "Invalid exception_type (value=3)",
			calendarDate: newCalendarDate(func(cd *CalendarDate) {
				cd.ExceptionType = tt.NewInt(3)
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:exception_type"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.calendarDate)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}
