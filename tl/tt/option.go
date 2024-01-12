package tt

import (
	"database/sql/driver"
	"encoding/json"
)

type Option[T any] struct {
	Val   T
	Valid bool
}

func (r *Option[T]) Present() bool {
	return r.Valid
}

func (r Option[T]) String() string {
	if !r.Valid {
		return ""
	}
	out := ""
	if err := convertAssign(&out, r.Val); err != nil {
		b, _ := r.MarshalJSON()
		return string(b)
	}
	return out
}

func (r *Option[T]) Error() error {
	return nil
}

func (r *Option[T]) Scan(src interface{}) error {
	err := convertAssign(&r.Val, src)
	r.Valid = (src != nil && err == nil)
	return err
}

func (r Option[T]) Value() (driver.Value, error) {
	if !r.Valid {
		return nil, nil
	}
	return r.Val, nil
}

func (r *Option[T]) UnmarshalJSON(v []byte) error {
	err := json.Unmarshal(v, &r.Val)
	r.Valid = (err == nil)
	return err
}

func (r Option[T]) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Val)
}
