package tt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
)

// Strings helps read and write []String as JSON
type Strings []String

func (a Strings) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *Strings) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}

func (r *Strings) UnmarshalJSON(v []byte) error {
	return nil
}

func (r Strings) MarshalJSON() ([]byte, error) {
	if r == nil {
		return []byte("null"), nil
	}
	return json.Marshal(r)
}

func (r *Strings) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

func (r Strings) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
