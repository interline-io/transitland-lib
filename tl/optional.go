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
func (r *OString) UnmarshalJSON(v []byte) error {
	r.String, r.Valid = "", false
	if len(v) == 0 {
		return nil
	}
	err := json.Unmarshal(v, &r.String)
	r.Valid = (err == nil && r.String != "")
	return err
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
	return r.Scan(v)
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
	case float64:
		r.Int = int(v)
	default:
		err = errors.New("cant convert")
	}
	r.Valid = (err == nil)
	return err
}

func (r *OInt) String() string {
	return strconv.Itoa(r.Int)
}

// UnmarshalJSON implements the json.marshaler interface.
func (r *OInt) UnmarshalJSON(v []byte) error {
	r.Int, r.Valid = 0, false
	if len(v) == 0 {
		return nil
	}
	err := json.Unmarshal(v, &r.Int)
	r.Valid = (err == nil)
	return err
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
	return r.Scan(v)
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

func (r *OFloat) String() string {
	if r.Float > -100_000 && r.Float < 100_000 {
		return fmt.Sprintf("%g", r.Float)
	}
	return fmt.Sprintf("%0.5f", r.Float)
}

// UnmarshalJSON implements the json.marshaler interface.
func (r *OFloat) UnmarshalJSON(v []byte) error {
	r.Float, r.Valid = 0, false
	if len(v) == 0 {
		return nil
	}
	err := json.Unmarshal(v, &r.Float)
	r.Valid = (err == nil)
	return err
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
	return r.Scan(v)
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

func (r *OKey) Int() int {
	a, _ := strconv.Atoi(r.Key)
	return a
}

// UnmarshalJSON implements the json.marshaler interface.
func (r *OKey) UnmarshalJSON(v []byte) error {
	r.Key, r.Valid = "", false
	if len(v) == 0 {
		return nil
	}
	err := json.Unmarshal(v, &r.Key)
	r.Valid = (err == nil)
	return err
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
	return r.Scan(v)
}

// MarshalGQL implements the graphql.Marshaler interface
func (r OKey) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}

/////////////////////

// OTime is a nullable date without time component
type OTime struct {
	Time  time.Time
	Valid bool
}

func NewOTime(v time.Time) OTime {
	return OTime{Valid: true, Time: v}
}

// IsZero returns if this is a zero value.
func (r *OTime) IsZero() bool {
	return !r.Valid
}

func (r *OTime) String() string {
	if !r.Valid {
		return ""
	}
	return r.Time.Format(time.RFC3339)
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
func (r *OTime) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Time.Format(time.RFC3339))
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

/////////////////////

// ODate is a nullable date, but can scan strings
type ODate struct {
	Time  time.Time
	Valid bool
}

func NewODate(v time.Time) ODate {
	return ODate{Valid: true, Time: v}
}

// IsZero returns if this is a zero value.
func (r *ODate) IsZero() bool {
	return !r.Valid
}

func (r *ODate) String() string {
	if !r.Valid {
		return ""
	}
	return r.Time.Format("20060102")
}

// Value returns nil if empty
func (r ODate) Value() (driver.Value, error) {
	if !r.Valid {
		return nil, nil
	}
	return driver.Value(r.Time), nil
}

// Scan implements sql.Scanner
func (r *ODate) Scan(src interface{}) error {
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
func (r *ODate) UnmarshalJSON(v []byte) error {
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
func (r *ODate) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Time.Format("2006-01-02"))
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (r *ODate) UnmarshalGQL(src interface{}) error {
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
func (r ODate) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}

/////////////////////

// Tags is a map[string]string with json and gql marshal support.
// This is a struct instead of bare map[string]string because of a gqlgen issue.
type Tags struct {
	tags map[string]string
}

// Value .
func (r Tags) Value() (driver.Value, error) {
	return json.Marshal(r.tags)
}

// Scan .
func (r *Tags) Scan(value interface{}) error {
	r.tags = nil
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &r.tags)
}

// MarshalJSON implements the json.marshaler interface.
func (r *Tags) MarshalJSON() ([]byte, error) {
	if r.tags == nil {
		return []byte("null"), nil
	}
	return json.Marshal(r.tags)
}

// MarshalGQL implements the graphql.Marshaler interface
func (r Tags) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}

// UnmarshalJSON implements json.Marshaler interface.
func (r *Tags) UnmarshalJSON(v []byte) error {
	r.tags = nil
	if len(v) == 0 {
		return nil
	}
	return json.Unmarshal(v, &r.tags)
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (r *Tags) UnmarshalGQL(v interface{}) error {
	rt := map[string]string{}
	a, ok := v.(map[string]interface{})
	if !ok {
		return errors.New("cannot unmarshal")
	}
	for k, value := range a {
		if c, ok := value.(string); ok {
			rt[k] = c
		} else {
			return errors.New("cannot unmarshal")
		}
	}
	r.tags = rt
	return nil
}

// Keys return the tag keys
func (r *Tags) Keys() []string {
	var ret []string
	for k := range r.tags {
		ret = append(ret, k)
	}
	return ret
}

// Set a tag value
func (r *Tags) Set(k, v string) {
	if r.tags == nil {
		r.tags = map[string]string{}
	}
	r.tags[k] = v
}

// Get a tag value by key
func (r *Tags) Get(k string) (string, bool) {
	if r.tags == nil {
		return "", false
	}
	a, ok := r.tags[k]
	return a, ok
}

/////////////////

// IntSlice .
type IntSlice struct {
	Valid bool
	Ints  []int
}

// Value .
func (a IntSlice) Value() (driver.Value, error) {
	if !a.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(a.Ints)
}

// Scan .
func (a *IntSlice) Scan(value interface{}) error {
	a.Ints, a.Valid = nil, false
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a.Ints)
}
