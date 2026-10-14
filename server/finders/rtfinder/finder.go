package rtfinder

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/clock"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/caches/kvcache"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
)

// Cache looks up decoded RT data and stays current as feeds are refreshed.
type Cache interface {
	AddData(context.Context, string, []byte) error
	GetSource(context.Context, string) (*Source, bool)
	Close() error
}

////////

type Finder struct {
	Clock  clock.Clock
	cache  Cache
	lookup Lookup
}

// NewFinder returns an RTFinder backed by store for RT payload storage and
// cross-process distribution (via the store's pub/sub capability, if any).
// It asks lookup about static GTFS data.
func NewFinder(store kvcache.Store, lookup Lookup) *Finder {
	return &Finder{
		Clock:  &clock.Real{},
		cache:  newStoreCache(store),
		lookup: lookup,
	}
}

func (f *Finder) Close() error {
	return f.cache.Close()
}

func (f *Finder) AddData(ctx context.Context, topic string, data []byte) error {
	return f.cache.AddData(ctx, topic, data)
}

func (f *Finder) StopTimezone(ctx context.Context, id int, known string) (*time.Location, bool) {
	return f.lookup.StopTimezone(ctx, id, known)
}

func (f *Finder) FeedVersionTimezone(ctx context.Context, fvid int) (*time.Location, bool) {
	return f.lookup.FeedVersionTimezone(ctx, fvid)
}

// FindTrip returns the trip update on a trip's runs: the requested ones,
// or its current run.
func (f *Finder) FindTrip(ctx context.Context, t *model.Trip) *pb.TripUpdate {
	if found := f.modelTripUpdates(ctx, t); len(found) > 0 {
		return found[0]
	}
	return nil
}

// modelTripUpdates returns the trip updates on a model.Trip's runs, one from
// each realtime feed.
func (f *Finder) modelTripUpdates(ctx context.Context, t *model.Trip) []*pb.TripUpdate {
	return f.runUpdates(ctx, t.FeedVersionID, t.TripID.Val, f.tripRunsOf(ctx, t))
}

// runUpdates returns the trip updates naming a trip_id on its runs, one from
// each of the feed version's realtime feeds. A feed should send one update per
// run; where it sends several, the last wins.
func (f *Finder) runUpdates(ctx context.Context, fvid int, tripId string, runs tripRuns) []*pb.TripUpdate {
	var ret []*pb.TripUpdate
	topics, _ := f.lookup.GetFeedVersionRTFeeds(ctx, fvid)
	for _, topic := range topics {
		var found *pb.TripUpdate
		for _, a := range f.getTrips(ctx, topic, tripId) {
			// A DUPLICATED update names the trip it copies, but describes a
			// new run, not one of that trip's. So does an ADDED update naming
			// a scheduled trip, the form duplicates took before DUPLICATED.
			switch a.GetTrip().GetScheduleRelationship() {
			case pb.TripDescriptor_DUPLICATED:
				continue
			case pb.TripDescriptor_ADDED:
				if !runs.added {
					continue
				}
			}
			if runs.describes(a.GetTrip()) {
				found = a
			}
		}
		if found != nil {
			ret = append(ret, found)
		}
	}
	return ret
}

// FindAlertsForTrip returns the alerts on a trip's runs: the requested ones, or
// its current run.
func (f *Finder) FindAlertsForTrip(ctx context.Context, t *model.Trip, limit *int, active *bool) []*model.Alert {
	tripId := t.TripID.Val
	// Looked up at most once per call, and only for a selector naming this trip
	// along with an agency, route or mode.
	route := sync.OnceValues(func() (GtfsRoute, bool) { return f.lookup.GetRoute(ctx, t.RouteID.Int()) })
	runs := f.tripRunsOf(ctx, t)
	return f.findAlerts(ctx, t.FeedVersionID, limit, active, func(_ string, _ *Source, s *pb.EntitySelector) bool {
		return tripId != "" && s.GetTrip().GetTripId() == tripId && tripAgrees(s, route) && runs.describes(s.GetTrip())
	})
}

