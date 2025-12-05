package cmds

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseErrorThresholds(t *testing.T) {
	testCases := []struct {
		name        string
		input       []string
		expected    map[string]float64
		expectError bool
		errorSubstr string
	}{
		{
			name:     "empty input",
			input:    []string{},
			expected: nil,
		},
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "single default threshold",
			input:    []string{"*:10"},
			expected: map[string]float64{"*": 10},
		},
		{
			name:     "single file threshold",
			input:    []string{"stops.txt:5"},
			expected: map[string]float64{"stops.txt": 5},
		},
		{
			name:     "multiple thresholds",
			input:    []string{"*:10", "stops.txt:5", "trips.txt:15"},
			expected: map[string]float64{"*": 10, "stops.txt": 5, "trips.txt": 15},
		},
		{
			name:     "zero threshold",
			input:    []string{"*:0"},
			expected: map[string]float64{"*": 0},
		},
		{
			name:     "decimal threshold",
			input:    []string{"stops.txt:5.5"},
			expected: map[string]float64{"stops.txt": 5.5},
		},
		{
			name:        "empty filename",
			input:       []string{":10"},
			expectError: true,
			errorSubstr: "filename cannot be empty",
		},
		{
			name:        "empty percentage",
			input:       []string{"stops.txt:"},
			expectError: true,
			errorSubstr: "percentage cannot be empty",
		},
		{
			name:        "missing colon",
			input:       []string{"stops.txt10"},
			expectError: true,
			errorSubstr: "expected 'filename:percent'",
		},
		{
			name:        "invalid percentage",
			input:       []string{"stops.txt:abc"},
			expectError: true,
			errorSubstr: "invalid error threshold percentage",
		},
		{
			name:        "negative percentage",
			input:       []string{"stops.txt:-5"},
			expectError: true,
			errorSubstr: "cannot be negative",
		},
		{
			name:     "whitespace trimmed",
			input:    []string{" stops.txt : 10 "},
			expected: map[string]float64{"stops.txt": 10},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseErrorThresholds(tc.input)
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorSubstr != "" {
					assert.Contains(t, err.Error(), tc.errorSubstr)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}
