package gotransit

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// The only nullable types in the database are foreign-key constraints and some times.

// OptionalRelationship is a nullable foreign key constraint, similar to sql.NullString
type OptionalRelationship struct {
	Key   string `abc:"test"`
	Valid bool
}

// IsZero returns if this is a zero value.
func (r *OptionalRelationship) IsZero() bool {
	return r.Key == ""
}

func (r *OptionalRelationship) String() string {
	return r.Key
}

// Value returns nil if empty
func (r OptionalRelationship) Value() (driver.Value, error) {
	if r.IsZero() {
		return nil, nil
	}
	return r.Key, nil
}

// Scan implements sql.Scanner
func (r *OptionalRelationship) Scan(src interface{}) error {
	r.Valid = false
	switch v := src.(type) {
	case string:
		r.Key = v
	case int:
		r.Key = strconv.Itoa(v)
	case int64:
		r.Key = strconv.Itoa(int(v))
	case nil:
		r.Valid = false
		return nil
	default:
		fmt.Printf("src: %T %#v\n", src, src)
		return errors.New("cant convert")
	}
	r.Valid = true
	return nil
}

// OptionalTime is a nullable time.
type OptionalTime struct {
	Time  time.Time
	Valid bool
}

// IsZero returns if this is a zero value.
func (r *OptionalTime) IsZero() bool {
	return r.Time.IsZero()
}

func (r *OptionalTime) String() string {
	return r.Time.Format("20060102")
}

// Value returns nil if empty
func (r OptionalTime) Value() (driver.Value, error) {
	if r.Time.IsZero() || !r.Valid {
		return nil, nil
	}
	return driver.Value(r.Time), nil
}

// Scan implements sql.Scanner
func (r *OptionalTime) Scan(src interface{}) error {
	r.Valid = false
	var p error
	switch v := src.(type) {
	case string:
		if t, err := time.Parse("20060102", v); err == nil {
			r.Time = t
		}
	case time.Time:
		r.Time = v
	default:
		p = errors.New("cant convert")
	}
	if p == nil {
		r.Valid = true
	}
	return p
}
