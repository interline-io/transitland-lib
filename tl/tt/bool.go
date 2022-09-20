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
	return r.Val, nil
}

func (r *Bool) Scan(src interface{}) error {
	r.Val, r.Valid = false, false
	if src == nil {
		return nil
	}
	var err error
	switch v := src.(type) {
	case string:
		if v == "" {
			return nil
		}
	// 	r.Val = v
	// case int:
	// 	r.Val = strconv.Itoa(v)
	// case int64:
	// 	r.Val = strconv.Itoa(int(v))
	default:
		err = errors.New("cant convert")
	}
	r.Valid = (err == nil)
	return err
}

func (r *Bool) UnmarshalJSON(v []byte) error {
	r.Val, r.Valid = false, false
	if len(v) == 0 {
		return nil
	}
	err := json.Unmarshal(v, &r.Val)
	r.Valid = (err == nil)
	return err
}

func (r *Bool) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
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