// FindAlertsForRoute returns the alerts on a route, as a whole or at one of its
// stops; with includeModes also those on its mode, and with includeTrips those
// on its trips.
func (f *Finder) FindAlertsForRoute(ctx context.Context, t *model.Route, limit *int, active *bool, includeModes bool, includeTrips bool) []*model.Alert {
	// Looked up at most once per topic, and only for a trip named by trip_id alone.
	tripRoutes := map[string]func() map[string]string{}
	return f.findAlerts(ctx, t.FeedVersionID, limit, active, func(topic string, src *Source, s *pb.EntitySelector) bool {
		// The agency is checked first so that another agency's trip never costs a
		// trip lookup.
		if !f.agencyMatches(ctx, s, t) {
			return false
		}
		lookup, ok := tripRoutes[topic]
		if !ok {
			lookup = sync.OnceValue(func() map[string]string { return f.tripRouteIDs(ctx, src, t.FeedVersionID) })
			tripRoutes[topic] = lookup
		}
		return onRoute(s, t.RouteID.Val, t.RouteType.Int(), includeModes, includeTrips, lookup)
	})
}

// FindAlertsForAgency returns the alerts on an agency, and also its mode-wide
// alerts on any of routeTypes.
func (f *Finder) FindAlertsForAgency(ctx context.Context, t *model.Agency, limit *int, active *bool, routeTypes []int) []*model.Alert {
	return f.findAlerts(ctx, t.FeedVersionID, limit, active, func(_ string, _ *Source, s *pb.EntitySelector) bool {
		return namesAgency(s, t.AgencyID.Val) || matchesAgencyMode(s, t.AgencyID.Val, routeTypes)
	})
}

// FindAlertsForStop returns the alerts on a stop, with or without a route, but
// not those on a trip at the stop.
func (f *Finder) FindAlertsForStop(ctx context.Context, t *model.Stop, limit *int, active *bool) []*model.Alert {
	stopId := t.StopID.Val
	fv := f.feedVersionRefs(ctx, t.FeedVersionID)
	return f.findAlerts(ctx, t.FeedVersionID, limit, active, func(_ string, _ *Source, s *pb.EntitySelector) bool {
		return stopId != "" && s.GetStopId() == stopId && !namesTrip(s) && stopAgrees(s, fv)
	})
}

// alertMatch reports whether a selector of an alert from a realtime feed is on
// the entity being asked about.
type alertMatch func(topic string, src *Source, s *pb.EntitySelector) bool

// findAlerts collects the alerts having a selector that match accepts, from
// every realtime feed associated with a feed version.
func (f *Finder) findAlerts(ctx context.Context, fvid int, limit *int, active *bool, match alertMatch) []*model.Alert {
	ret := []*model.Alert{}
	topics, _ := f.lookup.GetFeedVersionRTFeeds(ctx, fvid)
	tnow := f.Clock.Now()
	for _, topic := range topics {
		// Read even when stale: an alert carries its own active periods, and
		// many feeds serve alerts from files that never update the header.
		src, ok := f.cache.GetSource(ctx, getTopicKey(topic, "realtime_alerts"))
		if !ok || src == nil {
			continue
		}
		for _, ent := range src.alerts {
			if ent.Alert == nil || !checkAlertActivePeriod(tnow, active, ent.Alert) {
				continue
			}
			if slices.ContainsFunc(ent.Alert.GetInformedEntity(), func(s *pb.EntitySelector) bool { return s != nil && match(topic, src, s) }) {
				ret = append(ret, makeAlert(ent, topic))
			}
		}
	}
	return limitAlerts(ret, limit)
}

// agencyMatches reports whether a selector's agency_id, if it gives one, is the
// route's agency.
func (f *Finder) agencyMatches(ctx context.Context, s *pb.EntitySelector, t *model.Route) bool {
	if s.GetAgencyId() == "" {
		return true
	}
	agencyId, ok := f.lookup.GetGtfsAgencyID(ctx, t.AgencyID.Int())
	return ok && agencyIdMatches(s, agencyId)
}

// agencyIdMatches reports whether a selector's agency_id, if it gives one, is
// this GTFS agency_id.
func agencyIdMatches(s *pb.EntitySelector, agencyId string) bool {
	// An agency without an agency_id matches no selector naming one: alerts also
	// come from RT feeds shared with other operators, naming their agencies.
	aid := s.GetAgencyId()
	return aid == "" || aid == agencyId
}

