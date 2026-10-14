package rtfinder

import (
	"context"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
)

// A trip update is matched to a run's stop times all at once, in order along
// the trip, and its delays carry to the stop times it doesn't predict.

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

// RunUpdate is the realtime data on one run of a trip: the trip updates on the
// run, one from each realtime feed that has one.
type RunUpdate struct {
	updates []*pb.TripUpdate
}

// FindRunUpdate returns the trip updates on a run, or nil when no realtime feed
// has one.
func (f *Finder) FindRunUpdate(ctx context.Context, run Run) *RunUpdate {
	updates := f.runUpdates(ctx, run.FeedVersionID, run.TripID, f.runsOf(ctx, run))
	if len(updates) == 0 {
		return nil
	}
	return &RunUpdate{updates: updates}
}

// TripUpdate returns the first realtime feed's trip update on the run.
func (u *RunUpdate) TripUpdate() *pb.TripUpdate {
	return u.updates[0]
}

// MatchStops returns the realtime data for each of a run's stop times, in
// their order, from the trip update TripUpdate returns. base is the instant
// the stop times' seconds count from, in unix seconds: the start of the run's
// service day, or 0 for a run without a date, whose estimates then come only
// from absolute times.
func (u *RunUpdate) MatchStops(ctx context.Context, base int64, stops []StopRef) []*model.RTStopTimeUpdate {
	return matchRun(ctx, u.TripUpdate(), base, stops)
}

// FindStopTimeUpdate returns the realtime data for a stop time on a trip's
// runs: the requested ones, or its current run. False when the run has no
// trip update.
func (f *Finder) FindStopTimeUpdate(ctx context.Context, t *model.Trip, st *model.StopTime) (*model.RTStopTimeUpdate, bool) {
	r, ok := f.tripStopUpdates(ctx, t)[st.StopSequence.Int()]
	return r, ok
}

// FindStopTimeUpdates returns the realtime data for a trip's stop times, in
// their order: nil for a stop time on a run without a trip update.
func (f *Finder) FindStopTimeUpdates(ctx context.Context, t *model.Trip, sts []*model.StopTime) []*model.RTStopTimeUpdate {
	if len(sts) == 0 {
		return nil
	}
	matched := f.tripStopUpdates(ctx, t)
	ret := make([]*model.RTStopTimeUpdate, len(sts))
	for i, st := range sts {
		ret[i] = matched[st.StopSequence.Int()]
	}
	return ret
}

