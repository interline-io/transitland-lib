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
	return r.Val, nil
}

func (r *Date) Scan(src interface{}) error {
	r.Val, r.Valid = time.Time{}, false
	var err error
	switch v := src.(type) {
	case nil:
		return nil
	case string:
		if isEmpty(v) {
			return nil
		}
		r.Val, err = time.Parse("20060102", v)
		if err != nil {
			v2, err2 := time.Parse("2006-01-02", v)
			if err2 == nil {
				r.Val = v2
				err = nil
			}
		}
	case time.Time:
		r.Val = v
	default:
		err = fmt.Errorf("cant convert %T to Date", src)
	}
	r.Valid = (err == nil)
	return err
}

func (r *Date) UnmarshalJSON(v []byte) error {
	return r.Scan(string(stripQuotes(v)))
}

func (r Date) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return jsonNull(), nil
	}
	return json.Marshal(r.Val.Format("2006-01-02"))
}

func (r *Date) UnmarshalGQL(src interface{}) error {
	return r.Scan(src)
}

func (r Date) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
