package enum

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"strconv"
)

type String struct {
	Valid  bool
	String string
}

func NewString(v string) String {
	return String{Valid: true, String: v}
}

// Value returns nil if empty
func (r String) Value() (driver.Value, error) {
	if r.Valid {
		return r.String, nil
	}
	return nil, nil
}

// Scan implements sql.Scanner
func (r *String) Scan(src interface{}) error {
	r.String, r.Valid = "", false
	if src == nil {
		return nil
	}
	switch v := src.(type) {
	case string:
		r.String = v
	case int:
		r.String = strconv.Itoa(v)
	case int64:
		r.String = strconv.Itoa(int(v))
	default:
		return errors.New("cant convert")
	}
	if r.String != "" {
		r.Valid = true
	}
	return nil
}

// UnmarshalJSON implements json.Marshaler interface.
func (r *String) UnmarshalJSON(v []byte) error {
	r.String, r.Valid = "", false
	if len(v) == 0 {
		return nil
	}
	err := json.Unmarshal(v, &r.String)
	r.Valid = (err == nil && r.String != "")
	return err
}

// MarshalJSON implements the json.marshaler interface.
func (r *String) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.String)
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

//////////

// Strings helps read and write []String as JSON
type Strings []String

func (a Strings) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *Strings) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}