// onRoute reports whether a selector is on this route: naming it as a whole or
// at a stop, or its mode with includeModes, or one of its trips with
// includeTrips. The selector's agency is not checked.
func onRoute(s *pb.EntitySelector, routeId string, routeType int, includeModes bool, includeTrips bool, tripRoutes func() map[string]string) bool {
	rid, ok := selectorRouteID(s)
	if !ok || (rid != "" && rid != routeId) || !modeAgrees(s, routeType) {
		return false
	}
	if namesTrip(s) {
		if !includeTrips {
			return false
		}
		if rid != "" {
			return true
		}
		// A trip named by trip_id alone is placed on its route by the schedule.
		tid := s.GetTrip().GetTripId()
		if tid == "" {
			return false
		}
		tripRouteId, ok := tripRoutes()[tid]
		return ok && tripRouteId == routeId
	}
	return rid != "" || (includeModes && isModeWide(s))
}

// selectorRouteID returns the route_id a selector gives, directly or in its trip
// descriptor, and false if the two disagree.
func selectorRouteID(s *pb.EntitySelector) (string, bool) {
	rid, tdRid := s.GetRouteId(), s.GetTrip().GetRouteId()
	if rid == "" {
		return tdRid, true
	}
	return rid, tdRid == "" || tdRid == rid
}

// tripOnRoute reports whether a trip descriptor names a trip of this route, by
// its route_id or, without one, its trip_id.
func tripOnRoute(td *pb.TripDescriptor, routeId string, tripRoutes func() map[string]string) bool {
	if rid := td.GetRouteId(); rid != "" {
		return rid == routeId
	}
	if tid := td.GetTripId(); tid != "" {
		rid, ok := tripRoutes()[tid]
		return ok && rid == routeId
	}
	return false
}

// modeAgrees reports whether a selector's route_type, if it gives one, is this
// route type's mode.
func modeAgrees(s *pb.EntitySelector, routeType int) bool {
	return s.RouteType == nil || tt.BasicRouteType(int(s.GetRouteType())) == tt.BasicRouteType(routeType)
}

// isModeWide reports whether a selector names a route type and no route, stop
// or trip.
func isModeWide(s *pb.EntitySelector) bool {
	return s.RouteType != nil && !namesRouteStopOrTrip(s)
}

// namesAgency reports whether a selector names this agency and nothing narrower.
func namesAgency(s *pb.EntitySelector, agencyId string) bool {
	aid := s.GetAgencyId()
	return aid != "" && aid == agencyId && s.RouteType == nil && !namesRouteStopOrTrip(s)
}

// namesRouteStopOrTrip reports whether a selector names a route, stop or trip.
func namesRouteStopOrTrip(s *pb.EntitySelector) bool {
	return s.GetRouteId() != "" || s.GetTrip().GetRouteId() != "" || s.GetStopId() != "" || namesTrip(s)
}

// namesTrip reports whether a selector narrows to a trip: its trip descriptor
// gives a trip_id, or a start time or date that picks out one of a route's trips.
func namesTrip(s *pb.EntitySelector) bool {
	td := s.GetTrip()
	return td.GetTripId() != "" || td.GetStartTime() != "" || td.GetStartDate() != ""
}

// matchesAgencyMode reports whether a mode-wide selector for this agency covers
// any of these route types.
func matchesAgencyMode(s *pb.EntitySelector, agencyId string, routeTypes []int) bool {
	if !isModeWide(s) || !agencyIdMatches(s, agencyId) {
		return false
	}
	return slices.ContainsFunc(routeTypes, func(rt int) bool { return modeAgrees(s, rt) })
}

// tripAgrees reports whether the agency, route and mode a selector gives, if
// any, are those of the trip's route.
func tripAgrees(s *pb.EntitySelector, route func() (GtfsRoute, bool)) bool {
	rid, ok := selectorRouteID(s)
	if !ok {
		return false
	}
	if rid == "" && s.GetAgencyId() == "" && s.RouteType == nil {
		return true
	}
	r, found := route()
	return found && (rid == "" || rid == r.RouteID) && agencyIdMatches(s, r.AgencyID) && modeAgrees(s, r.RouteType)
}

