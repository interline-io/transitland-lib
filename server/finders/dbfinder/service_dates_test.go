package dbfinder

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCalendarServiceDates(t *testing.T) {
	parse := func(dates ...string) []time.Time {
		var out []time.Time
		for _, d := range dates {
			v, err := time.Parse("2006-01-02", d)
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, v)
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
