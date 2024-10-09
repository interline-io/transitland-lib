package tt

import (
	"database/sql/driver"
	"encoding/json"
)

// Map helps read and write map[string]any as JSON
type Map struct {
	Option[map[string]any]
}

func NewMap(val map[string]any) Map {
	s := Map{Option: NewOption(map[string]any{})}
	for k, v := range val {
		s.Val[k] = v
	}
	return s
}

func (r Map) Value() (driver.Value, error) {
	return json.Marshal(r)
}
