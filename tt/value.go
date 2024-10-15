package tt

import (
	"database/sql/driver"
	"encoding/json"
)

type Value[T any] struct {
	Val T
}

func NewValue[T any](v T) Value[T] {
	return Value[T]{Val: v}
}

func (r Value[T]) IsValid() bool {
	return true
}

func (r Value[T]) Check() error {
	return nil
}

func (r *Value[T]) Set(v T) {
	r.Val = v
}

func (r Value[T]) IsPresent() bool {
	return true
}

func (r *Value[T]) IsZero() bool {
	return false
}

func (r Value[T]) String() string {
	out := ""
	if err := convertAssign(&out, r.Val); err != nil {
		b, _ := r.MarshalJSON()
		return string(b)
	}
	return out
}

func (r *Value[T]) Scan(src interface{}) error {
	return convertAssign(&r.Val, src)
}

func (r Value[T]) Value() (driver.Value, error) {
	if driver.IsValue(r.Val) {
		return r.Val, nil
	}
	return json.Marshal(r.Val)
}

func (r *Value[T]) UnmarshalJSON(v []byte) error {
	return r.Scan(stripQuotes(v))
}

func (r Value[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Val)
}

//////

type DefaultInt struct {
	Value[int64]
}

func (r DefaultInt) Int() int {
	return int(r.Val)
}
