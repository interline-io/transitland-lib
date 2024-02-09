package tt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
)

// Counts is a simple map[string]int with json support
type Counts map[string]int

func (r Counts) Value() (driver.Value, error) {
	return json.Marshal(r)
}

func (r *Counts) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &r)
}

func (r *Counts) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

func (r Counts) MarshalGQL(w io.Writer) {
	b, _ := json.Marshal(&r)
	w.Write(b)
}
