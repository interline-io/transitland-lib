package rtfinder

import (
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

func TestTripRuns(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	date := func(s string) tt.Date {
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			t.Fatal(err)
		}
		return tt.NewDate(d)
	}
	at := func(s string) time.Time {
		v, err := time.ParseInLocation("2006-01-02 15:04", s, la)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	// A trip running 4 pm to 5 pm, as the runs of the given dates.
	runs := func(dates ...string) tripRuns {
		r := tripRuns{span: func(d tt.Date) (time.Time, time.Time, bool) {
			day := serviceDayStart(d, la)
			return day.Add(16 * time.Hour), day.Add(17 * time.Hour), true
		}}
		for _, d := range dates {
			r.dates = append(r.dates, date(d))
		}
		return r
	}
	dated := func(d string) *pb.TripDescriptor {
		return &pb.TripDescriptor{TripId: ptr("t"), StartDate: ptr(d)}
	}
	undated := &pb.TripDescriptor{TripId: ptr("t")}
	period := func(start, end string) []*pb.TimeRange {
		return []*pb.TimeRange{{Start: ptr(uint64(at(start).Unix())), End: ptr(uint64(at(end).Unix()))}}
	}

	t.Run("a dated message describes its run only, whenever it is read", func(t *testing.T) {
		now := at("2018-05-30 15:00")
		assert.True(t, runs("2018-05-30").currentFor(dated("20180530"), now))
		assert.False(t, runs("2018-05-30").currentFor(dated("20180531"), now))
		assert.True(t, runs("2018-06-30").currentFor(dated("20180630"), now))
		assert.True(t, runs("2018-05-29", "2018-05-31").coveredBy(dated("20180531"), nil))
	})

	t.Run("an undated trip update describes the current run", func(t *testing.T) {
		now := at("2018-05-30 15:00")
		assert.True(t, runs("2018-05-30").currentFor(undated, now))
		assert.False(t, runs("2018-05-31").currentFor(undated, now))
		assert.False(t, runs("2018-05-29").currentFor(undated, now))
		// Late in the evening, yesterday's run is a day gone.
		assert.False(t, runs("2018-05-29").currentFor(undated, at("2018-05-30 23:00")))
	})

	t.Run("an undated alert covers the runs operating while it is in force", func(t *testing.T) {
		during := period("2018-05-30 16:30", "2018-05-30 18:00")
		assert.True(t, runs("2018-05-30").coveredBy(undated, during))
		assert.False(t, runs("2018-05-31").coveredBy(undated, during))
		assert.False(t, runs("2018-05-30").coveredBy(undated, period("2018-05-30 17:30", "2018-05-30 18:00")))
		assert.True(t, runs("2018-05-31").coveredBy(undated, nil), "an alert with no period is always in force")
		openEnded := []*pb.TimeRange{{Start: ptr(uint64(at("2018-05-30 12:00").Unix()))}}
		assert.True(t, runs("2018-06-30").coveredBy(undated, openEnded))
	})

	t.Run("a trip reached as no run matches every message", func(t *testing.T) {
		now := at("2018-05-30 15:00")
		assert.True(t, runs().currentFor(dated("20200101"), now))
		assert.True(t, runs().currentFor(undated, now))
		assert.True(t, runs().coveredBy(undated, period("2020-01-01 00:00", "2020-01-02 00:00")))
	})

	t.Run("a service day counts from noon less twelve hours", func(t *testing.T) {
		// The clocks went forward at 2 am, so the day's times count from 11 pm the
		// evening before.
		start := serviceDayStart(date("2018-03-11"), la)
		assert.Equal(t, "2018-03-10T23:00:00-08:00", start.Format(time.RFC3339))
		assert.Equal(t, "2018-03-11T12:00:00-07:00", start.Add(12*time.Hour).Format(time.RFC3339))
	})
}