// tripStopUpdates matches the trip update on a trip's run to the trip's stop
// times, by stop_sequence. Nil without a trip update. The stop times come from
// the Lookup rather than the caller, which may hold only some of them, since
// matching needs them all in order.
func (f *Finder) tripStopUpdates(ctx context.Context, t *model.Trip) map[int]*model.RTStopTimeUpdate {
	updates := f.modelTripUpdates(ctx, t)
	if len(updates) == 0 {
		return nil
	}
	stops, ok := f.lookup.GetTripStops(ctx, t.FeedVersionID, t.ID)
	if !ok {
		log.For(ctx).Debug().Int("feed_version_id", t.FeedVersionID).Int("trip_id", t.ID).Msg("rtfinder: trip stops not found; its trip update isn't matched")
		return nil
	}
	ret := make(map[int]*model.RTStopTimeUpdate, len(stops))
	for i, r := range matchRun(ctx, updates[0], f.runStart(ctx, t), stops) {
		ret[stops[i].StopSequence] = r
	}
	return ret
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

// matchRun matches a trip update to a run's stop times and estimates each
// one's arrival and departure.
//
// A delay carries to the stop times the update doesn't predict: the trip's own
// delay until the first stop time update, then the delay given or implied at
// the nearest earlier one. A SKIPPED stop passes the carried delay on, and
// NO_DATA ends it.
func matchRun(ctx context.Context, tu *pb.TripUpdate, base int64, stops []StopRef) []*model.RTStopTimeUpdate {
	placed := placeUpdates(ctx, tu, stops)
	ret := make([]*model.RTStopTimeUpdate, len(stops))
	carried, hasCarried := int64(tu.GetDelay()), tu.Delay != nil
	for i, stop := range stops {
		stu := placed[i]
		r := &model.RTStopTimeUpdate{TripUpdate: tu, StopTimeUpdate: stu}
		ret[i] = r
		if stu != nil {
			switch stu.GetScheduleRelationship() {
			case pb.TripUpdate_StopTimeUpdate_SKIPPED:
				continue
			case pb.TripUpdate_StopTimeUpdate_NO_DATA:
				hasCarried = false
				continue
			}
		}
		arr, arrOK := instant(base, stop.Arrival)
		dep, depOK := instant(base, stop.Departure)
		if stu != nil {
			r.EstimatedArrival = estimate(r.ArrivalEvent(), arr, arrOK)
			r.EstimatedDeparture = estimate(r.DepartureEvent(), dep, depOK)
			if d, ok := eventDelay(stu.Departure, dep, depOK); ok {
				carried, hasCarried = d, true
			} else if d, ok := eventDelay(stu.Arrival, arr, arrOK); ok {
				carried, hasCarried = d, true
			}
		}
		if hasCarried {
			if r.EstimatedArrival == 0 && arrOK {
				r.EstimatedArrival = arr + carried
			}
			if r.EstimatedDeparture == 0 && depOK {
				r.EstimatedDeparture = dep + carried
			}
		}
	}
	return ret
}

// placeUpdates places a trip update's stop time updates on a run's stop times,
// in order along the trip: each is looked for at or after the previous one's
// stop time, which picks the right visit on a loop. One that names no stop
// time from there on is left out. Returns the update at each stop time, nil
// for none.
func placeUpdates(ctx context.Context, tu *pb.TripUpdate, stops []StopRef) []*pb.TripUpdate_StopTimeUpdate {
	placed := make([]*pb.TripUpdate_StopTimeUpdate, len(stops))
	from := 0
	for _, stu := range tu.StopTimeUpdate {
		i := stopIndex(stops, from, func(s StopRef) bool { return names(stu, s) })
		if i < 0 && stu.StopSequence != nil && stu.GetStopId() != "" {
			// The update's stop_sequence values aren't this schedule's, as after
			// a renumbering, while its stop_ids still are.
			i = stopIndex(stops, from, func(s StopRef) bool { return s.StopID != "" && s.StopID == stu.GetStopId() })
		}
		if i < 0 {
			log.TraceCheck(func() {
				log.For(ctx).Trace().Str("trip_id", tu.GetTrip().GetTripId()).Uint32("stop_sequence", stu.GetStopSequence()).Str("stop_id", stu.GetStopId()).Msg("rtfinder: stop time update names no stop time")
			})
			continue
		}
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

// stopIndex returns the index of the first stop time at or after from that
// match accepts, or -1.
func stopIndex(stops []StopRef, from int, match func(StopRef) bool) int {
	for i := from; i < len(stops); i++ {
		if match(stops[i]) {
			return i
		}
	}
	return -1
}

// instant returns when a scheduled time falls, counting its seconds from base,
// or false when either isn't known.
func instant(base int64, secs tt.Seconds) (int64, bool) {
	if base == 0 || !secs.Valid {
		return 0, false
	}
	return base + secs.Val, true
}

// estimate returns an event's prediction: its time, or else its delay added to
// the scheduled instant; 0 for none.
func estimate(e *pb.TripUpdate_StopTimeEvent, sched int64, schedOK bool) int64 {
	if e == nil {
		return 0
	}
	if e.Time != nil {
		return e.GetTime()
	}
	if e.Delay != nil && schedOK {
		return sched + int64(e.GetDelay())
	}
	return 0
}

// eventDelay returns the delay an event gives, or else the one its time implies
// against the scheduled instant.
func eventDelay(e *pb.TripUpdate_StopTimeEvent, sched int64, schedOK bool) (int64, bool) {
	if e == nil {
		return 0, false
	}
	if e.Delay != nil {
		return int64(e.GetDelay()), true
	}
	if e.Time != nil && schedOK {
		return e.GetTime() - sched, true
	}
	return 0, false
}

// AddedStopTimeUpdate returns the realtime data for a stop time of a trip
// added in real time. Such a trip has no schedule, so its estimates are the
// times its update gives.
func AddedStopTimeUpdate(tu *pb.TripUpdate, stu *pb.TripUpdate_StopTimeUpdate) *model.RTStopTimeUpdate {
	r := &model.RTStopTimeUpdate{TripUpdate: tu, StopTimeUpdate: stu}
	r.EstimatedArrival = estimate(r.ArrivalEvent(), 0, false)
	r.EstimatedDeparture = estimate(r.DepartureEvent(), 0, false)
	return r
}
