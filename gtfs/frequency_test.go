package gtfs

import (
	"fmt"
	"slices"
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
		expectedErrors []ExpectError
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
			expectedErrors: PE("InvalidFieldError:end_time"),
		},
		{
			name: "end_time equal to start_time",
			entity: newFrequency(func(f *Frequency) {
				f.EndTime = tt.NewSeconds(3600) // 01:00:00
			}),
			expectedErrors: PE("InvalidFieldError:end_time"),
		},
		{
			name: "end_time 00:00:00 before start_time",
			entity: newFrequency(func(f *Frequency) {
				f.StartTime = tt.NewSeconds(82800) // 23:00:00
				f.EndTime = tt.NewSeconds(0)
			}),
			expectedErrors: PE("InvalidFieldError:end_time"),
		},
		{
			name: "start_time 00:00:00",
			entity: newFrequency(func(f *Frequency) {
				f.StartTime = tt.NewSeconds(0)
			}),
			expectedErrors: nil,
		},
		{
			name: "Missing headway_secs",
			entity: newFrequency(func(f *Frequency) {
				f.HeadwaySecs = tt.Int{}
			}),
			expectedErrors: PE("RequiredFieldError:headway_secs"),
		},
		{
			name: "Missing start_time",
			entity: newFrequency(func(f *Frequency) {
				f.StartTime = tt.Seconds{}
			}),
			expectedErrors: PE("RequiredFieldError:start_time"),
		},
		{
			name: "Missing end_time",
			entity: newFrequency(func(f *Frequency) {
				f.EndTime = tt.Seconds{}
			}),
			expectedErrors: PE("RequiredFieldError:end_time"),
		},
		{
			name: "Invalid headway_secs (zero)",
			entity: newFrequency(func(f *Frequency) {
				f.HeadwaySecs = tt.NewInt(0)
			}),
			expectedErrors: PE("InvalidFieldError:headway_secs"),
		},
		{
			name: "headway_secs under a minute",
			entity: newFrequency(func(f *Frequency) {
				f.HeadwaySecs = tt.NewInt(59)
			}),
			expectedErrors: PE("InvalidFieldError:headway_secs"),
		},
		{
			name: "headway_secs of a minute",
			entity: newFrequency(func(f *Frequency) {
				f.HeadwaySecs = tt.NewInt(60)
			}),
			expectedErrors: nil,
		},
		{
			name: "Invalid exact_times",
			entity: newFrequency(func(f *Frequency) {
				f.ExactTimes = tt.NewInt(2)
			}),
			expectedErrors: PE("InvalidFieldError:exact_times"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.entity)
			testutil.CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

func TestFrequency_Starts(t *testing.T) {
	newFrequency := func(start string, end string, hw int64) Frequency {
		f := Frequency{}
		f.StartTime, _ = tt.NewSecondsFromString(start)
		f.EndTime, _ = tt.NewSecondsFromString(end)
		f.HeadwaySecs.Set(hw)
		return f
	}
	tcs := []struct {
		start  string
		end    string
		hw     int64
		expect int
	}{
		{"08:00:00", "07:00:00", 60, 0},
		{"08:00:00", "09:00:00", 0, 0},
		{"08:00:00", "09:00:00", -1, 0},

		// No trip starts at end_time, so a period that ends where it begins runs none
		{"08:00:00", "08:00:00", 60, 0},
		{"08:00:00", "08:59:59", 60, 60},
		{"08:00:00", "09:00:00", 60, 60},
		{"08:00:00", "09:00:01", 60, 61},

		{"08:00:00", "08:00:00", 600, 0},
		{"08:00:00", "08:59:59", 600, 6},
		{"08:00:00", "09:00:00", 600, 6},

		{"00:00:00", "24:00:00", 60, 1440},
		{"00:00:00", "23:59:59", 60, 1440},
		{"00:00:00", "25:00:00", 60, 1500},

		{"08:00:00", "08:00:00", 3600, 0},
		{"08:00:00", "08:59:59", 3600, 1},
		{"08:00:00", "09:00:00", 3600, 1},

		{"08:00:00", "08:00:00", 3601, 0},
		{"08:00:00", "08:59:59", 3601, 1},
		{"08:00:00", "09:00:00", 3601, 1},
	}
	for _, tc := range tcs {
		t.Run(fmt.Sprintf("%s->%s:%d", tc.start, tc.end, tc.hw), func(t *testing.T) {
			f := newFrequency(tc.start, tc.end, tc.hw)
			starts := f.Starts()
			if len(starts) != tc.expect {
				t.Errorf("got %d starts from %s -> %s hw %d, expected %d", len(starts), tc.start, tc.end, tc.hw, tc.expect)
			}
			if n := f.RepeatCount(); n != tc.expect {
				t.Errorf("got %d repeat count from %s -> %s hw %d, expected %d", n, tc.start, tc.end, tc.hw, tc.expect)
			}
			for i, s := range starts {
				if expect := f.StartTime.Int() + i*int(tc.hw); s != expect {
					t.Errorf("start %d: got %d, expected %d", i, s, expect)
				}
			}
		})
	}
	t.Run("periods that meet", func(t *testing.T) {
		// A period may begin where the previous one ends; the start there belongs to the second
		var starts []int
		for _, f := range []Frequency{
			newFrequency("08:00:00", "09:00:00", 1800),
			newFrequency("09:00:00", "10:00:00", 1800),
		} {
			starts = append(starts, f.Starts()...)
		}
		expect := []int{8 * 3600, 8*3600 + 1800, 9 * 3600, 9*3600 + 1800}
		if !slices.Equal(starts, expect) {
			t.Errorf("got starts %v, expected %v", starts, expect)
		}
	})
}
