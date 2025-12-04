package gtfs

import (
	"fmt"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

/////////////////////////

// Test helpers - local implementations to avoid import cycle with internal/testutil

// ExpectError describes an expected validation error (matches testreader.ExpectError interface)
type ExpectError struct {
	Field     string
	ErrorType string
	Filename  string
	EntityID  string
}

// ParseExpectErrors converts shorthand error strings to ExpectError structs
// Format: "ErrorType:Field:Filename:EntityID" (Filename and EntityID are optional)
func ParseExpectErrors(errorStrings ...string) []ExpectError {
	var errors []ExpectError
	for _, s := range errorStrings {
		parts := strings.Split(s, ":")
		// Pad to 4 parts
		for len(parts) < 4 {
			parts = append(parts, "")
		}
		errors = append(errors, ExpectError{
			ErrorType: parts[0],
			Field:     parts[1],
			Filename:  parts[2],
			EntityID:  parts[3],
		})
	}
	return errors
}

// CheckErrors validates that actual errors match expected errors
func CheckErrors(expectedErrors []ExpectError, errs []error, t *testing.T) {
	t.Helper()

	if len(expectedErrors) == 0 && len(errs) == 0 {
		return
	}

	if len(expectedErrors) != len(errs) {
		t.Errorf("Expected %d errors, got %d", len(expectedErrors), len(errs))
		for _, err := range errs {
			t.Logf("  Actual error: %v", err)
		}
		return
	}

	// Check that each expected error matches an actual error
	for i, expected := range expectedErrors {
		if i >= len(errs) {
			break
		}
		errStr := errs[i].Error()
		errType := fmt.Sprintf("%T", errs[i])

		// Extract just the type name without package path
		if idx := strings.LastIndex(errType, "."); idx >= 0 {
			errType = errType[idx+1:]
		}
		// Remove pointer marker
		errType = strings.TrimPrefix(errType, "*")

		// Check that error string contains the expected field
		if expected.Field != "" && !strings.Contains(errStr, expected.Field) {
			t.Errorf("Error %d: expected field %q not found in error: %v", i, expected.Field, errStr)
		}

		// Check error type matches
		if expected.ErrorType != "" && errType != expected.ErrorType {
			t.Errorf("Error %d: expected error type %q, got %q in error: %v", i, expected.ErrorType, errType, errStr)
		}
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
