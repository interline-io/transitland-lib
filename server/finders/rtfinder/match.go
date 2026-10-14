package rtfinder

import (
	"context"
	"slices"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
)

// StopRef is one of a run's stop times: its stop_sequence and GTFS stop_id,
// and its scheduled times in seconds into the service day. Without a StopID,
// it matches only by stop_sequence; without a time, it gets no estimate from a
// delay.
type StopRef struct {
	StopSequence int        `db:"stop_sequence"`
	StopID       string     `db:"stop_id"`
	Arrival      tt.Seconds `db:"arrival_time"`
	Departure    tt.Seconds `db:"departure_time"`
}

// FindRunUpdate returns the trip update on a run, or nil when no realtime feed
// has one. With several realtime feeds, the first feed's.
func (f *Finder) FindRunUpdate(ctx context.Context, run Run) *pb.TripUpdate {
	updates := f.runUpdates(ctx, run.FeedVersionID, run.TripID, f.runsOf(ctx, run))
	if len(updates) == 0 {
		return nil
	}
	return updates[0]
}

// FindStopTimeUpdate returns the realtime data for a stop time on a trip's
// runs: the requested ones, or its current run. False when the run has no
// trip update.
func (f *Finder) FindStopTimeUpdate(ctx context.Context, t *model.Trip, st *model.StopTime) (*model.RTStopTimeUpdate, bool) {
	return f.matchTrip(ctx, t).at(st.StopSequence.Int())
}

// FindStopTimeUpdates returns the realtime data for a trip's stop times, in
// their order: nil for a stop time on a run without a trip update.
func (f *Finder) FindStopTimeUpdates(ctx context.Context, t *model.Trip, sts []*model.StopTime) []*model.RTStopTimeUpdate {
	if len(sts) == 0 {
		return nil
	}
	m := f.matchTrip(ctx, t)
	ret := make([]*model.RTStopTimeUpdate, len(sts))
	for i, st := range sts {
		ret[i], _ = m.at(st.StopSequence.Int())
	}
	return ret
}

// tripMatch is a trip update matched to a trip's stop times. Without stop
// times, the update applies to the trip's run but nothing is matched to its
// stop times: the update has nothing to place or carry, or the stop times
// couldn't be looked up.
type tripMatch struct {
	tu      *pb.TripUpdate // nil when the run has no update
	stops   []StopRef
	matched []*model.RTStopTimeUpdate
}

// at returns the realtime data for the stop time at a stop_sequence: the trip
// update alone when nothing is matched to the stop times, and nothing for a
// stop_sequence the trip doesn't have.
func (m tripMatch) at(seq int) (*model.RTStopTimeUpdate, bool) {
	if m.tu == nil {
		return nil, false
	}
	if len(m.stops) == 0 {
		return &model.RTStopTimeUpdate{TripUpdate: m.tu}, true
	}
	i := slices.IndexFunc(m.stops, func(s StopRef) bool { return s.StopSequence == seq })
	if i < 0 {
		return nil, false
	}
	return m.matched[i], true
}

// matchTrip matches the trip update on a trip's run to the trip's stop times.
// The stop times come from the Lookup rather than the caller, which may hold
// only some of them, since matching needs them all in order. An update with
// no stop time updates and no delay, such as a cancellation, needs none, and
// when the Lookup can't answer, the update applies without estimates rather
// than not at all.
func (f *Finder) matchTrip(ctx context.Context, t *model.Trip) tripMatch {
	updates := f.modelTripUpdates(ctx, t)
	if len(updates) == 0 {
		return tripMatch{}
	}
	tu := updates[0]
	if len(tu.StopTimeUpdate) == 0 && tu.Delay == nil {
		return tripMatch{tu: tu}
	}
	stops, ok := f.lookup.GetTripStops(ctx, t.FeedVersionID, t.ID)
	if !ok {
		log.For(ctx).Debug().Int("feed_version_id", t.FeedVersionID).Int("trip_id", t.ID).Msg("rtfinder: trip stops not found; its trip update applies without estimates")
		return tripMatch{tu: tu}
	}
	return tripMatch{tu: tu, stops: stops, matched: MatchStops(tu, f.runStart(ctx, t), stops)}
}

// runStart returns the instant a trip's run's stop times count from, or 0
// when the trip isn't a single run or its feed version's timezone isn't known.
func (f *Finder) runStart(ctx context.Context, t *model.Trip) int64 {
	if len(t.RunDates) != 1 {
		return 0
	}
	loc, ok := f.lookup.FeedVersionTimezone(ctx, t.FeedVersionID)
	if !ok {
		return 0
	}
	return tt.ServiceDayStart(t.RunDates[0], loc).Unix()
}

