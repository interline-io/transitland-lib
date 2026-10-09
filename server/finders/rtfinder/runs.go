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

// Realtime data is matched to the runs of a trip it describes. A run is one trip
// on one service date.
//
// A message naming its run by the trip descriptor's start_date matches that run
// only. One naming no date describes the trip's current run, for a trip update
// or vehicle position, as the GTFS-RT spec has it; for an alert, every run
// operating while the alert is in force. A trip reached as no run in particular
// matches every message that names the trip.

// currentRunWindow is how near now a run must be to be the current one that an
// undated trip update or vehicle position describes. A trip runs at most once a
// service day, so this picks one run of a trip that runs daily.
const currentRunWindow = 12 * time.Hour

// tripRuns is the runs of one trip that realtime data is matched to.
type tripRuns struct {
	dates []tt.Date
	// When a run operates. Only resolved for a message naming no date.
	span func(date tt.Date) (start time.Time, end time.Time, ok bool)
}

// anyRun reports whether the trip was reached as no run in particular.
func (r tripRuns) anyRun() bool {
	return len(r.dates) == 0
}

// named reports whether a trip descriptor names one of the runs by date, and
// whether it names a date at all.
func (r tripRuns) named(td *pb.TripDescriptor) (match bool, dated bool) {
	d, ok := descriptorDate(td)
	if !ok {
		return false, false
	}
	return slices.ContainsFunc(r.dates, func(rd tt.Date) bool { return sameDay(rd.Val, d) }), true
}

// currentFor reports whether a trip update or vehicle position with this trip
// descriptor describes one of the runs.
func (r tripRuns) currentFor(td *pb.TripDescriptor, now time.Time) bool {
	if r.anyRun() {
		return true
	}
	if match, dated := r.named(td); dated {
		return match
	}
	for _, d := range r.dates {
		if start, end, ok := r.span(d); ok && start.Add(-currentRunWindow).Before(now) && now.Before(end.Add(currentRunWindow)) {
			return true
		}
	}
	return false
}

// coveredBy reports whether an alert, through a selector with this trip
// descriptor, concerns one of the runs.
func (r tripRuns) coveredBy(td *pb.TripDescriptor, periods []*pb.TimeRange) bool {
	if r.anyRun() {
		return true
	}
	if match, dated := r.named(td); dated {
		return match
	}
	if len(periods) == 0 {
		return true
	}
	for _, d := range r.dates {
		start, end, ok := r.span(d)
		if !ok {
			continue
		}
		for _, p := range periods {
			// An open end runs on forever, an open start from always.
			if (p.End == nil || start.Unix() < int64(p.GetEnd())) && (p.Start == nil || int64(p.GetStart()) < end.Unix()) {
				return true
			}
		}
	}
	return false
}

// tripRunsOf returns the runs of a trip as a whole, each operating from its
// first scheduled departure to its last arrival.
func (f *Finder) tripRunsOf(ctx context.Context, t *model.Trip) tripRuns {
	span := sync.OnceValues(func() (tripSpan, bool) { return f.lc.GetTripSpan(ctx, t.FeedVersionID, t.ID) })
	loc := sync.OnceValues(func() (*time.Location, bool) { return f.lc.FeedVersionTimezone(ctx, t.FeedVersionID) })
	return tripRuns{
		dates: t.RunDates,
		span: func(d tt.Date) (time.Time, time.Time, bool) {
			s, ok := span()
			l, lok := loc()
			if !ok || !lok {
				return time.Time{}, time.Time{}, false
			}
			day := serviceDayStart(d, l)
			return day.Add(time.Duration(s.FirstDeparture) * time.Second), day.Add(time.Duration(s.LastArrival) * time.Second), true
		},
	}
}

// stopTimeRuns returns the runs that include a stop time, each at the moment the
// stop time is scheduled: its own service date if it has one, else its trip's.
func (f *Finder) stopTimeRuns(ctx context.Context, t *model.Trip, st *model.StopTime) tripRuns {
	dates := t.RunDates
	if st.ServiceDate.Valid {
		dates = []tt.Date{st.ServiceDate}
	}
	at := st.DepartureTime.Int()
	if at == 0 {
		at = st.ArrivalTime.Int()
	}
	loc := sync.OnceValues(func() (*time.Location, bool) { return f.lc.FeedVersionTimezone(ctx, t.FeedVersionID) })
	return tripRuns{
		dates: dates,
		span: func(d tt.Date) (time.Time, time.Time, bool) {
			l, ok := loc()
			if !ok {
				return time.Time{}, time.Time{}, false
			}
			call := serviceDayStart(d, l).Add(time.Duration(at) * time.Second)
			return call, call, true
		},
	}
}

// descriptorDate returns the service date a trip descriptor names, if any.
func descriptorDate(td *pb.TripDescriptor) (time.Time, bool) {
	s := td.GetStartDate()
	if s == "" {
		return time.Time{}, false
	}
	d, err := time.Parse("20060102", s)
	return d, err == nil
}

func sameDay(a time.Time, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// serviceDayStart is the moment a service date's times count from: noon less
// twelve hours, which differs from midnight on the days clocks change.
func serviceDayStart(d tt.Date, loc *time.Location) time.Time {
	y, m, day := d.Val.Date()
	return time.Date(y, m, day, 12, 0, 0, 0, loc).Add(-12 * time.Hour)
}
