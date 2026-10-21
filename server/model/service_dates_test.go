package model

import (
	"slices"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

func testDate(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func testDates(t *testing.T, dates ...string) []*tt.Date {
	t.Helper()
	var out []*tt.Date
	for _, d := range dates {
		v := tt.NewDate(testDate(t, d))
		out = append(out, &v)
	}
	return out
}

// A window covering one week, whose fallback week is the same Monday-Sunday.
func testServiceWindow(t *testing.T) *ServiceWindow {
	t.Helper()
	return &ServiceWindow{
		NowLocal: testDate(t, "2026-05-13"),
		FeedServiceWindow: FeedServiceWindow{
			StartDate:    testDate(t, "2026-05-11"),
			EndDate:      testDate(t, "2026-05-17"),
			FallbackWeek: testDate(t, "2026-05-11"),
		},
	}
}

func mustResolveTripDates(t *testing.T, where *TripFilter, fvsw *ServiceWindow) []TripDate {
	t.Helper()
	dates, err := ResolveTripDates(where, fvsw)
	if err != nil {
		t.Fatal(err)
	}
	return dates
}

func TestCalendarServiceDates(t *testing.T) {
	parse := func(dates ...string) []time.Time {
		var out []time.Time
		for _, d := range dates {
			out = append(out, testDate(t, d))
		}
		return out
	}
	format := func(dates []time.Time) []string {
		var out []string
		for _, d := range dates {
			out = append(out, d.Format("2006-01-02"))
		}
		return out
	}
	tcs := []struct {
		name     string
		dates    []string
		lookback int
		expect   []string
	}{
		{
			"no lookback sorts and deduplicates",
			[]string{"2026-05-11", "2026-05-10", "2026-05-11"}, 0,
			[]string{"2026-05-10", "2026-05-11"},
		},
		{
			"single date reaches back one day",
			[]string{"2026-05-10"}, 1,
			[]string{"2026-05-09", "2026-05-10"},
		},
		{
			// A week is 8 service dates, not 14.
			"consecutive dates overlap",
			[]string{"2026-05-10", "2026-05-11", "2026-05-12", "2026-05-13", "2026-05-14", "2026-05-15", "2026-05-16"}, 1,
			[]string{"2026-05-09", "2026-05-10", "2026-05-11", "2026-05-12", "2026-05-13", "2026-05-14", "2026-05-15", "2026-05-16"},
		},
		{
			"gaps are not filled in",
			[]string{"2026-05-10", "2026-05-11", "2026-05-15"}, 1,
			[]string{"2026-05-09", "2026-05-10", "2026-05-11", "2026-05-14", "2026-05-15"},
		},
		{
			"lookback covers multi-day trips",
			[]string{"2026-05-10"}, 3,
			[]string{"2026-05-07", "2026-05-08", "2026-05-09", "2026-05-10"},
		},
		{
			"crosses a month boundary",
			[]string{"2026-06-01"}, 1,
			[]string{"2026-05-31", "2026-06-01"},
		},
		{"no dates", nil, 1, nil},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, format(calendarServiceDates(parse(tc.dates...), tc.lookback)))
		})
	}
}

// A date's local noon exists even on a day whose midnight a DST change skips:
// America/Santiago jumped from midnight to 01:00 on 2024-09-08.
func TestTzTruncate(t *testing.T) {
	santiago, err := time.LoadLocation("America/Santiago")
	if err != nil {
		t.Fatal(err)
	}
	tcs := []struct {
		name   string
		in     time.Time
		expect string
	}{
		{"a date scalar on the day DST starts", time.Date(2024, 9, 8, 0, 0, 0, 0, time.UTC), "2024-09-08"},
		{"late on the evening before", time.Date(2024, 9, 7, 23, 30, 0, 0, santiago), "2024-09-07"},
		{"early on the morning after", time.Date(2024, 9, 9, 0, 30, 0, 0, santiago), "2024-09-09"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := tzTruncate(tc.in, santiago)
			assert.Equal(t, tc.expect, got.String())
			assert.Equal(t, 12, got.Val.Hour())
		})
	}
}

