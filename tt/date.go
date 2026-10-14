package tt

import (
	"encoding/json"
	"time"
)

// Date is a nullable date
type Date struct {
	Option[time.Time]
}

func ParseDate(s string) (Date, error) {
	d := Date{}
	err := d.Scan(s)
	return d, err
}

func NewDate(v time.Time) Date {
	return Date{Option: NewOption(v)}
}

func (r Date) Before(other Date) bool {
	return r.Val.Before(other.Val)
}

func (r Date) After(other Date) bool {
	return r.Val.After(other.Val)
}

func (r Date) Format(fmt string) string {
	return r.Val.Format(fmt)
}

func (r Date) ToCsv() string {
	if !r.Valid {
		return ""
	}
	return r.Val.Format("20060102")
}

func (r Date) String() string {
	if !r.Valid {
		return ""
	}
	return r.Val.Format("2006-01-02")
}

func (r Date) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return jsonNull(), nil
	}
	return json.Marshal(r.Val.Format("2006-01-02"))
}

// ServiceDayStart is the instant a service date's times count from: noon less
// twelve hours in loc, which differs from midnight on the days clocks change.
func ServiceDayStart(d Date, loc *time.Location) time.Time {
	y, m, day := d.Val.Date()
	return time.Date(y, m, day, 12, 0, 0, 0, loc).Add(-12 * time.Hour)
}
