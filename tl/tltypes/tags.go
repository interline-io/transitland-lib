package tltypes

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
)

// Tags is a map[string]string with json and gql marshal support.
// This is a struct instead of bare map[string]string because of a gqlgen issue.
type Tags struct {
	tags map[string]string
}

// Value .
func (r Tags) Value() (driver.Value, error) {
	return json.Marshal(r.tags)
}

// Scan .
func (r *Tags) Scan(value interface{}) error {
	r.tags = nil
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &r.tags)
}

// MarshalJSON implements the json.marshaler interface.
func (r *Tags) MarshalJSON() ([]byte, error) {
	if r.tags == nil {
		return []byte("null"), nil
	}
	return json.Marshal(r.tags)
}

// MarshalGQL implements the graphql.Marshaler interface
func (r Tags) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}

// UnmarshalJSON implements json.Marshaler interface.
func (r *Tags) UnmarshalJSON(v []byte) error {
	r.tags = nil
	if len(v) == 0 {
		return nil
	}
	return json.Unmarshal(v, &r.tags)
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (r *Tags) UnmarshalGQL(v interface{}) error {
	rt := map[string]string{}
	a, ok := v.(map[string]interface{})
	if !ok {
		return errors.New("cannot unmarshal")
	}
	for k, value := range a {
		if c, ok := value.(string); ok {
			rt[k] = c
		} else {
			return errors.New("cannot unmarshal")
		}
	}
	r.tags = rt
	return nil
}

// Keys return the tag keys
func (r *Tags) Keys() []string {
	var ret []string
	for k := range r.tags {
		ret = append(ret, k)
	}
	return ret
}

// Set a tag value
func (r *Tags) Set(k, v string) {
	if r.tags == nil {
		r.tags = map[string]string{}
	}
	r.tags[k] = v
}

// Get a tag value by key
func (r *Tags) Get(k string) (string, bool) {
	if r.tags == nil {
		return "", false
	}
	a, ok := r.tags[k]
	return a, ok
}
