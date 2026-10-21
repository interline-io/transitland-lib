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

func mustResolveTripDates(t *testing.T, where *model.TripFilter, fvsw *model.ServiceWindow) []model.TripDate {
	t.Helper()
	dates, err := model.ResolveTripDates(where, fvsw)
	if err != nil {
		t.Fatal(err)
	}
	return dates
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

// The callers build one filter and hand it to tripSelect once per feed version,
// so resolving a date must not write back into it: the next feed version would
// then resolve an already-relocated date against its own window. End to end this
// is nearly invisible, because relocation preserves the weekday and a route on a
// weekday calendar returns the same trips either way, so assert it directly.
func TestTripSelectDoesNotModifyFilter(t *testing.T) {
	requested := testDates(t, "2030-01-02")[0]
	where := &model.TripFilter{
		ServiceDate:      ptr(*requested),
		UseServiceWindow: ptr(true),
	}
	// A window that excludes the requested date, so it is certain to relocate.
	fvsw := testServiceWindow(t)
	for i := 0; i < 2; i++ {
		if _, _, err := tripSelect(nil, nil, nil, false, nil, where, fvsw); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, requested.Val.Format("2006-01-02"), where.ServiceDate.Val.Format("2006-01-02"),
			"filter was rewritten on pass %d", i)
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
		dates := mustResolveTripDates(t,
			&model.TripFilter{ServiceDates: testDates(t, "2030-01-02"), UseServiceWindow: ptr(true)},
			testServiceWindow(t),
		)
		ent := trip("2026-05-13")
		expandTripServiceDates([]*model.Trip{ent}, dates)
		assert.Equal(t, []string{"2030-01-02"}, got(ent))
	})

	t.Run("expands one database date into every date it stands for", func(t *testing.T) {
		dates := mustResolveTripDates(t,
			&model.TripFilter{ServiceDates: testDates(t, "2030-01-09", "2030-01-02"), UseServiceWindow: ptr(true)},
			testServiceWindow(t),
		)
		ent := trip("2026-05-13")
		expandTripServiceDates([]*model.Trip{ent}, dates)
		assert.Equal(t, []string{"2030-01-02", "2030-01-09"}, got(ent))
	})

	t.Run("sorts across database dates", func(t *testing.T) {
		dates := mustResolveTripDates(t, &model.TripFilter{ServiceDates: testDates(t, "2026-05-13", "2026-05-11")}, nil)
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
		dates := mustResolveTripDates(t, &model.TripFilter{ServiceDates: testDates(t, "2026-05-13")}, nil)
		ent := trip("")
		expandTripServiceDates([]*model.Trip{ent}, dates)
		assert.Nil(t, got(ent))
	})
}