// MatchStops matches a trip update to a run's stop times, in their order, and
// estimates each one's arrival and departure. base is the instant the stop
// times' seconds count from, in unix seconds: the start of the run's service
// day, or 0 for a run without a date, whose estimates then come only from
// absolute times.
//
// A delay carries to the stop times the update doesn't predict: the trip's own
// delay until the first stop time update, then the delay given or implied at
// the nearest earlier one. A SKIPPED stop has no estimate, though a delay its
// update gives still carries, and NO_DATA ends the carry.
func MatchStops(tu *pb.TripUpdate, base int64, stops []StopRef) []*model.RTStopTimeUpdate {
	placed := placeUpdates(tu, stops)
	recs := make([]model.RTStopTimeUpdate, len(stops))
	ret := make([]*model.RTStopTimeUpdate, len(stops))
	carried, hasCarried := int64(tu.GetDelay()), tu.Delay != nil
	// carry takes the delay a stop time update gives or implies: its
	// departure's, or else its arrival's.
	carry := func(stu *pb.TripUpdate_StopTimeUpdate, arr int64, dep int64) {
		if d, ok := eventDelay(stu.GetDeparture(), dep); ok {
			carried, hasCarried = d, true
		} else if d, ok := eventDelay(stu.GetArrival(), arr); ok {
			carried, hasCarried = d, true
		}
	}
	for i, stop := range stops {
		stu := placed[i]
		r := &recs[i]
		*r = model.RTStopTimeUpdate{TripUpdate: tu, StopTimeUpdate: stu}
		ret[i] = r
		arr := instant(base, stop.Arrival)
		dep := instant(base, stop.Departure)
		switch stu.GetScheduleRelationship() {
		case pb.TripUpdate_StopTimeUpdate_NO_DATA:
			hasCarried = false
			continue
		case pb.TripUpdate_StopTimeUpdate_SKIPPED:
			carry(stu, arr, dep)
			continue
		}
		r.EstimatedArrival = estimate(r.ArrivalEvent(), arr)
		r.EstimatedDeparture = estimate(r.DepartureEvent(), dep)
		carry(stu, arr, dep)
		if hasCarried {
			if r.EstimatedArrival == 0 && arr != 0 {
				r.EstimatedArrival = arr + carried
			}
			if r.EstimatedDeparture == 0 && dep != 0 {
				r.EstimatedDeparture = dep + carried
			}
		}
	}
	return ret
}

// AddedStopTimeUpdate returns the realtime data for a stop time of a trip
// added in real time. Such a trip has no schedule, so its estimates are the
// times its update gives.
func AddedStopTimeUpdate(tu *pb.TripUpdate, stu *pb.TripUpdate_StopTimeUpdate) *model.RTStopTimeUpdate {
	r := &model.RTStopTimeUpdate{TripUpdate: tu, StopTimeUpdate: stu}
	r.EstimatedArrival = estimate(r.ArrivalEvent(), 0)
	r.EstimatedDeparture = estimate(r.DepartureEvent(), 0)
	return r
}

// placeUpdates places a trip update's stop time updates on a run's stop times,
// in order along the trip: each is looked for at or after the previous one's
// stop time, which picks the right visit on a loop. One that names no stop
// time from there on is left out. Returns the update at each stop time, nil
// for none.
func placeUpdates(tu *pb.TripUpdate, stops []StopRef) []*pb.TripUpdate_StopTimeUpdate {
	placed := make([]*pb.TripUpdate_StopTimeUpdate, len(stops))
	from := 0
	for _, stu := range tu.StopTimeUpdate {
		i := slices.IndexFunc(stops[from:], func(s StopRef) bool { return names(stu, s) })
		if i < 0 && stu.StopSequence != nil && stu.GetStopId() != "" {
			// The update's stop_sequence and stop_id name different stop times.
			// Either its stop_sequence values aren't this schedule's, as after a
			// renumbering, while its stop_ids are, or its stop_id is an assigned
			// platform the schedule doesn't have, while its stop_sequence is right.
			i = slices.IndexFunc(stops[from:], func(s StopRef) bool { return s.StopID == stu.GetStopId() })
			if i < 0 {
				i = slices.IndexFunc(stops[from:], func(s StopRef) bool { return s.StopSequence == int(stu.GetStopSequence()) })
			}
		}
		if i < 0 {
			continue
		}
		i += from
		placed[i] = stu
		from = i + 1
	}
	return placed
}

// names reports whether a stop time update names a stop time: by stop_sequence,
// agreeing on stop_id when both give one, or by stop_id alone.
func names(stu *pb.TripUpdate_StopTimeUpdate, s StopRef) bool {
	if stu.StopSequence != nil {
		return int(stu.GetStopSequence()) == s.StopSequence && (stu.GetStopId() == "" || s.StopID == "" || stu.GetStopId() == s.StopID)
	}
	return stu.GetStopId() != "" && stu.GetStopId() == s.StopID
}

// instant returns when a scheduled time falls, counting its seconds from base,
// or 0 when either isn't known.
func instant(base int64, secs tt.Seconds) int64 {
	if base == 0 || !secs.Valid {
		return 0
	}
	return base + secs.Val
}

// estimate returns an event's prediction: its time, or else its delay added to
// the scheduled instant when that's known; 0 for none.
func estimate(e *pb.TripUpdate_StopTimeEvent, sched int64) int64 {
	if e == nil {
		return 0
	}
	if e.Time != nil {
		return e.GetTime()
	}
	if e.Delay != nil && sched != 0 {
		return sched + int64(e.GetDelay())
	}
	return 0
}

// eventDelay returns the delay an event gives, or else the one its time implies
// against the scheduled instant when that's known.
func eventDelay(e *pb.TripUpdate_StopTimeEvent, sched int64) (int64, bool) {
	if e == nil {
		return 0, false
	}
	if e.Delay != nil {
		return int64(e.GetDelay()), true
	}
	if e.Time != nil && sched != 0 {
		return e.GetTime() - sched, true
	}
	return 0, false
}
