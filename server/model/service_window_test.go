package model

import (
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

// week is a feed version's service level for one week: service seconds Monday
// through Sunday.
func week(t *testing.T, start, end string, mon, tue, wed, thu, fri, sat, sun int) dmfr.FeedVersionServiceLevel {
	t.Helper()
	return dmfr.FeedVersionServiceLevel{
		StartDate: tt.NewDate(testDate(t, start)),
		EndDate:   tt.NewDate(testDate(t, end)),
		Monday:    mon,
		Tuesday:   tue,
		Wednesday: wed,
		Thursday:  thu,
		Friday:    fri,
		Saturday:  sat,
		Sunday:    sun,
	}
}

// bestWeekOf runs bestWeek over every week, in date order.
func bestWeekOf(levels []dmfr.FeedVersionServiceLevel) string {
	levels = append([]dmfr.FeedVersionServiceLevel(nil), levels...)
	sort.Slice(levels, func(i, j int) bool { return levels[i].StartDate.Val.Before(levels[j].StartDate.Val) })
	return bestWeek(levels, levels[0].StartDate.Val, levels[len(levels)-1].EndDate.Val).Format("2006-01-02")
}

// Caltrain weeks with partial service, weeks with reduced but nonzero service,
// and one week that far outranks the rest.
func TestBestWeek_Caltrain(t *testing.T) {
	levels := []dmfr.FeedVersionServiceLevel{
		week(t, "2025-08-18", "2025-08-24", 0, 0, 487260, 487260, 487260, 315840, 315840),
		week(t, "2025-08-25", "2025-08-31", 487260, 487260, 487260, 487260, 487260, 315840, 315840),
		week(t, "2025-09-01", "2025-09-07", 315840, 487260, 487260, 487260, 487260, 315840, 315840),
		week(t, "2025-09-08", "2025-11-23", 487260, 487260, 487260, 487260, 487260, 315840, 315840),
		week(t, "2025-06-30", "2025-07-06", 21510411, 21510411, 21510411, 21510411, 13458933, 16536528, 13458933),
		week(t, "2025-12-29", "2026-01-04", 22059857, 22059857, 22059857, 0, 0, 0, 0),
	}
	assert.Equal(t, "2025-06-30", bestWeekOf(levels), "the busiest full-service week")

	// A day with reduced service still counts as a day with service.
	assert.False(t, hasFullService(levels[0]))
	assert.False(t, hasFullService(levels[5]))
	assert.True(t, hasFullService(levels[1]))
	assert.True(t, hasFullService(levels[2]))
	assert.True(t, hasFullService(levels[4]))
}

// MBTA feed version f1aa4431c87ded2609b3eb069dddbd4fe2a8aba9, whose first week
// has no service Monday through Wednesday and was once chosen as its fallback.
func TestBestWeek_MBTA(t *testing.T) {
	levels := []dmfr.FeedVersionServiceLevel{
		week(t, "2025-08-04", "2025-08-10", 0, 0, 0, 30639300, 30661200, 20477520, 15909060),
		week(t, "2025-08-11", "2025-08-17", 31320720, 31334820, 31334820, 31334820, 31374840, 21093000, 16452720),
		week(t, "2025-08-18", "2025-08-24", 30618420, 30618420, 30618420, 30618420, 30662040, 21130320, 0),
		week(t, "2025-09-01", "2025-09-07", 16566840, 31430700, 31430700, 31430700, 31555320, 20879460, 16527240),
		week(t, "2025-09-08", "2025-09-14", 31430700, 31430700, 31430700, 31430700, 31555320, 20879460, 16527240),
	}
	assert.Equal(t, "2025-09-08", bestWeekOf(levels), "the busiest full-service week")

	t.Run("every week partial", func(t *testing.T) {
		partial := []dmfr.FeedVersionServiceLevel{
			week(t, "2025-08-04", "2025-08-10", 0, 30602160, 30639300, 30639300, 30661200, 20477520, 15909060),
			week(t, "2025-08-11", "2025-08-17", 31320720, 0, 31334820, 31334820, 31374840, 21093000, 16452720),
		}
		assert.Equal(t, "2025-08-11", bestWeekOf(partial), "the busiest week")
	})
}

// RTD weeks including one with no service at all.
func TestBestWeek_RTD(t *testing.T) {
	levels := []dmfr.FeedVersionServiceLevel{
		week(t, "2025-05-19", "2025-05-25", 0, 0, 0, 0, 0, 0, 0),
		week(t, "2025-05-26", "2025-06-01", 13458933, 21510411, 21510411, 21510411, 21595911, 16536528, 13458933),
		week(t, "2025-06-30", "2025-07-06", 21510411, 21510411, 21510411, 21510411, 13458933, 16536528, 13458933),
		week(t, "2025-09-15", "2025-11-23", 22059857, 22059857, 22059857, 22059857, 22126637, 16538433, 13424463),
	}
	assert.False(t, hasFullService(levels[0]))
	assert.Equal(t, "2025-09-15", bestWeekOf(levels), "the busiest full-service week")
}

func TestNewFeedServiceWindow(t *testing.T) {
	levels := []dmfr.FeedVersionServiceLevel{
		week(t, "2025-08-11", "2025-08-17", 1, 1, 1, 1, 1, 1, 1),
	}
	fvsw := dmfr.FeedVersionServiceWindow{DefaultTimezone: tt.NewString("America/Los_Angeles")}

	t.Run("in the feed version's default timezone", func(t *testing.T) {
		var asked string
		w, err := NewFeedServiceWindow(fvsw, levels, time.Time{}, func(name string) (*time.Location, error) {
			asked = name
			return time.LoadLocation(name)
		})
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, "America/Los_Angeles", asked)
		assert.Equal(t, "America/Los_Angeles", w.Location.String())
		assert.Equal(t, "2025-08-11", w.FallbackWeek.Format("2006-01-02"))
	})

	t.Run("a timezone that doesn't load is an error", func(t *testing.T) {
		_, err := NewFeedServiceWindow(fvsw, levels, time.Time{}, func(name string) (*time.Location, error) {
			return nil, errors.New("no such timezone")
		})
		assert.ErrorContains(t, err, "no such timezone")
	})

	t.Run("no location is an error", func(t *testing.T) {
		_, err := NewFeedServiceWindow(fvsw, levels, time.Time{}, func(name string) (*time.Location, error) {
			return nil, nil
		})
		assert.Error(t, err)
	})
}
