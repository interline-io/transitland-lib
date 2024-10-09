package tt

import (
	"encoding/json"
	"io"
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

func (r Date) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
