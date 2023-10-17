package tt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// Counts is a simple map[string]int with json support
type Counts map[string]int

func (a Counts) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *Counts) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}
