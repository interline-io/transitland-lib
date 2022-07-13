package tt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"strconv"
)

type String struct {
	Val   string
	Valid bool
}

func NewString(v string) String {
	return String{Valid: true, Val: v}
}

// Value returns nil if empty
func (r String) Value() (driver.Value, error) {
	if r.Valid {
		return r.Val, nil
	}
	return nil, nil
}

// Scan implements sql.Scanner
func (r *String) Scan(src interface{}) error {
	r.Val, r.Valid = "", false
	if src == nil {
		return nil
	}
	switch v := src.(type) {
	case string:
		r.Val = v
	case int:
		r.Val = strconv.Itoa(v)
	case int64:
		r.Val = strconv.Itoa(int(v))
	default:
		return errors.New("cant convert")
	}
	if r.Val != "" {
		r.Valid = true
	}
	return nil
}

// UnmarshalJSON implements json.Marshaler interface.
func (r *String) UnmarshalJSON(v []byte) error {
	r.Val, r.Valid = "", false
	if len(v) == 0 {
		return nil
	}
	err := json.Unmarshal(v, &r.Val)
	r.Valid = (err == nil && r.Val != "")
	return err
}

// MarshalJSON implements the json.marshaler interface.
func (r *String) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Val)
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (r *String) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

// MarshalGQL implements the graphql.Marshaler interface
func (r String) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
