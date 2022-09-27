package tt

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
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
		return r.Val, nil
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
		r.Val, err = strconv.Atoi(v) // strconv.ParseInt(v, 10, 64)
	case int:
		r.Val = int(v)
	case int64:
		r.Val = int(v)
	case float64:
		r.Val = int(v)
	default:
		err = fmt.Errorf("cant convert %T to Int", src)
	}
	r.Valid = (err == nil)
	return err
}

func (r *Int) String() string {
	return strconv.Itoa(int(r.Val))
}

func (r *Int) UnmarshalJSON(v []byte) error {
	r.Val, r.Valid = 0, false
	if len(v) == 0 {
		return nil
	}
	var j json.Number
	err := json.Unmarshal(v, &j)
	if err != nil {
		return err
	}
	rr := int64(0)
	rr, err = j.Int64()
	r.Val = int(rr)
	r.Valid = (err == nil)
	return err

}

func (r Int) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return jsonNull(), nil
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
