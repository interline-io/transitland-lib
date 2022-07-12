package tt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// IntSlice .
type IntSlice struct {
	Valid bool
	Ints  []int
}

func NewIntSlice(v []int) IntSlice {
	return IntSlice{Valid: true, Ints: v}
}

// Value .
func (a IntSlice) Value() (driver.Value, error) {
	if !a.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(a.Ints)
}

// Scan .
func (a *IntSlice) Scan(value interface{}) error {
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
