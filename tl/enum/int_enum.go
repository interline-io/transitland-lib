package enum

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"strconv"
)

type IntEnum struct {
	Valid bool
	Int   int
}

func NewIntEnum(v int) IntEnum {
	return IntEnum{Valid: true, Int: v}
}

// Value returns nil if empty
func (r IntEnum) Value() (driver.Value, error) {
	if r.Valid {
		return int64(r.Int), nil
	}
	return nil, nil
}

// Scan implements sql.Scanner
func (r *IntEnum) Scan(src interface{}) error {
	r.Int, r.Valid = 0, false
	if src == nil {
		return nil
	}
	var err error
	switch v := src.(type) {
	case string:
		r.Int, err = strconv.Atoi(v)
	case int:
		r.Int = v
	case int64:
		r.Int = int(v)
	default:
		err = errors.New("cant convert")
	}
	r.Valid = (err == nil)
	return err
}

func (r *IntEnum) String() string {
	if !r.Valid {
		return ""
	}
	return strconv.Itoa(r.Int)
}

func (r *IntEnum) IsValid() bool {
	return r.Valid
}

func (r *IntEnum) Error() error {
	return nil
}

// UnmarshalJSON implements the json.marshaler interface.
func (r *IntEnum) UnmarshalJSON(v []byte) error {
	r.Int, r.Valid = 0, false
	if len(v) == 0 {
		return nil
	}
	err := json.Unmarshal(v, &r.Int)
	r.Valid = (err == nil)
	return err
}

// MarshalJSON implements the json.Marshaler interface
func (r *IntEnum) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Int)
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (r *IntEnum) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

// MarshalGQL implements the graphql.Marshaler interface
func (r IntEnum) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
