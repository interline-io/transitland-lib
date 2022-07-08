package enum

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

/////////////////////

// Date is a nullable date, but can scan strings
type Date struct {
	Time  time.Time
	Valid bool
}

func NewDate(v time.Time) Date {
	return Date{Valid: true, Time: v}
}

// IsZero returns if this is a zero value.
func (r *Date) IsZero() bool {
	return !r.Valid
}

func (r *Date) String() string {
	if !r.Valid {
		return ""
	}
	return r.Time.Format("20060102")
}

// Value returns nil if empty
func (r Date) Value() (driver.Value, error) {
	if !r.Valid {
		return nil, nil
	}
	return driver.Value(r.Time), nil
}

// Scan implements sql.Scanner
func (r *Date) Scan(src interface{}) error {
	r.Time, r.Valid = time.Time{}, false
	if src == nil {
		return nil
	}
	var err error
	switch v := src.(type) {
	case string:
		r.Time, err = time.Parse("20060102", v)
	case time.Time:
		r.Time = v
	default:
		err = fmt.Errorf("cant convert %T", src)
	}
	r.Valid = (err == nil)
	return err
}

// UnmarshalJSON implements the json.Marshaler interface
func (r *Date) UnmarshalJSON(v []byte) error {
	r.Time, r.Valid = time.Time{}, false
	if len(v) == 0 {
		return nil
	}
	b := ""
	if err := json.Unmarshal(v, &b); err != nil {
		return err
	}
	a, err := time.Parse("2006-01-02", b)
	if err != nil {
		return err
	}
	r.Time, r.Valid = a, true
	return nil
}

// MarshalJSON implements the json.Marshaler interface
func (r *Date) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Time.Format("2006-01-02"))
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (r *Date) UnmarshalGQL(src interface{}) error {
	r.Valid = false
	var p error
	switch v := src.(type) {
	case nil:
		// pass
	case string:
		r.Time, p = time.Parse("2006-01-02", v)
	case time.Time:
		r.Time = v
	default:
		p = fmt.Errorf("cant convert %T", src)
	}
	if p == nil {
		r.Valid = true
	}
	return p
}

// MarshalGQL implements the graphql.Marshaler interface
func (r Date) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
