package rtfinder

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
)

// Realtime data is matched to runs of trips. A run is one trip on one service
// date, and every realtime message describes exactly one run: the run on its
// trip descriptor's start_date, or where it names none, the trip's current run.
//
// The current run is the one going now, or failing that, the nearest of
// yesterday's, today's and tomorrow's.

// tripRuns holds the runs of one trip that a query wants matched.
type tripRuns struct {
	// The requested runs. None means the trip's current run.
	dates []tt.Date
	// The trip's current run, looked up only when needed.
	current func() (tt.Date, bool)
	// Set for a trip that stands for no run, such as a pattern's representative
	// trip, which no message describes.
	none bool
}

// describes reports whether a message with this trip descriptor describes one
// of the runs.
func (r tripRuns) describes(td *pb.TripDescriptor) bool {
	if r.none {
		return false
	}
	run, ok := descriptorDate(td)
	if !ok {
		if run, ok = r.current(); !ok {
			return false
		}
	}
	if len(r.dates) == 0 {
		current, ok := r.current()
		return ok && sameDay(current, run)
	}
	return slices.ContainsFunc(r.dates, func(d tt.Date) bool { return sameDay(d, run) })
}

// tripRunsOf returns a trip's runs: the requested ones, or its current run.
func (f *Finder) tripRunsOf(ctx context.Context, t *model.Trip) tripRuns {
	return tripRuns{
		dates:   t.RunDates,
		current: sync.OnceValues(func() (tt.Date, bool) { return f.currentRun(ctx, t.FeedVersionID, t.ID) }),
		none:    t.NoRealtime,
	}
}

// currentRun returns the service date of a trip's current run.
func (f *Finder) currentRun(ctx context.Context, fvid int, tripId int) (tt.Date, bool) {
	loc, ok := f.lc.FeedVersionTimezone(ctx, fvid)
	if !ok {
		return tt.Date{}, false
	}
	return currentRunDate(f.Clock.Now(), loc, f.lc.GetTripSpan(ctx, fvid, tripId)), true
}

// currentRunDate returns the service date of a trip's current run: the run going
// now, or failing that, the nearest of yesterday's, today's and tomorrow's. A trip
// whose span isn't known, such as one added in real time, is on today's.
func currentRunDate(now time.Time, loc *time.Location, span tripSpan) tt.Date {
	local := now.In(loc)
	today := tt.NewDate(time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC))
	if span.LastArrival == 0 {
		return today
	}
	current, nearest := today, time.Duration(-1)
	for _, days := range []int{-1, 0, 1} {
		d := tt.NewDate(today.Val.AddDate(0, 0, days))
		day := serviceDayStart(d, loc)
		start := day.Add(time.Duration(span.FirstDeparture) * time.Second)
		end := day.Add(time.Duration(span.LastArrival) * time.Second)
		// Zero while the run is going.
		gap := max(start.Sub(now), now.Sub(end), 0)
		if nearest < 0 || gap < nearest {
			current, nearest = d, gap
		}
	}
	return current
}

// descriptorDate returns the service date a trip descriptor names, if any.
func descriptorDate(td *pb.TripDescriptor) (tt.Date, bool) {
	d, err := tt.ParseDate(td.GetStartDate())
	return d, err == nil && d.Valid
}

func sameDay(a tt.Date, b tt.Date) bool {
	ay, am, ad := a.Val.Date()
	by, bm, bd := b.Val.Date()
	return ay == by && am == bm && ad == bd
}

// serviceDayStart is the start of a service date's clock: noon less twelve
// hours, which differs from midnight on the days clocks change.
func serviceDayStart(d tt.Date, loc *time.Location) time.Time {
	y, m, day := d.Val.Date()
	return time.Date(y, m, day, 12, 0, 0, 0, loc).Add(-12 * time.Hour)
}