func TestMapIntoServiceWindow(t *testing.T) {
	// A feed version's window carries local midnight, while a date scalar
	// parses as midnight UTC. West of UTC the requested day then sorts before
	// the window's first day; east of it, after the window's last day. Either
	// way an in-window boundary day would be relocated.
	for _, tz := range []string{"America/Los_Angeles", "Asia/Kolkata", "UTC"} {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			t.Fatal(err)
		}
		day := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, loc) }
		fvsw := &ServiceWindow{
			NowLocal: time.Date(2026, 5, 13, 9, 30, 0, 0, loc),
			FeedServiceWindow: FeedServiceWindow{
				StartDate:    day(2026, 5, 11), // Monday
				EndDate:      day(2026, 5, 17), // Sunday
				FallbackWeek: day(2026, 3, 2),  // Monday
			},
		}
		tcs := []struct{ name, in, expect string }{
			{"first day of the window is inside it", "2026-05-11", "2026-05-11"},
			{"last day of the window is inside it", "2026-05-17", "2026-05-17"},
			{"mid-window is unchanged", "2026-05-13", "2026-05-13"},
			// 2026-01-05 is a Monday, 2027-01-09 a Saturday.
			{"before the window takes the fallback weekday", "2026-01-05", "2026-03-02"},
			{"after the window takes the fallback weekday", "2027-01-09", "2026-03-07"},
		}
		for _, tc := range tcs {
			t.Run(tz+"/"+tc.name, func(t *testing.T) {
				got := mapIntoServiceWindow(testDate(t, tc.in), fvsw)
				assert.Equal(t, tc.expect, got.Val.Format("2006-01-02"))
			})
		}
	}
}

// A window ending on a day whose midnight a DST change skips still holds that
// day: America/Santiago jumped from midnight to 01:00 on 2024-09-08.
func TestMapIntoServiceWindow_MidnightDST(t *testing.T) {
	loc, err := time.LoadLocation("America/Santiago")
	if err != nil {
		t.Fatal(err)
	}
	fvsw := &ServiceWindow{
		NowLocal: time.Date(2024, 9, 8, 9, 30, 0, 0, loc),
		FeedServiceWindow: FeedServiceWindow{
			StartDate:    time.Date(2024, 9, 2, 12, 0, 0, 0, loc), // Monday
			EndDate:      time.Date(2024, 9, 8, 12, 0, 0, 0, loc), // Sunday
			FallbackWeek: time.Date(2024, 3, 4, 12, 0, 0, 0, loc), // Monday
		},
	}
	got := mapIntoServiceWindow(testDate(t, "2024-09-08"), fvsw)
	assert.Equal(t, "2024-09-08", got.String())
}

func TestResolveTripDates(t *testing.T) {
	fmtPairs := func(dates []TripDate) []string {
		var out []string
		for _, d := range dates {
			s := d.Query.Format("2006-01-02") + " ->"
			for _, r := range d.Report {
				s += " " + r.Format("2006-01-02")
			}
			if d.Single {
				s += " (single)"
			}
			out = append(out, s)
		}
		return out
	}
	tcs := []struct {
		name   string
		where  *TripFilter
		fvsw   *ServiceWindow
		expect []string
	}{
		{"no filter", nil, nil, nil},
		{"no dates", &TripFilter{}, nil, nil},
		{
			// service_dates are matched as given: no lookback, so nothing
			// outside the requested set comes back.
			"service_dates are exact",
			&TripFilter{ServiceDates: testDates(t, "2026-05-13", "2026-05-12")}, nil,
			[]string{"2026-05-12 -> 2026-05-12", "2026-05-13 -> 2026-05-13"},
		},
		{
			"dates reach back for after-midnight departures",
			&TripFilter{Dates: testDates(t, "2026-05-13")}, nil,
			[]string{"2026-05-12 -> 2026-05-12", "2026-05-13 -> 2026-05-13"},
		},
		{
			"dates take precedence over service_dates",
			&TripFilter{
				Dates:        testDates(t, "2026-05-13"),
				ServiceDates: testDates(t, "2026-01-01"),
			}, nil,
			[]string{"2026-05-12 -> 2026-05-12", "2026-05-13 -> 2026-05-13"},
		},
		{
			"service_dates take precedence over service_date",
			&TripFilter{
				ServiceDates: testDates(t, "2026-05-13"),
				ServiceDate:  testDates(t, "2026-01-01")[0],
			}, nil,
			[]string{"2026-05-13 -> 2026-05-13"},
		},
		{
			"a single service_date",
			&TripFilter{ServiceDate: testDates(t, "2026-05-13")[0]}, nil,
			[]string{"2026-05-13 -> 2026-05-13 (single)"},
		},
		{
			// 2030-01-02 is a Wednesday.
			"a single service_date outside the window relocates but reports as requested",
			&TripFilter{ServiceDate: testDates(t, "2030-01-02")[0], UseServiceWindow: ptrTo(true)}, testServiceWindow(t),
			[]string{"2026-05-13 -> 2030-01-02 (single)"},
		},
		{
			"use_service_window without a window is inert",
			&TripFilter{ServiceDates: testDates(t, "2030-01-01"), UseServiceWindow: ptrTo(true)}, nil,
			[]string{"2030-01-01 -> 2030-01-01"},
		},
		{
			"dates inside the window are not relocated",
			&TripFilter{ServiceDates: testDates(t, "2026-05-13"), UseServiceWindow: ptrTo(true)}, testServiceWindow(t),
			[]string{"2026-05-13 -> 2026-05-13"},
		},
		{
			// 2030-01-02 is a Wednesday, relocating to the fallback week's
			// Wednesday; the caller still gets back the date it asked about.
			"dates outside the window relocate but report as requested",
			&TripFilter{ServiceDates: testDates(t, "2030-01-02"), UseServiceWindow: ptrTo(true)}, testServiceWindow(t),
			[]string{"2026-05-13 -> 2030-01-02"},
		},
		{
			// Two Wednesdays collapse onto one fallback day, which must expand
			// back to both rather than being queried or reported twice.
			"same weekday collapses to one query date",
			&TripFilter{ServiceDates: testDates(t, "2030-01-02", "2030-01-09"), UseServiceWindow: ptrTo(true)}, testServiceWindow(t),
			[]string{"2026-05-13 -> 2030-01-02 2030-01-09"},
		},
		{
			// The lookback runs before relocation, so the preceding day maps by
			// its own weekday: Tuesday 2030-01-01 to the fallback Tuesday.
			"lookback relocates independently of the requested date",
			&TripFilter{Dates: testDates(t, "2030-01-02"), UseServiceWindow: ptrTo(true)}, testServiceWindow(t),
			[]string{"2026-05-12 -> 2030-01-01", "2026-05-13 -> 2030-01-02"},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, fmtPairs(mustResolveTripDates(t, tc.where, tc.fvsw)))
		})
	}
}

