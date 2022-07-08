package tltypes

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"strconv"
)

// Key is a nullable foreign key constraint, similar to sql.NullString
type Key struct {
	Key   string
	Valid bool
}

func NewKey(v string) Key {
	return Key{Valid: true, Key: v}
}

func (r *Key) String() string {
	return r.Key
}

// Value returns nil if empty
func (r Key) Value() (driver.Value, error) {
	if !r.Valid || r.Key == "" {
		return nil, nil
	}
	return r.Key, nil
}

// Scan implements sql.Scanner
func (r *Key) Scan(src interface{}) error {
	r.Key, r.Valid = "", false
	if src == nil {
		return nil
	}
	var err error
	switch v := src.(type) {
	case string:
		if v == "" {
			return nil
		}
		r.Key = v
	case int:
		r.Key = strconv.Itoa(v)
	case int64:
		r.Key = strconv.Itoa(int(v))
	default:
		err = errors.New("cant convert")
	}
	r.Valid = (err == nil && r.Key != "")
	return err
}

func (r *Key) Int() int {
	a, _ := strconv.Atoi(r.Key)
	return a
}

// UnmarshalJSON implements the json.marshaler interface.
func (r *Key) UnmarshalJSON(v []byte) error {
	r.Key, r.Valid = "", false
	if len(v) == 0 {
		return nil
	}
	err := json.Unmarshal(v, &r.Key)
	r.Valid = (err == nil)
	return err
}

// MarshalJSON implements the json.Marshaler interface
func (r *Key) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Key)
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (r *Key) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

// MarshalGQL implements the graphql.Marshaler interface
func (r Key) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
