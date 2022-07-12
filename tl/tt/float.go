package tt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
)

type Float struct {
	Valid bool
	Float float64
}

func NewFloat(v float64) Float {
	return Float{Valid: true, Float: v}
}

// Value returns nil if empty
func (r Float) Value() (driver.Value, error) {
	if r.Valid {
		return r.Float, nil
	}
	return nil, nil
}

// Scan implements sql.Scanner
func (r *Float) Scan(src interface{}) error {
	r.Float, r.Valid = 0.0, false
	if src == nil {
		return nil
	}
	var err error
	switch v := src.(type) {
	case string:
		r.Float, err = strconv.ParseFloat(v, 64)
	case int:
		r.Float = float64(v)
	case int64:
		r.Float = float64(v)
	case float64:
		r.Float = v
	default:
		err = errors.New("cant convert")
	}
	r.Valid = (err == nil)
	return err
}

func (r *Float) String() string {
	if r.Float > -100_000 && r.Float < 100_000 {
		return fmt.Sprintf("%g", r.Float)
	}
	return fmt.Sprintf("%0.5f", r.Float)
}

// UnmarshalJSON implements the json.marshaler interface.
func (r *Float) UnmarshalJSON(v []byte) error {
	r.Float, r.Valid = 0, false
	if len(v) == 0 {
		return nil
	}
	err := json.Unmarshal(v, &r.Float)
	r.Valid = (err == nil)
	return err
}

// MarshalJSON implements the json.Marshaler interface
func (r *Float) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Float)
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (r *Float) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

// MarshalGQL implements the graphql.Marshaler interface
func (r Float) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
