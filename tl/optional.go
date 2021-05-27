package tl

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"
)

type OString struct {
	Valid  bool
	String string
}

func NewOString(v string) OString {
	return OString{Valid: true, String: v}
}

// Value returns nil if empty
func (r OString) Value() (driver.Value, error) {
	if r.Valid {
		return r.String, nil
	}
	return nil, nil
}

// Scan implements sql.Scanner
func (r *OString) Scan(src interface{}) error {
	r.Valid = false
	switch v := src.(type) {
	case nil:
		return nil
	case string:
		r.String = v
	case int:
		r.String = strconv.Itoa(v)
	case int64:
		r.String = strconv.Itoa(int(v))
	default:
		return errors.New("cant convert")
	}
	r.Valid = true
	return nil
}

// UnmarshalJSON implements json.Marshaler interface.
func (r *OString) UnmarshalJSON(v []byte) error {
	err := json.Unmarshal(v, &r.String)
	if err != nil {
		return err
	}
	r.Valid = true
	return nil
}

// MarshalJSON implements the json.marshaler interface.
func (r *OString) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.String)
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (r *OString) UnmarshalGQL(v interface{}) error {
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (r OString) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}

/////////////////////

type OInt struct {
	Valid bool
	Int   int
}

func NewOInt(v int) OInt {
	return OInt{Valid: true, Int: v}
}

// Value returns nil if empty
func (r OInt) Value() (driver.Value, error) {
	if r.Valid {
		return int64(r.Int), nil
	}
	return nil, nil
}

// Scan implements sql.Scanner
func (r *OInt) Scan(src interface{}) error {
	r.Valid = false
	var err error
	switch v := src.(type) {
	case nil:
		return nil
	case string:
		r.Int, err = strconv.Atoi(v)
	case int:
		r.Int = v
	case int64:
		r.Int = int(v)
	case float64:
		r.Int = int(v)
	default:
		err = errors.New("cant convert")
	}
	if err != nil {
		return err
	}
	r.Valid = true
	return nil
}

func (r *OInt) String() string {
	return strconv.Itoa(r.Int)
}

// MarshalJSON implements the json.marshaler interface.
func (r *OInt) UnmarshalJSON(v []byte) error {
	err := json.Unmarshal(v, &r.Int)
	if err != nil {
		return err
	}
	r.Valid = true
	return nil
}

// MarshalJSON implements the json.Marshaler interface
func (r *OInt) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Int)
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (r *OInt) UnmarshalGQL(v interface{}) error {
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (r OInt) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}

/////////////////////

type OFloat struct {
	Valid bool
	Float float64
}

func NewOFloat(v float64) OFloat {
	return OFloat{Valid: true, Float: v}
}

// Value returns nil if empty
func (r OFloat) Value() (driver.Value, error) {
	if r.Valid {
		return r.Float, nil
	}
	return nil, nil
}

// Scan implements sql.Scanner
func (r *OFloat) Scan(src interface{}) error {
	r.Valid = false
	var err error
	switch v := src.(type) {
	case nil:
		return nil
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
	if err != nil {
		return err
	}
	r.Valid = true
	return nil
}

func (r *OFloat) String() string {
	return fmt.Sprintf("%0.5f", r.Float)
}

// MarshalJSON implements the json.marshaler interface.
func (r *OFloat) UnmarshalJSON(v []byte) error {
	err := json.Unmarshal(v, &r.Float)
	if err != nil {
		return err
	}
	r.Valid = true
	return nil
}

// MarshalJSON implements the json.Marshaler interface
func (r *OFloat) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Float)
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (r *OFloat) UnmarshalGQL(v interface{}) error {
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (r OFloat) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}

/////////////////////

// OKey is a nullable foreign key constraint, similar to sql.NullString
type OKey struct {
	Key   string
	Valid bool
}

func NewOKey(v string) OKey {
	return OKey{Valid: true, Key: v}
}

func (r *OKey) String() string {
	return r.Key
}

// Value returns nil if empty
func (r OKey) Value() (driver.Value, error) {
	if !r.Valid || r.Key == "" {
		return nil, nil
	}
	return r.Key, nil
}

// Scan implements sql.Scanner
func (r *OKey) Scan(src interface{}) error {
	r.Valid = false
	switch v := src.(type) {
	case nil:
		return nil
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
		return errors.New("cant convert")
	}
	r.Valid = true
	return nil
}

func (r *OKey) Int() int {
	a, _ := strconv.Atoi(r.Key)
	return a
}

// MarshalJSON implements the json.marshaler interface.
func (r *OKey) UnmarshalJSON(v []byte) error {
	err := json.Unmarshal(v, &r.Key)
	if err != nil {
		return err
	}
	r.Valid = true
	return nil
}

// MarshalJSON implements the json.Marshaler interface
func (r *OKey) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Key)
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (r *OKey) UnmarshalGQL(v interface{}) error {
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (r OKey) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}

/////////////////////

// OTime is a nullable time, but can scan strings
type OTime struct {
	Time  time.Time
	Valid bool
}

// IsZero returns if this is a zero value.
func (r *OTime) IsZero() bool {
	return !r.Valid
}

func (r *OTime) String() string {
	if !r.Valid {
		return ""
	}
	return r.Time.Format("20060102")
}

// Value returns nil if empty
func (r OTime) Value() (driver.Value, error) {
	if !r.Valid {
		return nil, nil
	}
	return driver.Value(r.Time), nil
}

// Scan implements sql.Scanner
func (r *OTime) Scan(src interface{}) error {
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

// MarshalJSON implements the json.Marshaler interface
func (r *OTime) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Time.Format("2006-01-02"))
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (r *OTime) UnmarshalGQL(v interface{}) error {
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (r OTime) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
