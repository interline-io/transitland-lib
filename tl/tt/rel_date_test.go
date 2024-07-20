package tt

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRelativeDate(t *testing.T) {
	defaultUtc := "2024-07-16T01:30:00 UTC"
	defaultLocal := "2024-07-15T10:00:00 America/Los_Angeles"
	tcs := []struct {
		name        string
		dateLabel   string
		whenLocal   string
		expect      string
		expectError bool
	}{
		{
			name:      "empty (now, default UTC)",
			whenLocal: defaultUtc,
			expect:    "2024-07-16T01:30:00Z",
		},
		{
			name:      "date specified",
			dateLabel: "2024-07-10",
			whenLocal: defaultUtc,
			expect:    "2024-07-10T01:30:00Z",
		},
		{
			name:      "date specified 2",
			dateLabel: "2024-07-10",
			whenLocal: "2024-07-16T10:00:00 UTC",
			expect:    "2024-07-10T10:00:00Z",
		},
		{
			name:      "date with tz",
			dateLabel: "2024-07-16",
			whenLocal: defaultLocal,
			expect:    "2024-07-16T10:00:00-07:00",
		},
		{
			name:      "now with tz",
			dateLabel: "now",
			whenLocal: defaultLocal,
			expect:    "2024-07-15T10:00:00-07:00",
		},
		// same dow allowed
		{
			name:      "monday",
			dateLabel: "monday",
			whenLocal: "2024-07-15T10:00:00 America/Los_Angeles",
			expect:    "2024-07-15T10:00:00-07:00",
		},
		{
			name:      "friday",
			dateLabel: "friday",
			whenLocal: "2024-07-19T10:00:00 America/Los_Angeles",
			expect:    "2024-07-19T10:00:00-07:00",
		},
		// one day before
		{
			name:      "next-sunday",
			dateLabel: "next-sunday",
			whenLocal: "2024-07-20T10:00:00 America/Los_Angeles",
			expect:    "2024-07-21T10:00:00-07:00",
		},
		{
			name:      "next-monday",
			dateLabel: "next-monday",
			whenLocal: "2024-07-21T10:00:00 America/Los_Angeles",
			expect:    "2024-07-22T10:00:00-07:00",
		},
		{
			name:      "next-tuesday",
			dateLabel: "next-tuesday",
			whenLocal: "2024-07-15T10:00:00 America/Los_Angeles",
			expect:    "2024-07-16T10:00:00-07:00",
		},
		{
			name:      "next-wednesday",
			dateLabel: "next-wednesday",
			whenLocal: "2024-07-16T10:00:00 America/Los_Angeles",
			expect:    "2024-07-17T10:00:00-07:00",
		},
		{
			name:      "next-thursday",
			dateLabel: "next-thursday",
			whenLocal: "2024-07-17T10:00:00 America/Los_Angeles",
			expect:    "2024-07-18T10:00:00-07:00",
		},
		{
			name:      "next-friday",
			dateLabel: "next-friday",
			whenLocal: "2024-07-18T10:00:00 America/Los_Angeles",
			expect:    "2024-07-19T10:00:00-07:00",
		},
		{
			name:      "next-saturday",
			dateLabel: "next-saturday",
			whenLocal: "2024-07-19T10:00:00 America/Los_Angeles",
			expect:    "2024-07-20T10:00:00-07:00",
		},
		// same day of week
		{
			name:      "next-sunday same dow",
			dateLabel: "next-sunday",
			whenLocal: "2024-07-14T10:00:00 America/Los_Angeles",
			expect:    "2024-07-21T10:00:00-07:00",
		},
		{
			name:      "next-monday same dow",
			dateLabel: "next-monday",
			whenLocal: "2024-07-15T10:00:00 America/Los_Angeles",
			expect:    "2024-07-22T10:00:00-07:00",
		},
		{
			name:      "next-tuesday same dow",
			dateLabel: "next-tuesday",
			whenLocal: "2024-07-16T10:00:00 America/Los_Angeles",
			expect:    "2024-07-23T10:00:00-07:00",
		},
		{
			name:      "next-wednesday same dow",
			dateLabel: "next-wednesday",
			whenLocal: "2024-07-17T10:00:00 America/Los_Angeles",
			expect:    "2024-07-24T10:00:00-07:00",
		},
		{
			name:      "next-thursday same dow",
			dateLabel: "next-thursday",
			whenLocal: "2024-07-18T10:00:00 America/Los_Angeles",
			expect:    "2024-07-25T10:00:00-07:00",
		},
		{
			name:      "next-friday same dow",
			dateLabel: "next-friday",
			whenLocal: "2024-07-19T10:00:00 America/Los_Angeles",
			expect:    "2024-07-26T10:00:00-07:00",
		},
		{
			name:      "next-saturday same dow",
			dateLabel: "next-saturday",
			whenLocal: "2024-07-20T10:00:00 America/Los_Angeles",
			expect:    "2024-07-27T10:00:00-07:00",
		},
		// Errors
		{
			name:        "check error date",
			dateLabel:   "asd",
			whenLocal:   defaultLocal,
			expectError: true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			w := tc.whenLocal
			ws := strings.Split(w, " ")
			wd, err := time.Parse("2006-01-02T15:04:05", ws[0])
			if err != nil {
				t.Fatal(err)
			}
			loc, err := time.LoadLocation(ws[1])
			if err != nil {
				t.Fatal(err)
			}
			currentTime := time.Date(wd.Year(), wd.Month(), wd.Day(), wd.Hour(), wd.Minute(), wd.Second(), 0, loc)
			relDate, err := RelativeDate(currentTime, tc.dateLabel)
			// fmt.Println("currentTime:", currentTime, "label:", tc.dateLabel, "relDate:", relDate, "err:", err)
			if err != nil && tc.expectError {
				// OK
				return
			} else if err != nil && !tc.expectError {
				t.Fatal(err)
			} else if err == nil && tc.expectError {
				t.Fatal("expected error, got none")
			}
			vf := relDate.Format(time.RFC3339)
			assert.Equal(t, tc.expect, vf)
		})
	}
}

