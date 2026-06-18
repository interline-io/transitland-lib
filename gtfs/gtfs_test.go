package gtfs

import (
	"fmt"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tt"
)

/////////////////////////

// ExpectError is an alias to testutil.ExpectError
type ExpectError = testutil.ExpectError

// PE wraps testutil.PE
func PE(errorStrings ...string) []ExpectError {
	return testutil.ParseExpectErrors(errorStrings...)
}

// CheckErrors wraps testutil.CheckErrors
func CheckErrors(expectedErrors []ExpectError, errs []error, t *testing.T) {
	testutil.CheckErrors(expectedErrors, errs, t)
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
