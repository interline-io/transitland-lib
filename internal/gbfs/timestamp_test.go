package gbfs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTimestamp_UnmarshalJSON(t *testing.T) {
	tcs := []struct {
		name   string
		json   string
		expect int64
		valid  bool
	}{
		{"posix", `1689593653`, 1689593653, true},
		{"rfc3339 offset", `"2023-07-17T13:34:13+02:00"`, 1689593653, true},
		{"rfc3339 fraction", `"2023-07-17T11:34:13.473623459Z"`, 1689593653, true},
		{"unparseable", `"yesterday"`, 0, false},
		{"null", `null`, 0, false},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var v Timestamp
			assert.NoError(t, json.Unmarshal([]byte(tc.json), &v))
			assert.Equal(t, tc.valid, v.Valid)
			assert.Equal(t, tc.expect, v.Val)
		})
	}
}
