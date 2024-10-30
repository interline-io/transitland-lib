package gtfs

import (
	"github.com/interline-io/transitland-lib/tt"
)

// CalendarDate calendar_dates.txt
type CalendarDate struct {
	ServiceID     tt.Key  `csv:",required" target:"calendar.txt"`
	Date          tt.Date `csv:",required"`
	ExceptionType tt.Int  `csv:",required" enum:"1,2"`
	tt.BaseEntity
}

// Filename calendar_dates.txt
func (ent *CalendarDate) Filename() string {
	return "calendar_dates.txt"
}

// TableName gtfs_calendar_dates
func (ent *CalendarDate) TableName() string {
	return "gtfs_calendar_dates"
}
