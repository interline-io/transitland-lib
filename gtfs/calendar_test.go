package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestCalendar_Errors(t *testing.T) {
	newCalendar := func(fn func(*Calendar)) *Calendar {
		startDate, _ := tt.ParseDate("20100101")
		endDate, _ := tt.ParseDate("21001231")
		calendar := &Calendar{
			ServiceID: tt.NewString("ok"),
			Monday:    tt.NewInt(1),
			Tuesday:   tt.NewInt(1),
			Wednesday: tt.NewInt(1),
			Thursday:  tt.NewInt(1),
			Friday:    tt.NewInt(1),
			Saturday:  tt.NewInt(1),
			Sunday:    tt.NewInt(1),
			StartDate: startDate,
			EndDate:   endDate,
		}
		if fn != nil {
			fn(calendar)
		}
		return calendar
	}

	tests := []struct {
		name           string
		calendar       *Calendar
		expectedErrors []ExpectError
	}{
		{
			name:           "Valid calendar",
			calendar:       newCalendar(nil),
			expectedErrors: nil,
		},
		{
			name: "Valid weekend service",
			calendar: newCalendar(func(c *Calendar) {
				c.ServiceID = tt.NewString("weekend")
				c.Monday = tt.NewInt(0)
				c.Tuesday = tt.NewInt(0)
				c.Wednesday = tt.NewInt(0)
				c.Thursday = tt.NewInt(0)
				c.Friday = tt.NewInt(0)
			}),
			expectedErrors: nil,
		},
		{
			name: "Missing monday (required field)",
			calendar: newCalendar(func(c *Calendar) {
				c.Monday = tt.Int{}
			}),
			expectedErrors: PE("RequiredFieldError:monday"),
		},
		{
			name: "Missing tuesday (required field)",
			calendar: newCalendar(func(c *Calendar) {
				c.Tuesday = tt.Int{}
			}),
			expectedErrors: PE("RequiredFieldError:tuesday"),
		},
		{
			name: "Missing wednesday (required field)",
			calendar: newCalendar(func(c *Calendar) {
				c.Wednesday = tt.Int{}
			}),
			expectedErrors: PE("RequiredFieldError:wednesday"),
		},
		{
			name: "Missing thursday (required field)",
			calendar: newCalendar(func(c *Calendar) {
				c.Thursday = tt.Int{}
			}),
			expectedErrors: PE("RequiredFieldError:thursday"),
		},
		{
			name: "Missing friday (required field)",
			calendar: newCalendar(func(c *Calendar) {
				c.Friday = tt.Int{}
			}),
			expectedErrors: PE("RequiredFieldError:friday"),
		},
		{
			name: "Missing saturday (required field)",
			calendar: newCalendar(func(c *Calendar) {
				c.Saturday = tt.Int{}
			}),
			expectedErrors: PE("RequiredFieldError:saturday"),
		},
		{
			name: "Missing sunday (required field)",
			calendar: newCalendar(func(c *Calendar) {
				c.Sunday = tt.Int{}
			}),
			expectedErrors: PE("RequiredFieldError:sunday"),
		},
		{
			name: "Invalid monday (value > 1)",
			calendar: newCalendar(func(c *Calendar) {
				c.Monday = tt.NewInt(2)
			}),
			expectedErrors: PE("InvalidFieldError:monday"),
		},
		{
			name: "Invalid tuesday (value > 1)",
			calendar: newCalendar(func(c *Calendar) {
				c.Tuesday = tt.NewInt(2)
			}),
			expectedErrors: PE("InvalidFieldError:tuesday"),
		},
		{
			name: "Invalid wednesday (value > 1)",
			calendar: newCalendar(func(c *Calendar) {
				c.Wednesday = tt.NewInt(2)
			}),
			expectedErrors: PE("InvalidFieldError:wednesday"),
		},
		{
			name: "Invalid thursday (value > 1)",
			calendar: newCalendar(func(c *Calendar) {
				c.Thursday = tt.NewInt(2)
			}),
			expectedErrors: PE("InvalidFieldError:thursday"),
		},
		{
			name: "Invalid friday (value > 1)",
			calendar: newCalendar(func(c *Calendar) {
				c.Friday = tt.NewInt(2)
			}),
			expectedErrors: PE("InvalidFieldError:friday"),
		},
		{
			name: "Invalid saturday (value > 1)",
			calendar: newCalendar(func(c *Calendar) {
				c.Saturday = tt.NewInt(2)
			}),
			expectedErrors: PE("InvalidFieldError:saturday"),
		},
		{
			name: "Invalid sunday (value > 1)",
			calendar: newCalendar(func(c *Calendar) {
				c.Sunday = tt.NewInt(2)
			}),
			expectedErrors: PE("InvalidFieldError:sunday"),
		},
		{
			name: "Start date after end date",
			calendar: newCalendar(func(c *Calendar) {
				startDate, _ := tt.ParseDate("20100101")
				endDate, _ := tt.ParseDate("20010101")
				c.StartDate = startDate
				c.EndDate = endDate
			}),
			expectedErrors: PE("InvalidFieldError:end_date"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.calendar)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}
