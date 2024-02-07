package tt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
)

// Map helps read and write map[string]any as JSON
type Map struct {
	Valid bool
	Val   map[string]any
}

func NewMap(val map[string]any) Map {
	s := Map{
		Val:   map[string]any{},
		Valid: true,
	}
	for k, v := range val {
		s.Val[k] = v
	}
	return s
}

func (r Map) Value() (driver.Value, error) {
	return json.Marshal(r)
}

func (r *Map) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &r)
}

func (r *Map) UnmarshalJSON(v []byte) error {
	ss := map[string]any{}
	err := json.Unmarshal(v, &ss)
	if err != nil {
		return err
	}
	r.Val = ss
	r.Valid = (err == nil)
	return nil
}

func (r Map) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return jsonNull(), nil
	}
	return json.Marshal(r.Val)
}

func (r *Map) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

func (r Map) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
