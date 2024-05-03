package tt

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Time is a nullable date without time component
type Time struct {
	Val   time.Time
	Valid bool
}

func NewTime(v time.Time) Time {
	return Time{Valid: true, Val: v}
}

// IsZero returns if this is a zero value.
func (r Time) IsZero() bool {
	return !r.Valid
}

func (r Time) String() string {
	if !r.Valid {
		return ""
	}
	return r.Val.Format(time.RFC3339)
}

func (r Time) Value() (driver.Value, error) {
	if !r.Valid {
		return nil, nil
	}
	return r.Val, nil
}

func (r *Time) Scan(src interface{}) error {
	r.Val, r.Valid = time.Time{}, false
	var err error
	switch v := src.(type) {
	case nil:
		return nil
	case string:
		if isEmpty(string(v)) {
			return nil
		}
		r.Val, err = time.Parse(time.RFC3339, v)
	case time.Time:
		r.Val = v
	default:
		err = fmt.Errorf("cant convert %T to Time", src)
	}
	r.Valid = (err == nil)
	return err
}

func (r *Time) UnmarshalJSON(v []byte) error {
	return r.Scan(string(stripQuotes(v)))
}

func (r Time) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return jsonNull(), nil
	}
	return json.Marshal(r.Val.Format(time.RFC3339))
}

func (r *Time) UnmarshalGQL(v interface{}) error {
	return nil
}

func (r Time) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
