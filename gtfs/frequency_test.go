package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tt"
)

func TestFrequency_Errors(t *testing.T) {
	newFrequency := func(fn func(*Frequency)) *Frequency {
		frequency := &Frequency{
			TripID:      tt.NewString("ok"),
			StartTime:   tt.NewSeconds(3600), // 01:00:00
			EndTime:     tt.NewSeconds(7200), // 02:00:00
			HeadwaySecs: tt.NewInt(600),
			ExactTimes:  tt.NewInt(1),
		}
		if fn != nil {
			fn(frequency)
		}
		return frequency
	}

	testcases := []struct {
		name           string
		entity         *Frequency
		expectedErrors []testutil.ExpectError
	}{
		{
			name:           "Valid frequency",
			entity:         newFrequency(nil),
			expectedErrors: nil,
		},
		{
			name: "end_time before start_time",
			entity: newFrequency(func(f *Frequency) {
				f.StartTime = tt.NewSeconds(3600) // 01:00:00
				f.EndTime = tt.NewSeconds(1800)   // 00:30:00
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:end_time"),
		},
		{
			name: "Missing headway_secs",
			entity: newFrequency(func(f *Frequency) {
				f.HeadwaySecs = tt.Int{}
			}),
			expectedErrors: ParseExpectErrors("RequiredFieldError:headway_secs"),
		},
		{
			name: "Missing start_time",
			entity: newFrequency(func(f *Frequency) {
				f.StartTime = tt.Seconds{}
			}),
			expectedErrors: ParseExpectErrors("RequiredFieldError:start_time"),
		},
		{
			name: "Missing end_time",
			entity: newFrequency(func(f *Frequency) {
				f.EndTime = tt.Seconds{}
			}),
			expectedErrors: ParseExpectErrors("RequiredFieldError:end_time"),
		},
		{
			name: "Invalid headway_secs (zero)",
			entity: newFrequency(func(f *Frequency) {
				f.HeadwaySecs = tt.NewInt(0)
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:headway_secs"),
		},
		{
			name: "Invalid exact_times",
			entity: newFrequency(func(f *Frequency) {
				f.ExactTimes = tt.NewInt(2)
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:exact_times"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.entity)
			testutil.CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}
