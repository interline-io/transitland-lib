package snakejson

import (
	"bytes"
	"encoding/json"
	"regexp"
)

// Regexp definitions
var keyMatchRegex = regexp.MustCompile(`\"(\w+)\":`)
var wordBarrierRegex = regexp.MustCompile(`([a-z_0-9])([A-Z])`)

// SnakeMarshaller writes JSON with default snake_case property names.
type SnakeMarshaller struct {
	Value interface{}
}

// MarshalJSON implements json.Marshaler
func (c SnakeMarshaller) MarshalJSON() ([]byte, error) {
	marshalled, err := json.Marshal(c.Value)
	converted := keyMatchRegex.ReplaceAllFunc(
		marshalled,
		func(match []byte) []byte {
			return bytes.ToLower(wordBarrierRegex.ReplaceAll(
				match,
				[]byte(`${1}_${2}`),
			))
		},
	)
	return converted, err
}
