package tt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeAt(t *testing.T) {
	when, err := time.Parse("2006-01-02T15:04:05", "2024-07-16T01:30:00")
	if err != nil {
		t.Fatal(err)
	}
	cl := &MockClock{T: when}

	now := cl.Now()

	tcs := []struct {
		name         string
		date         string
		wt           string
		tz           string
		startDate    string
		endDate      string
		fallbackWeek string
		useFallback  bool
		expect       string
		expectError  bool
	}{
		{
			name:   "empty (now)",
			date:   "",
			tz:     "UTC",
			expect: "2024-07-16T01:30:00Z",
		},
		{
			name:   "empty (now, default UTC)",
			expect: "2024-07-16T01:30:00Z",
		},
		{
			name:   "date",
			date:   now.Format("2006-01-02"),
			tz:     "UTC",
			expect: "2024-07-16T01:30:00Z",
		},
		{
			name:        "date with wt",
			date:        now.Format("2006-01-02"),
			wt:          "10:00:00",
			tz:          "UTC",
			useFallback: false,
			expect:      "2024-07-16T10:00:00Z",
		},
		{
			name:        "date with wt and tz",
			date:        now.Format("2006-01-02"),
			wt:          "10:00:00",
			tz:          "America/Los_Angeles",
			useFallback: false,
			expect:      "2024-07-16T10:00:00-07:00",
		},
		{
			name:   "now with tz",
			date:   "now",
			wt:     "",
			tz:     "America/Los_Angeles",
			expect: "2024-07-15T18:30:00-07:00",
		},
		{
			name:   "next-friday with now",
			date:   "next-friday",
			wt:     "",
			tz:     "America/Los_Angeles",
			expect: "2024-07-19T18:30:00-07:00",
		},
		{
			name:   "next-sunday with wt",
			date:   "next-sunday",
			wt:     "10:00:00",
			tz:     "America/Los_Angeles",
			expect: "2024-07-21T10:00:00-07:00",
		},
		{
			name:   "next-monday with wt",
			date:   "next-monday",
			wt:     "10:00:00",
			tz:     "America/Los_Angeles",
			expect: "2024-07-15T10:00:00-07:00",
		}, {
			name:   "next-tuesday with wt",
			date:   "next-tuesday",
			wt:     "10:00:00",
			tz:     "America/Los_Angeles",
			expect: "2024-07-16T10:00:00-07:00",
		}, {
			name:   "next-wednesday with wt",
			date:   "next-wednesday",
			wt:     "10:00:00",
			tz:     "America/Los_Angeles",
			expect: "2024-07-17T10:00:00-07:00",
		}, {
			name:   "next-thursday with wt",
			date:   "next-thursday",
			wt:     "10:00:00",
			tz:     "America/Los_Angeles",
			expect: "2024-07-18T10:00:00-07:00",
		}, {
			name:   "next-friday with wt",
			date:   "next-friday",
			wt:     "10:00:00",
			tz:     "America/Los_Angeles",
			expect: "2024-07-19T10:00:00-07:00",
		}, {
			name:   "next-saturday with wt",
			date:   "next-saturday",
			wt:     "10:00:00",
			tz:     "America/Los_Angeles",
			expect: "2024-07-20T10:00:00-07:00",
		},
		// Fallback tests
		{
			name:         "use fallback, same as start date",
			date:         "2024-07-15",
			wt:           "10:00:00",
			tz:           "America/Los_Angeles",
			startDate:    "2024-07-15",
			endDate:      "2024-07-21",
			fallbackWeek: "2021-01-04",
			useFallback:  true,
			expect:       "2024-07-15T10:00:00-07:00",
		},
		{
			name:         "use fallback, same as end date",
			date:         "2024-07-21",
			wt:           "10:00:00",
			tz:           "America/Los_Angeles",
			startDate:    "2024-07-15",
			endDate:      "2024-07-21",
			fallbackWeek: "2021-01-04",
			useFallback:  true,
			expect:       "2024-07-21T10:00:00-07:00",
		},
		// NOTE: some are -08:00 because fallback week standard time
		{
			name:         "use fallback, one before start",
			date:         "2024-07-14",
			wt:           "10:00:00",
			tz:           "America/Los_Angeles",
			startDate:    "2024-07-15",
			endDate:      "2024-07-21",
			fallbackWeek: "2021-01-04",
			useFallback:  true,
			expect:       "2021-01-10T10:00:00-08:00",
		},
		{
			name:         "use fallback, one day after end",
			date:         "2024-07-22",
			wt:           "10:00:00",
			tz:           "America/Los_Angeles",
			startDate:    "2024-07-15",
			endDate:      "2024-07-21",
			fallbackWeek: "2021-01-04",
			useFallback:  true,
			expect:       "2021-01-04T10:00:00-08:00",
		},
		{
			name:         "use fallback",
			date:         "now",
			wt:           "10:00:00",
			tz:           "America/Los_Angeles",
			startDate:    "2020-01-01",
			endDate:      "2021-02-01",
			fallbackWeek: "2021-01-04",
			useFallback:  true,
			expect:       "2021-01-04T10:00:00-08:00",
		},
		{
			name:         "next-sunday with wt with fallback",
			date:         "next-sunday",
			wt:           "10:00:00",
			tz:           "America/Los_Angeles",
			startDate:    "2020-01-01",
			endDate:      "2021-02-01",
			fallbackWeek: "2021-01-04",
			useFallback:  true,
			expect:       "2021-01-10T10:00:00-08:00",
		},
		{
			name:         "next-monday with wt with fallback",
			date:         "next-monday",
			wt:           "10:00:00",
			tz:           "America/Los_Angeles",
			startDate:    "2020-01-01",
			endDate:      "2021-02-01",
			fallbackWeek: "2021-01-04",
			useFallback:  true,
			expect:       "2021-01-04T10:00:00-08:00",
		}, {
			name:         "next-tuesday with wt with fallback",
			date:         "next-tuesday",
			wt:           "10:00:00",
			tz:           "America/Los_Angeles",
			startDate:    "2020-01-01",
			endDate:      "2021-02-01",
			fallbackWeek: "2021-01-04",
			useFallback:  true,
			expect:       "2021-01-05T10:00:00-08:00",
		}, {
			name:         "next-wednesday with wt with fallback",
			date:         "next-wednesday",
			wt:           "10:00:00",
			tz:           "America/Los_Angeles",
			startDate:    "2020-01-01",
			endDate:      "2021-02-01",
			fallbackWeek: "2021-01-04",
			useFallback:  true,
			expect:       "2021-01-06T10:00:00-08:00",
		}, {
			name:         "next-thursday with wt with fallback",
			date:         "next-thursday",
			wt:           "10:00:00",
			tz:           "America/Los_Angeles",
			startDate:    "2020-01-01",
			endDate:      "2021-02-01",
			fallbackWeek: "2021-01-04",
			useFallback:  true,
			expect:       "2021-01-07T10:00:00-08:00",
		}, {
			name:         "next-friday with wt with fallback",
			date:         "next-friday",
			wt:           "10:00:00",
			tz:           "America/Los_Angeles",
			startDate:    "2020-01-01",
			endDate:      "2021-02-01",
			fallbackWeek: "2021-01-04",
			useFallback:  true,
			expect:       "2021-01-08T10:00:00-08:00",
		}, {
			name:         "next-saturday with wt with fallback",
			date:         "next-saturday",
			wt:           "10:00:00",
			tz:           "America/Los_Angeles",
			startDate:    "2020-01-01",
			endDate:      "2021-02-01",
			fallbackWeek: "2021-01-04",
			useFallback:  true,
			expect:       "2021-01-09T10:00:00-08:00",
		},
		// NOTE: -07:00 because fallback week daylight savings time
		{
			name:         "use fallback dst",
			date:         "now",
			wt:           "10:00:00",
			tz:           "America/Los_Angeles",
			startDate:    "2021-06-01",
			endDate:      "2021-06-30",
			fallbackWeek: "2021-06-13",
			useFallback:  true,
			expect:       "2021-06-14T10:00:00-07:00",
		},
		// Check errors
		{
			name:        "check error date",
			date:        "asd",
			expectError: true,
		},
		{
			name:        "check error tz",
			date:        "now",
			tz:          "asd",
			expectError: true,
		},
		{
			name:        "check error time",
			date:        "now",
			wt:          "abc",
			expectError: true,
		},
		{
			name:         "check error start date",
			date:         "now",
			startDate:    "asd",
			endDate:      "2021-06-30",
			fallbackWeek: "2021-06-13",
			useFallback:  true,
			expectError:  true,
		},
		{
			name:         "check error end date",
			date:         "now",
			startDate:    "2020-01-01",
			endDate:      "asd",
			fallbackWeek: "2021-06-13",
			useFallback:  true,
			expectError:  true,
		},
		{
			name:         "check error fallback date",
			date:         "now",
			startDate:    "2020-01-01",
			endDate:      "2021-06-30",
			fallbackWeek: "asd",
			useFallback:  true,
			expectError:  true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			v, err := timeAtClock(tc.date, tc.wt, tc.tz, tc.startDate, tc.endDate, tc.fallbackWeek, tc.useFallback, cl)
			if err != nil && tc.expectError {
				// OK
				return
			} else if err != nil && !tc.expectError {
				t.Fatal(err)
			} else if err == nil && tc.expectError {
				t.Fatal("expected error, got none")
			}
			vf := v.Format(time.RFC3339)
			assert.Equal(t, tc.expect, vf)
		})
	}
}
