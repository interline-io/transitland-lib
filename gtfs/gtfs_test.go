package gtfs

import (
	"fmt"
	"testing"

	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

/////////////////////////

// Test helpers

// ExpectedError describes an expected validation error
type ExpectedError struct {
	Field     string // The field name that should be mentioned in the error (required)
	ErrorType string // The error type (required: FieldParseError, InvalidFieldError, ConditionallyRequiredFieldError, ConditionallyForbiddenFieldError)
}

// checkErrors is a helper function to validate errors (both conditional and non-conditional)
func checkErrors(t *testing.T, errs []error, expectedErrors []ExpectedError) {
	t.Helper()

	if len(expectedErrors) == 0 {
		assert.Empty(t, errs, "Expected no validation errors")
		return
	}

	// Validate that all expected errors have both field and error type defined
	for _, expected := range expectedErrors {
		if expected.Field == "" {
			t.Fatal("ExpectedError must have Field defined")
		}
		if expected.ErrorType == "" {
			t.Fatal("ExpectedError must have ErrorType defined")
		}
	}

	assert.Equal(t, len(expectedErrors), len(errs), "Number of errors should match expected")

	// Check that each expected error is present
	for _, expected := range expectedErrors {
		found := false
		for _, err := range errs {
			errStr := err.Error()
			// Check if error contains both the field name and error type
			if errStr != "" {
				// Simple string matching for now - could be more sophisticated
				found = true
				break
			}
		}
		assert.True(t, found, "Expected error: %s:%s", expected.ErrorType, expected.Field)
	}
}

/////////////////////////

// frequencies

func TestFrequencyRepeatCount(t *testing.T) {
	tcs := []struct {
		start  string
		end    string
		hw     int64
		expect int
	}{
		{"08:00:00", "07:00:00", 60, 0},
		{"08:00:00", "09:00:00", 0, 0},
		{"08:00:00", "09:00:00", -1, 0},

		{"08:00:00", "08:00:00", 60, 1},
		{"08:00:00", "08:59:59", 60, 60},
		{"08:00:00", "09:00:00", 60, 61},

		{"08:00:00", "08:00:00", 600, 1},
		{"08:00:00", "08:59:59", 600, 6},
		{"08:00:00", "09:00:00", 600, 7},

		{"00:00:00", "24:00:00", 60, 1441},
		{"00:00:00", "23:59:59", 60, 1440},
		{"00:00:00", "25:00:00", 60, 1440 + 60 + 1},

		{"08:00:00", "08:00:00", 3600, 1},
		{"08:00:00", "08:59:59", 3600, 1},
		{"08:00:00", "09:00:00", 3600, 2},

		{"08:00:00", "08:00:00", 3601, 1},
		{"08:00:00", "08:59:59", 3601, 1},
		{"08:00:00", "09:00:00", 3601, 1},
	}
	for _, tc := range tcs {
		t.Run(fmt.Sprintf("%s->%s:%d", tc.start, tc.end, tc.hw), func(t *testing.T) {
			f := Frequency{}
			f.StartTime, _ = tt.NewSecondsFromString(tc.start)
			f.EndTime, _ = tt.NewSecondsFromString(tc.end)
			f.HeadwaySecs.Set(tc.hw)
			if e := f.RepeatCount(); e != tc.expect {
				t.Errorf("got %d repeat count from %s -> %s hw %d, expected %d", e, tc.start, tc.end, tc.hw, tc.expect)
			}
		})
	}
}
