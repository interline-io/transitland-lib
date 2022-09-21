package tt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"strconv"
)

// Key is a nullable foreign key constraint, similar to sql.NullString
type Key struct {
	Val   string
	Valid bool
}

func NewKey(v string) Key {
	return Key{Valid: true, Val: v}
}

func (r *Key) String() string {
	return r.Val
}

func (r Key) Value() (driver.Value, error) {
	if !r.Valid || r.Val == "" {
		return nil, nil
	}
	return r.Val, nil
}

func (r *Key) Scan(src interface{}) error {
	r.Val, r.Valid = "", false
	if src == nil {
		return nil
	}
	var err error
	switch v := src.(type) {
	case string:
		if v == "" {
			return nil
		}
		r.Val = v
	case int:
		r.Val = strconv.Itoa(v)
	case int64:
		r.Val = strconv.Itoa(int(v))
	default:
		err = errors.New("cant convert")
	}
	r.Valid = (err == nil && r.Val != "")
	return err
}

func (r *Key) Int() int {
	a, _ := strconv.Atoi(r.Val)
	return a
}

func (r *Key) UnmarshalJSON(v []byte) error {
	r.Val, r.Valid = "", false
	if len(v) == 0 {
		return nil
	}
	err := json.Unmarshal(v, &r.Val)
	r.Valid = (err == nil)
	return err
}

func (r Key) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Val)
}

func (r *Key) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

func (r Key) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