func TestFallbackDate(t *testing.T) {
	defaultUtc := "2024-07-16T01:30:00 UTC"
	tcs := []struct {
		name         string
		dateLabel    string
		whenLocal    string
		startDate    string
		endDate      string
		fallbackWeek string
		expect       string
		expectError  bool
	}{
		// Fallback tests
		{
			name:         "use fallback, same as start date",
			whenLocal:    "2024-07-15T10:00:00 America/Los_Angeles",
			startDate:    "2024-07-15",
			endDate:      "2024-07-21",
			fallbackWeek: "2021-01-04",
			expect:       "2024-07-15T10:00:00-07:00",
		},
		{
			name:         "use fallback, same as end date",
			whenLocal:    "2024-07-21T10:00:00 America/Los_Angeles",
			startDate:    "2024-07-15",
			endDate:      "2024-07-21",
			fallbackWeek: "2021-01-04",
			expect:       "2024-07-21T10:00:00-07:00",
		},
		// NOTE: some are -08:00 because fallback week standard time
		{
			name:         "use fallback, one before start",
			whenLocal:    "2024-07-14T10:00:00 America/Los_Angeles",
			startDate:    "2024-07-15",
			endDate:      "2024-07-21",
			fallbackWeek: "2021-01-04",
			expect:       "2021-01-10T10:00:00-08:00",
		},
		{
			name:         "use fallback, one day after end",
			whenLocal:    "2024-07-22T10:00:00 America/Los_Angeles",
			startDate:    "2024-07-15",
			endDate:      "2024-07-21",
			fallbackWeek: "2021-01-04",
			expect:       "2021-01-04T10:00:00-08:00",
		},
		{
			name:         "use fallback",
			whenLocal:    "2024-07-21T10:00:00 America/Los_Angeles",
			startDate:    "2020-01-01",
			endDate:      "2021-02-01",
			fallbackWeek: "2021-01-04",
			expect:       "2021-01-10T10:00:00-08:00",
		},
		// NOTE: -07:00 because fallback week daylight savings time
		{
			name:         "use fallback dst",
			whenLocal:    "2024-07-21T10:00:00 America/Los_Angeles",
			startDate:    "2021-06-01",
			endDate:      "2021-06-30",
			fallbackWeek: "2021-06-13",
			expect:       "2021-06-13T10:00:00-07:00",
		},
		// Errors
		{
			name:         "date wrong order",
			whenLocal:    "2024-07-21T10:00:00 America/Los_Angeles",
			endDate:      "2021-06-01",
			startDate:    "2021-06-30",
			fallbackWeek: "2021-06-13",
			expectError:  true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			w := defaultUtc
			if tc.whenLocal != "" {
				w = tc.whenLocal
			}
			ws := strings.Split(w, " ")
			wd, err := time.Parse("2006-01-02T15:04:05", ws[0])
			if err != nil {
				t.Fatal(err)
			}
			loc, err := time.LoadLocation(ws[1])
			if err != nil {
				t.Fatal(err)
			}
			currentTime := time.Date(wd.Year(), wd.Month(), wd.Day(), wd.Hour(), wd.Minute(), wd.Second(), 0, loc)
			// Parse others
			// Use as midnight in currentTime tz
			startTime, err := time.Parse("2006-01-02", tc.startDate)
			if err != nil {
				t.Fatal(err)
			}
			endTime, err := time.Parse("2006-01-02", tc.endDate)
			if err != nil {
				t.Fatal(err)
			}
			fallbackWeek, err := time.Parse("2006-01-02", tc.fallbackWeek)
			if err != nil {
				t.Fatal(err)
			}
			startTime = midnight(startTime, loc)
			endTime = midnight(endTime, loc)
			fallbackWeek = midnight(fallbackWeek, loc)

			relDate, ok, err := FallbackDate(currentTime, startTime, endTime, fallbackWeek)
			_ = ok
			if err != nil && tc.expectError {
				// OK
				return
			} else if err != nil && !tc.expectError {
				t.Fatal(err)
			} else if err == nil && tc.expectError {
				t.Fatal("expected error, got none")
			}
			vf := relDate.Format(time.RFC3339)
			assert.Equal(t, tc.expect, vf)
			assert.Equal(t, currentTime.Weekday(), relDate.Weekday())
		})
	}
}
