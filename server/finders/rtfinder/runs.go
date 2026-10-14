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
	// Set for a trip added in real time, which ADDED messages describe.
	added bool
}

// describes reports whether a message with this trip descriptor describes one
// of the runs.
func (r tripRuns) describes(td *pb.TripDescriptor) bool {
	if r.none {
		return false
	}
	// A DUPLICATED message names the trip it copies but describes a new run.
	// An ADDED message describes only a trip added in real time: naming a
	// scheduled trip, it's the form duplicates took before DUPLICATED.
	switch td.GetScheduleRelationship() {
	case pb.TripDescriptor_DUPLICATED:
		return false
	case pb.TripDescriptor_ADDED:
		if !r.added {
			return false
		}
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

// Run names one run of a scheduled trip: the trip on one service date.
type Run struct {
	FeedVersionID int
	TripID        string // GTFS trip_id
	// The service date. Unlike a model.Trip without run dates, a Run without
	// one matches no update.
	Date tt.Date
	// When the trip runs on its service day, which places an update without a
	// start_date on the trip's current run. A zero span places it on today's.
	Span TripSpan
}

// runsOf returns a run given by value as the runs to match.
func (f *Finder) runsOf(ctx context.Context, run Run) tripRuns {
	span := func() (TripSpan, bool) { return run.Span, true }
	return tripRuns{
		dates:   []tt.Date{run.Date},
		current: sync.OnceValues(func() (tt.Date, bool) { return f.currentRun(ctx, run.FeedVersionID, span) }),
	}
}

// tripRunsOf returns a trip's runs: the requested ones, or its current run.
func (f *Finder) tripRunsOf(ctx context.Context, t *model.Trip) tripRuns {
	span := func() (TripSpan, bool) { return f.lookup.GetTripSpan(ctx, t.FeedVersionID, t.ID) }
	return tripRuns{
		dates:   t.RunDates,
		current: sync.OnceValues(func() (tt.Date, bool) { return f.currentRun(ctx, t.FeedVersionID, span) }),
		none:    t.NoRealtime,
		added:   t.AddedInRealtime,
	}
}

// currentRun returns the service date of a trip's current run, from when the
// trip runs on its service day. span is called only once the feed version's
// timezone is known.
func (f *Finder) currentRun(ctx context.Context, fvid int, span func() (TripSpan, bool)) (tt.Date, bool) {
	loc, ok := f.lookup.FeedVersionTimezone(ctx, fvid)
	if !ok {
		return tt.Date{}, false
	}
	s, ok := span()
	if !ok {
		return tt.Date{}, false
	}
	return currentRunDate(f.Clock.Now(), loc, s), true
}

// currentRunDate returns the service date of a trip's current run: the run going
// now, or failing that, the nearest of yesterday's, today's and tomorrow's. A trip
// whose span isn't known, such as one added in real time, is on today's.
func currentRunDate(now time.Time, loc *time.Location, span TripSpan) tt.Date {
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
