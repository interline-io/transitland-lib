package tt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
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
