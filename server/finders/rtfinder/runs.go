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
// The current run is yesterday's while it is still running past midnight, and
// otherwise today's.

// tripRuns holds the runs of one trip that a query asks for.
type tripRuns struct {
	// The runs asked for. None means the trip's current run.
	dates []tt.Date
	// The trip's current run, looked up only when needed.
	current func() (tt.Date, bool)
}

// describes reports whether a message with this trip descriptor describes one
// of the runs.
func (r tripRuns) describes(td *pb.TripDescriptor) bool {
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

// tripRunsOf returns the runs a trip was reached as.
func (f *Finder) tripRunsOf(ctx context.Context, t *model.Trip) tripRuns {
	return tripRuns{
		dates:   t.RunDates,
		current: sync.OnceValues(func() (tt.Date, bool) { return f.currentRun(ctx, t.FeedVersionID, t.ID) }),
	}
}

// currentRun returns the service date of a trip's current run.
func (f *Finder) currentRun(ctx context.Context, fvid int, tripId int) (tt.Date, bool) {
	loc, ok := f.lc.FeedVersionTimezone(ctx, fvid)
	if !ok {
		return tt.Date{}, false
	}
	return currentRunDate(f.Clock.Now(), loc, f.lc.GetTripLastArrival(ctx, fvid, tripId)), true
}

// currentRunDate returns the service date of the current run of a trip whose
// last arrival is the given number of seconds into its service day.
func currentRunDate(now time.Time, loc *time.Location, lastArrival int) tt.Date {
	local := now.In(loc)
	today := tt.NewDate(time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC))
	yesterday := tt.NewDate(today.Val.AddDate(0, 0, -1))
	if now.Before(serviceDayStart(yesterday, loc).Add(time.Duration(lastArrival) * time.Second)) {
		return yesterday
	}
	return today
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
