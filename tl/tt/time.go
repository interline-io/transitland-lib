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
func (r *Time) IsZero() bool {
	return !r.Valid
}

func (r *Time) String() string {
	if !r.Valid {
		return ""
	}
	return r.Val.Format(time.RFC3339)
}

func (r Time) Value() (driver.Value, error) {
	if !r.Valid {
		return nil, nil
	}
	return driver.Value(r.Val), nil
}

func (r *Time) Scan(src interface{}) error {
	r.Val, r.Valid = time.Time{}, false
	if src == nil {
		return nil
	}
	var err error
	switch v := src.(type) {
	case string:
		r.Val, err = time.Parse(time.RFC3339, v)
	case time.Time:
		r.Val = v
	default:
		err = fmt.Errorf("cant convert %T", src)
	}
	r.Valid = (err == nil)
	return err
}

func (r *Time) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
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
