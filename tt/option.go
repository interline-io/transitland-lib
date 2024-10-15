package tt

import (
	"database/sql/driver"
	"encoding/json"
)

type Option[T any] struct {
	Val   T
	Valid bool
}

func NewOption[T any](v T) Option[T] {
	return Option[T]{Val: v, Valid: true}
}

func (r Option[T]) IsValid() bool {
	return r.Valid
}

func (r Option[T]) Check() error {
	return nil
}

func (r *Option[T]) Set(v T) {
	r.Val = v
	r.Valid = true
}

func (r *Option[T]) Unset() {
	r.Valid = false
}

func (r Option[T]) IsPresent() bool {
	return r.Valid
}

func (r *Option[T]) IsZero() bool {
	return !r.Valid
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
	if driver.IsValue(r.Val) {
		return r.Val, nil
	}
	return json.Marshal(r.Val)
}

func (r *Option[T]) UnmarshalJSON(v []byte) error {
	return r.Scan(stripQuotes(v))
}

func (r Option[T]) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Val)
}

func (r Option[T]) Ptr() *T {
	if r.Valid {
		return &r.Val
	}
	return nil
}