// feedVersionRefs looks up a feed version's agencies, route types and routes,
// each at most once per call.
type feedVersionRefs struct {
	agencyIds  func() ([]string, bool)
	routeTypes func() ([]int, bool)
	route      func(routeId string) (GtfsRoute, bool)
}

func (f *Finder) feedVersionRefs(ctx context.Context, fvid int) feedVersionRefs {
	type found struct {
		route GtfsRoute
		ok    bool
	}
	routes := map[string]found{}
	return feedVersionRefs{
		agencyIds:  sync.OnceValues(func() ([]string, bool) { return f.lookup.GetFeedVersionAgencyIDs(ctx, fvid) }),
		routeTypes: sync.OnceValues(func() ([]int, bool) { return f.lookup.GetFeedVersionRouteTypes(ctx, fvid) }),
		route: func(routeId string) (GtfsRoute, bool) {
			r, seen := routes[routeId]
			if !seen {
				r.route, r.ok = f.lookup.GetFeedVersionRoute(ctx, fvid, routeId)
				routes[routeId] = r
			}
			return r.route, r.ok
		},
	}
}

// stopAgrees reports whether the agency, route and mode a selector gives, if
// any, are in the stop's feed version and agree with each other.
func stopAgrees(s *pb.EntitySelector, fv feedVersionRefs) bool {
	rid, ok := selectorRouteID(s)
	if !ok {
		return false
	}
	if rid != "" {
		r, found := fv.route(rid)
		return found && agencyIdMatches(s, r.AgencyID) && modeAgrees(s, r.RouteType)
	}
	if aid := s.GetAgencyId(); aid != "" {
		if agencyIds, _ := fv.agencyIds(); !slices.Contains(agencyIds, aid) {
			return false
		}
	}
	if s.RouteType != nil {
		if routeTypes, _ := fv.routeTypes(); !slices.ContainsFunc(routeTypes, func(rt int) bool { return modeAgrees(s, rt) }) {
			return false
		}
	}
	return true
}

// GetMessage returns a feed's held payload however old, and whether it's
// stale: too far from now to match trip updates and vehicle positions.
func (f *Finder) GetMessage(ctx context.Context, topic string, topicKey string) (msg *pb.FeedMessage, stale bool, ok bool) {
	src, ok := f.cache.GetSource(ctx, getTopicKey(topic, topicKey))
	if !ok || src == nil {
		return nil, false, false
	}
	return src.msg, f.stale(src), true
}

// maxPayloadAge is how far a payload's header timestamp can be from the
// Finder's clock before the payload is stale. Live feeds are a few minutes old
// at most, while feeds that have stopped updating are hours old.
const maxPayloadAge = time.Hour

// stale reports whether a payload's header timestamp is more than
// maxPayloadAge from the Finder's clock. A header that far ahead of the clock
// is stale too, as from a producer stamping milliseconds or with a skewed
// clock. A payload without a header timestamp isn't stale.
func (f *Finder) stale(src *Source) bool {
	ts := src.GetTimestamp()
	if ts == 0 {
		return false
	}
	age := f.Clock.Now().Sub(time.Unix(int64(ts), 0))
	return age > maxPayloadAge || age < -maxPayloadAge
}

// currentSource returns a feed's held payload, or nothing when it's stale.
func (f *Finder) currentSource(ctx context.Context, topicKey string) (*Source, bool) {
	src, ok := f.cache.GetSource(ctx, topicKey)
	if !ok || src == nil {
		return nil, false
	}
	if f.stale(src) {
		log.TraceCheck(func() {
			log.For(ctx).Trace().Str("topic", topicKey).Uint64("header_timestamp", src.GetTimestamp()).Msg("rtfinder: payload too old to match")
		})
		return nil, false
	}
	return src, true
}

// FindStopTimeUpdate returns the update for a stop time on a trip's runs: the
// requested ones, or its current run.
func (f *Finder) FindStopTimeUpdate(ctx context.Context, t *model.Trip, st *model.StopTime) (*model.RTStopTimeUpdate, bool) {
	return matchStop(ctx, f.modelTripUpdates(ctx, t), st.StopSequence.Int(), f.gtfsStopID(ctx, st))
}

