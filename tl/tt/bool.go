package tt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
)

type Bool struct {
	Val   bool
	Valid bool
}

func NewBool(v bool) Bool {
	return Bool{Valid: true, Val: v}
}

func (r *Bool) String() string {
	return ""
}

func (r Bool) Value() (driver.Value, error) {
	if !r.Valid {
		return nil, nil
	}
	return r.Val, nil
}

func (r *Bool) Scan(src interface{}) error {
	r.Val, r.Valid = false, false
	var err error
	switch v := src.(type) {
	case nil:
		return nil
	case string:
		if isEmpty(v) {
			return nil
		}
		if v == "true" || v == "1" {
			r.Val = true
		} else if v == "false" || v == "0" {
			r.Val = false
		}
	case int:
		if v == 0 {
			r.Val = false
		} else if v == 1 {
			r.Val = true
		}
	case int64:
		if v == 0 {
			r.Val = false
		} else if v == 1 {
			r.Val = true
		}
	case bool:
		r.Val = v
	default:
		err = errors.New("cant convert")
	}
	r.Valid = (err == nil)
	return err
}

func (r *Bool) UnmarshalJSON(v []byte) error {
	return r.Scan(string(stripQuotes(v)))
}

func (r *Bool) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return jsonNull(), nil
	}
	return json.Marshal(r.Val)
}

func (r *Bool) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

func (r Bool) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
