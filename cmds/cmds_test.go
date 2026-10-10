package cmds

import (
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
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

func TestParseSecretEnv(t *testing.T) {
	t.Setenv("TEST_SECRET_ENV_KEY", "abcd")
	testCases := []struct {
		name        string
		input       string
		expected    dmfr.Secret
		expectError bool
	}{
		{name: "feed id", input: "f-test:TEST_SECRET_ENV_KEY", expected: dmfr.Secret{Key: "abcd", FeedID: "f-test"}},
		{name: "filename", input: "test.dmfr.json:TEST_SECRET_ENV_KEY", expected: dmfr.Secret{Key: "abcd", Filename: "test.dmfr.json"}},
		{name: "host", input: "f-test:TEST_SECRET_ENV_KEY:*.mta.info", expected: dmfr.Secret{Key: "abcd", FeedID: "f-test", Host: "*.mta.info"}},
		{name: "empty host", input: "f-test:TEST_SECRET_ENV_KEY:", expectError: true},
		{name: "no env var", input: "f-test", expectError: true},
		{name: "unset env var", input: "f-test:TEST_SECRET_ENV_UNSET", expectError: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			secret, err := parseSecretEnv(tc.input)
			if tc.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, secret)
		})
	}
}
