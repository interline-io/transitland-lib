package dbutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeLike(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		prefix   bool
		suffix   bool
		expected string
	}{
		{
			name:     "no escaping needed, no wildcards",
			input:    "hello",
			prefix:   false,
			suffix:   false,
			expected: "hello",
		},
		{
			name:     "suffix wildcard only",
			input:    "hello",
			prefix:   false,
			suffix:   true,
			expected: "hello%",
		},
		{
			name:     "prefix wildcard only",
			input:    "hello",
			prefix:   true,
			suffix:   false,
			expected: "%hello",
		},
		{
			name:     "both wildcards",
			input:    "hello",
			prefix:   true,
			suffix:   true,
			expected: "%hello%",
		},
		{
			name:     "escape percent sign",
			input:    "100%",
			prefix:   false,
			suffix:   false,
			expected: "100\\%",
		},
		{
			name:     "escape underscore",
			input:    "hello_world",
			prefix:   false,
			suffix:   false,
			expected: "hello\\_world",
		},
		{
			name:     "escape backslash",
			input:    "path\\to\\file",
			prefix:   false,
			suffix:   false,
			expected: "path\\\\to\\\\file",
		},
		{
			name:     "escape all special chars with wildcards",
			input:    "test%_\\value",
			prefix:   true,
			suffix:   true,
			expected: "%test\\%\\_\\\\value%",
		},
		{
			name:     "real geoid prefix",
			input:    "0500000US",
			prefix:   false,
			suffix:   true,
			expected: "0500000US%",
		},
		{
			name:     "real geoid prefix with tract",
			input:    "1400000US0600140",
			prefix:   false,
			suffix:   true,
			expected: "1400000US0600140%",
		},
		{
			name:     "contains search pattern",
			input:    "transit",
			prefix:   true,
			suffix:   true,
			expected: "%transit%",
		},
		{
			name:     "empty string with wildcards",
			input:    "",
			prefix:   true,
			suffix:   true,
			expected: "%%",
		},
		{
			name:     "empty string no wildcards",
			input:    "",
			prefix:   false,
			suffix:   false,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := EscapeLike(tc.input, tc.prefix, tc.suffix)
			assert.Equal(t, tc.expected, result)
		})
	}
}
