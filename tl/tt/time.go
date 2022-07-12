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
	Time  time.Time
	Valid bool
}

func NewTime(v time.Time) Time {
	return Time{Valid: true, Time: v}
}

// IsZero returns if this is a zero value.
func (r *Time) IsZero() bool {
	return !r.Valid
}

func (r *Time) String() string {
	if !r.Valid {
		return ""
	}
	return r.Time.Format(time.RFC3339)
}

// Value returns nil if empty
func (r Time) Value() (driver.Value, error) {
	if !r.Valid {
		return nil, nil
	}
	return driver.Value(r.Time), nil
}

// Scan implements sql.Scanner
func (r *Time) Scan(src interface{}) error {
	r.Time, r.Valid = time.Time{}, false
	if src == nil {
		return nil
	}
	var err error
	switch v := src.(type) {
	case string:
		r.Time, err = time.Parse(time.RFC3339, v)
	case time.Time:
		r.Time = v
	default:
		err = fmt.Errorf("cant convert %T", src)
	}
	r.Valid = (err == nil)
	return err
}

// MarshalJSON implements the json.Marshaler interface
func (r *Time) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Time.Format(time.RFC3339))
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (r *Time) UnmarshalGQL(v interface{}) error {
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (r Time) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
