package ext

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseExtensionArgs(t *testing.T) {
	tcs := []struct {
		Name        string
		Value       string
		ExpectName  string
		ExpectArgs  string
		ExpectError bool
	}{
		{"test (no args)", "test", "test", ``, false},
		{"test a=b", "test:a=b", "test", `{"a":"b"}`, false},
		{"test a=1 numeric", `test:a=1`, "test", `{"a":1}`, false},
		{"test a=b json", `test:{"a":"b"}`, "test", `{"a":"b"}`, false},
		{"test a=b json numeric", `test:{"a":1}`, "test", `{"a":1}`, false},
		{"test a=b,c=d", "test:a=b,c=d", "test", `{"a":"b","c":"d"}`, false},
		{"test a=b,c=d json", `test:{"a":"b","c":"d"}`, "test", `{"a":"b","c":"d"}`, false},
	}
	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			extName, extArgs, err := ParseExtensionArgs(tc.Value)
			assert.Equal(t, tc.ExpectName, extName)
			if tc.ExpectArgs != "" {
				assert.JSONEq(t, tc.ExpectArgs, extArgs)
			}
			if err != nil && tc.ExpectError {
			} else if err != nil && !tc.ExpectError {
				t.Errorf("unexpected error: %s", err.Error())
			} else if err == nil && tc.ExpectError {
				t.Error("expected error, got none")
			}
		})
	}
}
