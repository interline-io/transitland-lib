package enum

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// Ints is a slice of ints.
type Ints struct {
	Ints  []int
	Valid bool
}

func NewInts(v []int) Ints {
	return Ints{Valid: true, Ints: v}
}

// Value .
func (a Ints) Value() (driver.Value, error) {
	if !a.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(a.Ints)
}

// Scan .
func (a *Ints) Scan(value interface{}) error {
	a.Ints, a.Valid = nil, false
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a.Ints)
}
