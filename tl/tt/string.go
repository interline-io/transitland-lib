package tt

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

type String struct {
	Val   string
	Valid bool
}

func NewString(v string) String {
	return String{Valid: true, Val: v}
}

// Value returns nil if empty
func (r String) Value() (driver.Value, error) {
	if !r.Valid {
		return nil, nil
	}
	return r.Val, nil
}

func (r *String) Scan(src interface{}) error {
	r.Val, r.Valid = "", false
	var err error
	switch v := src.(type) {
	case nil:
		return nil
	case string:
		r.Val = v
	case int:
		r.Val = strconv.Itoa(v)
	case int64:
		r.Val = strconv.Itoa(int(v))
	default:
		err = fmt.Errorf("cant convert %T", src)
	}
	r.Valid = (err == nil && r.Val != "")
	return err
}

func (r *String) UnmarshalJSON(v []byte) error {
	r.Val, r.Valid = "", false
	if isEmpty(string(v)) {
		return nil
	}
	if v[0] != '"' && v[len(v)-1] != '"' {
		// Handle unquoted values, e.g. number
		return r.Scan(string(v))
	}
	err := json.Unmarshal(v, &r.Val)
	r.Valid = (err == nil && r.Val != "")
	return err
}

func (r String) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return jsonNull(), nil
	}
	return json.Marshal(r.Val)
}

func (r *String) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

func (r String) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
