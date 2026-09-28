package gbfs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalizedString_UnmarshalJSON(t *testing.T) {
	tcs := []struct {
		name   string
		json   string
		expect LocalizedString
	}{
		{"2.x string", `"Café \/ 1st"`, LocalizedString{{Text: "Café / 1st"}}},
		{"3.x array", `[{"text":"Gare","language":"fr"},{"text":"Station","language":"en"}]`, LocalizedString{{Text: "Gare", Language: "fr"}, {Text: "Station", Language: "en"}}},
		{"number", `12`, LocalizedString{{Text: "12"}}},
		{"null", `null`, nil},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var l LocalizedString
			assert.NoError(t, json.Unmarshal([]byte(tc.json), &l))
			assert.Equal(t, tc.expect, l)
		})
	}
}
