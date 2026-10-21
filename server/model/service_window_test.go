package model

import (
	"sort"
	"testing"

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
	assert.Equal(t, 13458933+21510411+21510411+21510411+21510411+13458933+16536528, totalService(levels[4]))
	assert.Equal(t, "2025-06-30", bestWeekOf(levels), "the busiest full-service week")

	// Reduced service on a day still counts as service on it.
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
	assert.Equal(t, 16527240+31430700+31430700+31430700+31430700+31555320+20879460, totalService(levels[4]))
	assert.Equal(t, "2025-09-08", bestWeekOf(levels), "the busiest full-service week")

	t.Run("every week partial", func(t *testing.T) {
		partial := []dmfr.FeedVersionServiceLevel{
			week(t, "2025-08-04", "2025-08-10", 0, 30602160, 30639300, 30639300, 30661200, 20477520, 15909060),
			week(t, "2025-08-11", "2025-08-17", 31320720, 0, 31334820, 31334820, 31374840, 21093000, 16452720),
		}
		assert.Equal(t, 16452720+31320720+0+31334820+31334820+31374840+21093000, totalService(partial[1]))
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
	assert.Equal(t, 0, totalService(levels[0]))
	assert.False(t, hasFullService(levels[0]))
	assert.Equal(t, 13424463+22059857+22059857+22059857+22059857+22126637+16538433, totalService(levels[3]))
	assert.Equal(t, "2025-09-15", bestWeekOf(levels), "the busiest full-service week")
}