// FindStopTimeUpdates returns the updates for a trip's stop times, in their
// order: nil for a stop time without one.
func (f *Finder) FindStopTimeUpdates(ctx context.Context, t *model.Trip, sts []*model.StopTime) []*model.RTStopTimeUpdate {
	if len(sts) == 0 {
		return nil
	}
	rtTrips := f.modelTripUpdates(ctx, t)
	ret := make([]*model.RTStopTimeUpdate, len(sts))
	for i, st := range sts {
		ret[i], _ = matchStop(ctx, rtTrips, st.StopSequence.Int(), f.gtfsStopID(ctx, st))
	}
	return ret
}

// gtfsStopID returns a function that looks up a stop time's GTFS stop_id.
func (f *Finder) gtfsStopID(ctx context.Context, st *model.StopTime) func() (string, bool) {
	return func() (string, bool) { return f.lookup.GetGtfsStopID(ctx, st.StopID.Int()) }
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

// StopRef names one of a run's stop times. Without a StopID, it matches only
// by stop_sequence.
type StopRef struct {
	StopSequence int
	StopID       string // GTFS stop_id
}

// MatchStops returns the updates for a run's stop times, in their order: nil
// for a stop time without one.
func (u *RunUpdate) MatchStops(ctx context.Context, stops []StopRef) []*model.RTStopTimeUpdate {
	ret := make([]*model.RTStopTimeUpdate, len(stops))
	for i, st := range stops {
		ret[i], _ = matchStop(ctx, u.updates, st.StopSequence, func() (string, bool) { return st.StopID, st.StopID != "" })
	}
	return ret
}

// matchStop returns the update for the stop time at stop_sequence seq from the
// trip updates on its run, one from each realtime feed. stopId returns the
// stop's GTFS stop_id, and is called only when no update names seq.
func matchStop(ctx context.Context, rtTrips []*pb.TripUpdate, seq int, stopId func() (string, bool)) (*model.RTStopTimeUpdate, bool) {
	// Attempt to match on stop sequence, among the updates that name one.
	for _, rtTrip := range rtTrips {
		for _, ste := range rtTrip.StopTimeUpdate {
			if ste.StopSequence != nil && int(ste.GetStopSequence()) == seq {
				log.TraceCheck(func() {
					log.For(ctx).Trace().Str("trip_id", rtTrip.GetTrip().GetTripId()).Int("seq", seq).Msg("found stop time update on trip_id/stop_sequence")
				})
				return &model.RTStopTimeUpdate{TripUpdate: rtTrip, StopTimeUpdate: ste}, true
			}
		}
	}
	// Attempt to match on stop id, in the first feed's update. The GTFS stop id
	// is looked up only now, as most stop times match on stop sequence.
	if len(rtTrips) == 0 {
		return nil, false
	}
	sid, ok := stopId()
	if !ok {
		return nil, false
	}
	rtTrip := rtTrips[0]
	// The stop_id identifies the stop time only in an update that names no
	// stop_sequence, which would have matched above if it were this stop
	// time's, and only if the update visits the stop once.
	byStopId := func(ste *pb.TripUpdate_StopTimeUpdate) bool {
		return ste.StopSequence == nil && ste.GetStopId() == sid
	}
	visits := 0
	for _, ste := range rtTrip.StopTimeUpdate {
		if byStopId(ste) {
			visits++
		}
	}
	if visits > 1 {
		return nil, true
	}
	var lastDelay *int32
	for _, ste := range rtTrip.StopTimeUpdate {
		if ste.Arrival != nil && ste.Arrival.Delay != nil {
			lastDelay = ste.Arrival.Delay
		}
		if ste.Departure != nil && ste.Departure.Delay != nil {
			lastDelay = ste.Departure.Delay
		}
		if byStopId(ste) {
			log.TraceCheck(func() {
				log.For(ctx).Trace().Str("trip_id", rtTrip.GetTrip().GetTripId()).Str("stop_id", sid).Msg("found stop time update on trip_id/stop_id")
			})
			return &model.RTStopTimeUpdate{TripUpdate: rtTrip, StopTimeUpdate: ste, LastDelay: copyPtr(lastDelay)}, true
		}
	}
	// Matched on trip, but no match on stop sequence or stop_id
	return &model.RTStopTimeUpdate{TripUpdate: rtTrip, LastDelay: copyPtr(lastDelay)}, true
}

// TODO: put this method on consumer and wrap, as with GetTrips
func (f *Finder) GetAddedTripsForStop(ctx context.Context, t *model.Stop) []*pb.TripUpdate {
	sid := t.StopID
	var ret []*pb.TripUpdate
	topics, _ := f.lookup.GetFeedVersionRTFeeds(ctx, t.FeedVersionID)
	for _, topic := range topics {
		a, ok := f.currentSource(ctx, getTopicKey(topic, "realtime_trip_updates"))
		if !ok {
			continue
		}
		// TODO: index more efficiently
		// One added trip per trip_id, the last in the feed, and only on a
		// route the feed version has: MakeTrip builds its trip from that route.
		for _, trips := range a.tripUpdates {
			trip := trips[len(trips)-1]
			if trip.Trip.GetScheduleRelationship() != pb.TripDescriptor_ADDED || !slices.ContainsFunc(trip.StopTimeUpdate, func(ste *pb.TripUpdate_StopTimeUpdate) bool { return ste.GetStopId() == sid.Val }) {
				continue
			}
			if _, ok := f.lookup.GetFeedVersionRoute(ctx, t.FeedVersionID, trip.Trip.GetRouteId()); !ok {
				continue
			}
			ret = append(ret, trip)
		}
	}
	return ret
}

func (f *Finder) MakeTrip(ctx context.Context, obj *model.Trip) (*model.Trip, error) {
	t := model.Trip{}
	t.FeedVersionID = obj.FeedVersionID
	t.TripID = obj.TripID
	t.RTTripID = obj.TripID.Val
	// Any update naming the trip will do: this only needs its route and
	// direction, whichever run it describes.
	topics, _ := f.lookup.GetFeedVersionRTFeeds(ctx, obj.FeedVersionID)
	var rtTrip *pb.TripUpdate
	for _, topic := range topics {
		if found := f.getTrips(ctx, topic, t.TripID.Val); len(found) > 0 {
			rtTrip = found[0]
			break
		}
	}
	if rtTrip != nil {
		rtt := rtTrip.Trip
		r, ok := f.lookup.GetFeedVersionRoute(ctx, obj.FeedVersionID, rtt.GetRouteId())
		if !ok {
			return nil, errors.New("not found")
		}
		t.RouteID.Set(strconv.Itoa(r.ID))
		t.DirectionID.SetInt(int(rtt.GetDirectionId()))
		return &t, nil
	}
	return nil, errors.New("not found")
}

// getTrips returns a feed's trip updates naming a trip_id, one for each run
// reported.
func (f *Finder) getTrips(ctx context.Context, topic string, tid string) []*pb.TripUpdate {
	if tid == "" {
		return nil
	}
	a, ok := f.currentSource(ctx, getTopicKey(topic, "realtime_trip_updates"))
	if !ok {
		return nil
	}
	return a.GetTrips(tid)
}

// tripRouteIDs maps the trip_ids a realtime message names without a route_id
// to their GTFS route_ids in a feed version. It holds each feed version's
// answer for as long as the message is current.
func (f *Finder) tripRouteIDs(ctx context.Context, src *Source, fvid int) map[string]string {
	if len(src.unroutedTripIds) == 0 {
		return nil
	}
	src.tripRoutesOnce.Do(func() {
		src.tripRoutes = newRefreshCache("triproutes", func(ctx context.Context, fvid int) (map[string]string, error) {
			ret, ok := f.lookup.GetTripRouteIDs(ctx, fvid, src.unroutedTripIds)
			if !ok {
				return nil, errLookupFailed
			}
			return ret, nil
		})
	})
	// A failed lookup is not cached: its trips match no route until a later
	// call succeeds.
	ret, _ := getUnlessGone(ctx, src.tripRoutes, fvid)
	return ret
}

func checkAlertActivePeriod(t time.Time, active *bool, a *pb.Alert) bool {
	if active == nil || *active == false {
		return true
	}
	tt := uint64(t.Unix())
	if len(a.ActivePeriod) == 0 {
		return true
	}
	for _, ap := range a.ActivePeriod {
		if ap == nil {
			continue
		}
		start := ap.Start
		end := ap.End
		if start != nil && end != nil && *start < tt && *end > tt {
			// fmt.Printf("\tstart %d < now %d < end %d\n", nilor(start), tt, nilor(end))
			return true
		} else if start != nil && end == nil && *start < tt {
			// fmt.Printf("\tstart %d < now %d\n", nilor(start), tt)
			return true
		} else if start == nil && end != nil && *end > tt {
			// fmt.Printf("\tnow %d < end %d\n", tt, nilor(end))
			return true
		} else {
			// fmt.Printf("not match: %d %d now: %d\n", nilor(start), nilor(end), tt)
		}
	}
	return false
}

func limitAlerts(alerts []*model.Alert, limit *int) []*model.Alert {
	lim := len(alerts)
	if limit != nil {
		lim = *limit
	}
	if len(alerts) > lim {
		return alerts[0:lim]
	}
	return alerts
}

func makeAlert(ent alertEntity, rtFeedOnestopID string) *model.Alert {
	a := ent.Alert
	r := model.Alert{
		EntityID:        ent.ID,
		RtFeedOnestopID: rtFeedOnestopID,
	}
	if a.Cause != nil {
		r.Cause = pstr(a.Cause.String())
	}
	if a.Effect != nil {
		r.Effect = pstr(a.Effect.String())
	}
	if a.SeverityLevel != nil {
		r.SeverityLevel = pstr(a.SeverityLevel.String())
	}
	for _, tr := range a.ActivePeriod {
		rttr := model.RTTimeRange{}
		if tr.Start != nil {
			v := int(*tr.Start)
			rttr.Start = &v
		}
		if tr.End != nil {
			v := int(*tr.End)
			rttr.End = &v
		}
		r.ActivePeriod = append(r.ActivePeriod, &rttr)
	}
	r.HeaderText = newTranslation(a.HeaderText)
	r.DescriptionText = newTranslation(a.DescriptionText)
	r.TtsHeaderText = newTranslation(a.TtsHeaderText)
	r.TtsDescriptionText = newTranslation(a.TtsDescriptionText)
	r.URL = newTranslation(a.Url)
	for _, ie := range a.InformedEntity {
		if ie != nil {
			r.InformedEntity = append(r.InformedEntity, makeEntitySelector(ie))
		}
	}
	return &r
}

func makeEntitySelector(ie *pb.EntitySelector) *model.RTEntitySelector {
	r := model.RTEntitySelector{
		AgencyID: pstr(ie.GetAgencyId()),
		RouteID:  pstr(ie.GetRouteId()),
		StopID:   pstr(ie.GetStopId()),
	}
	if ie.RouteType != nil {
		r.RouteType = ptr(int(ie.GetRouteType()))
	}
	if ie.DirectionId != nil {
		r.DirectionID = ptr(int(ie.GetDirectionId()))
	}
	if ie.Trip != nil {
		r.Trip = makeTripDescriptor(ie.Trip)
	}
	return &r
}

func pstr(v string) *string {
	if v == "" {
		return nil
	}
	v2 := v
	return &v2
}

func newTranslation(v *pb.TranslatedString) []*model.RTTranslation {
	if v == nil {
		return nil
	}
	var ret []*model.RTTranslation
	for _, tr := range v.Translation {
		ntr := model.RTTranslation{
			Language: tr.Language,
		}
		if tr.Text != nil {
			ntr.Text = *tr.Text
		}
		ret = append(ret, &ntr)
	}
	return ret
}

// getTopicKey is called once per associated RT feed per stop time, so it
// concatenates rather than formatting.
func getTopicKey(topic string, t string) string {
	return "rtdata:" + topic + ":" + t
}

func copyPtr[T any, PT *T](v PT) PT {
	if v == nil {
		return nil
	}
	a := *v
	return &a
}
