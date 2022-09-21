package tt

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Date is a nullable date
type Date struct {
	Val   time.Time
	Valid bool
}

func NewDate(v time.Time) Date {
	return Date{Valid: true, Val: v}
}

func (r *Date) IsZero() bool {
	return !r.Valid
}

func (r *Date) String() string {
	if !r.Valid {
		return ""
	}
	return r.Val.Format("20060102")
}

func (r Date) Value() (driver.Value, error) {
	if !r.Valid {
		return nil, nil
	}
	return driver.Value(r.Val), nil
}

func (r *Date) Scan(src interface{}) error {
	r.Val, r.Valid = time.Time{}, false
	if src == nil {
		return nil
	}
	var err error
	switch v := src.(type) {
	case string:
		r.Val, err = time.Parse("20060102", v)
	case time.Time:
		r.Val = v
	default:
		err = fmt.Errorf("cant convert %T", src)
	}
	r.Valid = (err == nil)
	return err
}

func (r *Date) UnmarshalJSON(v []byte) error {
	r.Val, r.Valid = time.Time{}, false
	if len(v) == 0 {
		return nil
	}
	b := ""
	if err := json.Unmarshal(v, &b); err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}
	a, err := time.Parse("2006-01-02", b)
	if err != nil {
		return err
	}
	r.Val, r.Valid = a, true
	return nil
}

func (r Date) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Val.Format("2006-01-02"))
}

func (r *Date) UnmarshalGQL(src interface{}) error {
	r.Valid = false
	var p error
	switch v := src.(type) {
	case nil:
		// pass
	case string:
		if len(v) == 0 {
			return nil
		}
		r.Val, p = time.Parse("2006-01-02", v)
	case time.Time:
		r.Val = v
	default:
		p = fmt.Errorf("cant convert %T", src)
	}
	if p == nil {
		r.Valid = true
	}
	return p
}

func (r Date) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
