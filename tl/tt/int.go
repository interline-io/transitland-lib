package tt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"strconv"
)

// Int is a nullable int
type Int struct {
	Val   int
	Valid bool
}

func NewInt(v int) Int {
	return Int{Valid: true, Val: v}
}

func (r Int) Value() (driver.Value, error) {
	if r.Valid {
		return int64(r.Val), nil
	}
	return nil, nil
}

func (r *Int) Scan(src interface{}) error {
	r.Val, r.Valid = 0, false
	if src == nil {
		return nil
	}
	var err error
	switch v := src.(type) {
	case string:
		r.Val, err = strconv.Atoi(v)
	case int:
		r.Val = v
	case int64:
		r.Val = int(v)
	case float64:
		r.Val = int(v)
	default:
		err = errors.New("cant convert")
	}
	r.Valid = (err == nil)
	return err
}

func (r *Int) String() string {
	return strconv.Itoa(r.Val)
}

func (r *Int) UnmarshalJSON(v []byte) error {
	r.Val, r.Valid = 0, false
	if len(v) == 0 {
		return nil
	}
	err := json.Unmarshal(v, &r.Val)
	r.Valid = (err == nil)
	return err
}

func (r *Int) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Val)
}

func (r *Int) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

func (r Int) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
