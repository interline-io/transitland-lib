package tt

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

// Float is a nullable float64
type Float struct {
	Val   float64
	Valid bool
}

func NewFloat(v float64) Float {
	return Float{Valid: true, Val: v}
}

func (r Float) Value() (driver.Value, error) {
	if r.Valid {
		return r.Val, nil
	}
	return nil, nil
}

func (r *Float) Scan(src interface{}) error {
	r.Val, r.Valid = 0.0, false
	if src == nil {
		return nil
	}
	var err error
	switch v := src.(type) {
	case string:
		r.Val, err = strconv.ParseFloat(v, 64)
	case int:
		r.Val = float64(v)
	case int64:
		r.Val = float64(v)
	case float64:
		r.Val = v
	default:
		err = fmt.Errorf("cant convert %T to Float", src)
	}
	r.Valid = (err == nil)
	return err
}

func (r *Float) String() string {
	if r.Val > -100_000 && r.Val < 100_000 {
		return fmt.Sprintf("%g", r.Val)
	}
	return fmt.Sprintf("%0.5f", r.Val)
}

func (r *Float) UnmarshalJSON(v []byte) error {
	r.Val, r.Valid = 0, false
	if len(v) == 0 {
		return nil
	}
	var j json.Number
	err := json.Unmarshal(v, &j)
	if err != nil {
		return err
	}
	r.Val, err = j.Float64()
	r.Valid = (err == nil)
	return err
}

func (r Float) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return jsonNull(), nil
	}
	return json.Marshal(r.Val)
}

func (r *Float) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

func (r Float) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
