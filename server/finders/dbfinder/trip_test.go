package dbfinder

import (
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/server/model"
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
		out = append(out, ptr(tt.NewDate(testDate(t, d))))
	}
	return out
}

// A window covering one week, whose fallback week is the same Monday-Sunday.
func testServiceWindow(t *testing.T) *model.ServiceWindow {
	t.Helper()
	return &model.ServiceWindow{
		NowLocal:     testDate(t, "2026-05-13"),
		StartDate:    testDate(t, "2026-05-11"),
		EndDate:      testDate(t, "2026-05-17"),
		FallbackWeek: testDate(t, "2026-05-11"),
	}
}

func TestResolveTripDates(t *testing.T) {
	fmtPairs := func(dates []tripDate) []string {
		var out []string
		for _, d := range dates {
			s := d.query.Format("2006-01-02") + " ->"
			for _, r := range d.report {
				s += " " + r.Format("2006-01-02")
			}
			out = append(out, s)
		}
		return out
	}
	tcs := []struct {
		name   string
		where  *model.TripFilter
		fvsw   *model.ServiceWindow
		expect []string
	}{
		{"no filter", nil, nil, nil},
		{"no dates", &model.TripFilter{}, nil, nil},
		{
			// service_dates are matched as given: no lookback, so nothing
			// outside the requested set comes back.
			"service_dates are exact",
			&model.TripFilter{ServiceDates: testDates(t, "2026-05-13", "2026-05-12")}, nil,
			[]string{"2026-05-12 -> 2026-05-12", "2026-05-13 -> 2026-05-13"},
		},
		{
			"dates reach back for after-midnight departures",
			&model.TripFilter{Dates: testDates(t, "2026-05-13")}, nil,
			[]string{"2026-05-12 -> 2026-05-12", "2026-05-13 -> 2026-05-13"},
		},
		{
			"dates take precedence over service_dates",
			&model.TripFilter{
				Dates:        testDates(t, "2026-05-13"),
				ServiceDates: testDates(t, "2026-01-01"),
			}, nil,
			[]string{"2026-05-12 -> 2026-05-12", "2026-05-13 -> 2026-05-13"},
		},
		{
			"use_service_window without a window is inert",
			&model.TripFilter{ServiceDates: testDates(t, "2030-01-01"), UseServiceWindow: ptr(true)}, nil,
			[]string{"2030-01-01 -> 2030-01-01"},
		},
		{
			"dates inside the window are not relocated",
			&model.TripFilter{ServiceDates: testDates(t, "2026-05-13"), UseServiceWindow: ptr(true)}, testServiceWindow(t),
			[]string{"2026-05-13 -> 2026-05-13"},
		},
		{
			// 2030-01-02 is a Wednesday, relocating to the fallback week's
			// Wednesday; the caller still gets back the date it asked about.
			"dates outside the window relocate but report as requested",
			&model.TripFilter{ServiceDates: testDates(t, "2030-01-02"), UseServiceWindow: ptr(true)}, testServiceWindow(t),
			[]string{"2026-05-13 -> 2030-01-02"},
		},
		{
			// Two Wednesdays collapse onto one fallback day, which must expand
			// back to both rather than being queried or reported twice.
			"same weekday collapses to one query date",
			&model.TripFilter{ServiceDates: testDates(t, "2030-01-02", "2030-01-09"), UseServiceWindow: ptr(true)}, testServiceWindow(t),
			[]string{"2026-05-13 -> 2030-01-02 2030-01-09"},
		},
		{
			// The lookback runs before relocation, so the preceding day maps by
			// its own weekday: Tuesday 2030-01-01 to the fallback Tuesday.
			"lookback relocates independently of the requested date",
			&model.TripFilter{Dates: testDates(t, "2030-01-02"), UseServiceWindow: ptr(true)}, testServiceWindow(t),
			[]string{"2026-05-12 -> 2030-01-01", "2026-05-13 -> 2030-01-02"},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, fmtPairs(resolveTripDates(tc.where, tc.fvsw)))
		})
	}
}

func TestExpandTripServiceDates(t *testing.T) {
	trip := func(agg string) *model.Trip {
		ent := &model.Trip{}
		if agg != "" {
			ent.ServiceDatesAgg.Set(agg)
		}
		return ent
	}
	got := func(ent *model.Trip) []string {
		var out []string
		for _, d := range ent.ServiceDates {
			out = append(out, d.Val.Format("2006-01-02"))
		}
		return out
	}

	t.Run("relabels database dates as requested dates", func(t *testing.T) {
		dates := resolveTripDates(
			&model.TripFilter{ServiceDates: testDates(t, "2030-01-02"), UseServiceWindow: ptr(true)},
			testServiceWindow(t),
		)
		ent := trip("2026-05-13")
		expandTripServiceDates([]*model.Trip{ent}, dates)
		assert.Equal(t, []string{"2030-01-02"}, got(ent))
	})

	t.Run("expands one database date into every date it stands for", func(t *testing.T) {
		dates := resolveTripDates(
			&model.TripFilter{ServiceDates: testDates(t, "2030-01-09", "2030-01-02"), UseServiceWindow: ptr(true)},
			testServiceWindow(t),
		)
		ent := trip("2026-05-13")
		expandTripServiceDates([]*model.Trip{ent}, dates)
		assert.Equal(t, []string{"2030-01-02", "2030-01-09"}, got(ent))
	})

	t.Run("sorts across database dates", func(t *testing.T) {
		dates := resolveTripDates(&model.TripFilter{ServiceDates: testDates(t, "2026-05-13", "2026-05-11")}, nil)
		ent := trip("2026-05-13,2026-05-11")
		expandTripServiceDates([]*model.Trip{ent}, dates)
		assert.Equal(t, []string{"2026-05-11", "2026-05-13"}, got(ent))
	})

	t.Run("no dates requested", func(t *testing.T) {
		ent := trip("2026-05-13")
		expandTripServiceDates([]*model.Trip{ent}, nil)
		assert.Nil(t, got(ent))
	})

	t.Run("trip matched no dates", func(t *testing.T) {
		dates := resolveTripDates(&model.TripFilter{ServiceDates: testDates(t, "2026-05-13")}, nil)
		ent := trip("")
		expandTripServiceDates([]*model.Trip{ent}, dates)
		assert.Nil(t, got(ent))
	})
}
