package gotransit

import (
	"database/sql/driver"
	"errors"
	"strconv"
	"time"
)

// OptionalRelationship is a nullable foreign key constraint, similar to sql.NullString
type OptionalRelationship struct {
	Key   string
	Valid bool
}

func (r *OptionalRelationship) String() string {
	return r.Key
}

// Value returns nil if empty
func (r OptionalRelationship) Value() (driver.Value, error) {
	if r.Key == "" || !r.Valid {
		return nil, nil
	}
	return r.String, nil
}

// Scan implements sql.Scanner
func (r *OptionalRelationship) Scan(src interface{}) error {
	r.Valid = false
	var p error
	switch v := src.(type) {
	case string:
		r.Key = v
	case int:
		r.Key = strconv.Itoa(v)
	default:
		r.Valid = false
		p = errors.New("cant convert")
	}
	if p == nil {
		r.Valid = true
	}
	return p
}

// OptionalTime is a nullable time.
type OptionalTime struct {
	Time  time.Time
	Valid bool
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