func TestSetTripRunDates(t *testing.T) {
	format := func(dates []tt.Date) []string {
		var out []string
		for _, d := range dates {
			out = append(out, d.Val.Format("2006-01-02"))
		}
		return out
	}
	serviceDates := func(trip *Trip) []string {
		var out []string
		for _, d := range trip.ServiceDates {
			out = append(out, d.Val.Format("2006-01-02"))
		}
		return out
	}
	// ranOn reports a trip as running on the given query dates.
	ranOn := func(dates []TripDate, days ...string) func(int) bool {
		return func(i int) bool {
			return slices.Contains(days, dates[i].Query.Format("2006-01-02"))
		}
	}

	t.Run("relabels query dates as requested dates", func(t *testing.T) {
		dates := mustResolveTripDates(t, &TripFilter{ServiceDates: testDates(t, "2030-01-02"), UseServiceWindow: ptrTo(true)}, testServiceWindow(t))
		trip := &Trip{}
		SetTripRunDates(trip, dates, ranOn(dates, "2026-05-13"))
		assert.Equal(t, []string{"2030-01-02"}, serviceDates(trip))
		assert.Equal(t, []string{"2030-01-02"}, format(trip.RunDates))
	})

	t.Run("expands one query date into every date it stands for, sorted", func(t *testing.T) {
		dates := mustResolveTripDates(t, &TripFilter{ServiceDates: testDates(t, "2030-01-09", "2030-01-02"), UseServiceWindow: ptrTo(true)}, testServiceWindow(t))
		trip := &Trip{}
		SetTripRunDates(trip, dates, ranOn(dates, "2026-05-13"))
		assert.Equal(t, []string{"2030-01-02", "2030-01-09"}, serviceDates(trip))
	})

	t.Run("only the dates the trip ran", func(t *testing.T) {
		dates := mustResolveTripDates(t, &TripFilter{Dates: testDates(t, "2026-05-13")}, nil)
		trip := &Trip{}
		SetTripRunDates(trip, dates, ranOn(dates, "2026-05-12"))
		assert.Equal(t, []string{"2026-05-12"}, serviceDates(trip))
	})

	t.Run("a single service_date is a run but not a service date", func(t *testing.T) {
		dates := mustResolveTripDates(t, &TripFilter{ServiceDate: testDates(t, "2026-05-13")[0]}, nil)
		trip := &Trip{}
		SetTripRunDates(trip, dates, ranOn(dates))
		assert.Nil(t, serviceDates(trip))
		assert.Equal(t, []string{"2026-05-13"}, format(trip.RunDates))
	})

	t.Run("no dates requested", func(t *testing.T) {
		trip := &Trip{}
		SetTripRunDates(trip, nil, ranOn(nil, "2026-05-13"))
		assert.Nil(t, serviceDates(trip))
		assert.Nil(t, trip.RunDates)
	})
}
