package rtfinder

import (
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

func TestTripRuns(t *testing.T) {
	date := func(s string) tt.Date {
		d, err := tt.ParseDate(s)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	// The runs of the given dates, with the given current run.
	runs := func(current string, dates ...string) tripRuns {
		r := tripRuns{current: func() (tt.Date, bool) { return date(current), true }}
		for _, d := range dates {
			r.dates = append(r.dates, date(d))
		}
		return r
	}
	dated := func(d string) *pb.TripDescriptor {
		return &pb.TripDescriptor{TripId: ptr("t"), StartDate: ptr(d)}
	}
	undated := &pb.TripDescriptor{TripId: ptr("t")}

	t.Run("a dated message describes the run on its date", func(t *testing.T) {
		assert.True(t, runs("2018-05-30", "2018-05-31").describes(dated("20180531")))
		assert.False(t, runs("2018-05-30", "2018-05-31").describes(dated("20180530")))
		assert.True(t, runs("2018-05-30", "2018-05-29", "2018-05-31").describes(dated("20180529")))
	})

	t.Run("an undated message describes the current run", func(t *testing.T) {
		assert.True(t, runs("2018-05-30", "2018-05-30").describes(undated))
		assert.False(t, runs("2018-05-30", "2018-05-31").describes(undated))
	})

	t.Run("a trip asked for without a date is its current run", func(t *testing.T) {
		assert.True(t, runs("2018-05-30").describes(undated))
		assert.True(t, runs("2018-05-30").describes(dated("20180530")))
		assert.False(t, runs("2018-05-30").describes(dated("20180531")))
	})

	t.Run("a start_date may also be written with hyphens", func(t *testing.T) {
		assert.True(t, runs("2018-05-30", "2018-05-31").describes(dated("2018-05-31")))
	})
}

func TestCurrentRunDate(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	at := func(s string) time.Time {
		v, err := time.ParseInLocation("2006-01-02 15:04", s, la)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	const lateRun = 24*3600 + 31*60 // last arrives at 24:31
	const dayRun = 17 * 3600        // last arrives at 17:00
	tcs := []struct {
		name        string
		now         string
		lastArrival int
		expect      string
	}{
		{"during the day, today's run", "2018-05-31 15:00", lateRun, "2018-05-31"},
		{"after midnight, yesterday's run while it is still going", "2018-05-31 00:10", lateRun, "2018-05-30"},
		{"after midnight, today's run once yesterday's has finished", "2018-05-31 00:45", lateRun, "2018-05-31"},
		{"after midnight, today's run for a trip that ends before midnight", "2018-05-31 00:10", dayRun, "2018-05-31"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, currentRunDate(at(tc.now), la, tc.lastArrival).Val.Format("2006-01-02"))
		})
	}

	t.Run("a service day counts from noon less twelve hours", func(t *testing.T) {
		// The clocks went forward at 2 am, so the day's times count from 11 pm the
		// evening before.
		start := serviceDayStart(tt.NewDate(time.Date(2018, 3, 11, 0, 0, 0, 0, time.UTC)), la)
		assert.Equal(t, "2018-03-10T23:00:00-08:00", start.Format(time.RFC3339))
	})
}
