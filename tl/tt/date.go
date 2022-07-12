package tt

import (
	"encoding/json"
	"io"
	"time"
)

// Date is a nullable date, without time
type Date struct {
	Option[time.Time]
}

func NewDate(v time.Time) Date {
	return Date{Option[time.Time]{Valid: true, Val: v}}
}

func (r Date) String() string {
	if !r.Valid {
		return ""
	}
	return r.Val.Format("20060102")
}

// UnmarshalJSON needs to use our more flexible dat format
func (r *Date) UnmarshalJSON(v []byte) error {
	r.Valid = false
	if len(v) == 0 {
		return nil
	}
	s := ""
	err := json.Unmarshal(v, &s)
	if err != nil {
		return err
	}
	return r.Scan(s)
}

func (r *Date) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

func (r *Date) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Val.Format("2006-01-02"))
}

func (r Date) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
