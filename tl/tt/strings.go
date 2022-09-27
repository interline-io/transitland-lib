package tt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
)

// Strings helps read and write []String as JSON
type Strings []String

func (r Strings) Value() (driver.Value, error) {
	return json.Marshal(r)
}

func (r *Strings) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &r)
}

func (r *Strings) UnmarshalJSON(v []byte) error {
	var ss []string
	err := json.Unmarshal(v, &ss)
	if err != nil {
		return err
	}
	var out []String
	for _, s := range ss {
		out = append(out, NewString(s))
	}
	*r = out
	return nil
}

func (r Strings) MarshalJSON() ([]byte, error) {
	if len(r) == 0 {
		return jsonNull(), nil
	}
	var out []string
	for _, s := range r {
		out = append(out, s.Val)
	}
	return json.Marshal(out)
}

func (r *Strings) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

func (r Strings) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
