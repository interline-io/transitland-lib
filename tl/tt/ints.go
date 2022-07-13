package tt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// Ints .
type Ints struct {
	Valid bool
	Val   []int
}

func NewInts(v []int) Ints {
	return Ints{Valid: true, Val: v}
}

// Value .
func (a Ints) Value() (driver.Value, error) {
	if !a.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(a.Val)
}

// Scan .
func (a *Ints) Scan(value interface{}) error {
	a.Val, a.Valid = nil, false
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a.Val)
}
