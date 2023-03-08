package tt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// Ints is a nullable slice of []int
type Ints struct {
	Valid bool
	Val   []int
}

func NewInts(v []int) Ints {
	return Ints{Valid: true, Val: v}
}

func (a Ints) Value() (driver.Value, error) {
	if !a.Valid {
		return jsonNull(), nil
	}
	return json.Marshal(a.Val)
}

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
