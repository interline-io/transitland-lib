package tl

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"
)

// The only nullable types in the database are foreign-key constraints and some times.

// OptionalRelationship is a nullable foreign key constraint, similar to sql.NullString
type OptionalRelationship struct {
	Key   string
	Valid bool
}

// IsZero returns if this is a zero value.
func (r *OptionalRelationship) IsZero() bool {
	return r.Key == ""
}

func (r *OptionalRelationship) String() string {
	return r.Key
}

// Int try to convert key to int
func (r *OptionalRelationship) Int() int {
	if !r.Valid {
		return 0
	}
	if v, err := strconv.Atoi(r.Key); err == nil {
		return v
	}
	return 0
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
	case nil:
		return nil
	case string:
		r.Key = v
	case int:
		r.Key = strconv.Itoa(v)
	case int64:
		r.Key = strconv.Itoa(int(v))
	default:
		return errors.New("cant convert")
	}
	r.Valid = true
	return nil
}

// OptionalKey is the same as sql.NullInt
type OptionalKey struct {
	sql.NullInt64
}

// OptionalTime is a nullable time, but can scan strings
type OptionalTime struct {
	Time  time.Time
	Valid bool
}

// IsZero returns if this is a zero value.
func (r *OptionalTime) IsZero() bool {
	return !r.Valid
}

func (r *OptionalTime) String() string {
	if !r.Valid {
		return ""
	}
	return r.Time.Format("20060102")
}

// Value returns nil if empty
func (r OptionalTime) Value() (driver.Value, error) {
	if !r.Valid {
		return nil, nil
	}
	return driver.Value(r.Time), nil
}

// Scan implements sql.Scanner
func (r *OptionalTime) Scan(src interface{}) error {
	r.Valid = false
	var p error
	switch v := src.(type) {
	case nil:
		// pass
	case string:
		r.Time, p = time.Parse("20060102", v)
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

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (r *OptionalTime) UnmarshalGQL(v interface{}) error {
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (r OptionalTime) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}

// MarshalJSON implements the json.Marshaler interface
func (r *OptionalTime) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return []byte("\"" + r.Time.Format("2006-01-02") + "\""), nil
}
