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
	"github.com/interline-io/transitland-lib/tldb"
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
	Clock clock.Clock
	cache Cache
	lc    *lookupCache
}

// NewFinder returns an RTFinder backed by store for RT payload storage and
// cross-process distribution (via the store's pub/sub capability, if any).
func NewFinder(store kvcache.Store, db tldb.Ext) *Finder {
	return &Finder{
		Clock: &clock.Real{},
		cache: newStoreCache(store),
		lc:    newLookupCache(db),
	}
}

func (f *Finder) Close() error {
	return f.cache.Close()
}

func (f *Finder) AddData(ctx context.Context, topic string, data []byte) error {
	return f.cache.AddData(ctx, topic, data)
}

func (f *Finder) StopTimezone(ctx context.Context, id int, known string) (*time.Location, bool) {
	return f.lc.StopTimezone(ctx, id, known)
}

func (f *Finder) FeedVersionTimezone(ctx context.Context, fvid int) (*time.Location, bool) {
	return f.lc.FeedVersionTimezone(ctx, fvid)
}

func (f *Finder) FindTrip(ctx context.Context, t *model.Trip) *pb.TripUpdate {
	topics, _ := f.lc.GetFeedVersionRTFeeds(ctx, t.FeedVersionID)
	for _, topic := range topics {
		if a, ok := f.getTrip(ctx, topic, t.TripID.Val); ok {
			return a
		}
	}
	return nil
}

// FindAlertsForTrip returns the alerts on a trip.
func (f *Finder) FindAlertsForTrip(ctx context.Context, t *model.Trip, limit *int, active *bool) []*model.Alert {
	foundAlerts := []*model.Alert{}
	topics, _ := f.lc.GetFeedVersionRTFeeds(ctx, t.FeedVersionID)
	tnow := f.Clock.Now()
	for _, topic := range topics {
		a, ok := f.cache.GetSource(ctx, getTopicKey(topic, "realtime_alerts"))
		if a == nil || !ok {
			continue
		}
		for _, ent := range a.alerts {
			alert := ent.Alert
			if alert == nil {
				continue
			}
			if !checkAlertActivePeriod(tnow, active, alert) {
				continue
			}
			found := false
			for _, s := range alert.GetInformedEntity() {
				if s == nil || t.TripID.Val == "" || s.GetTrip().GetTripId() != t.TripID.Val {
					continue
				}
				if f.tripAgrees(ctx, s, t) {
					found = true
					break
				}
			}
			if found {
				foundAlerts = append(foundAlerts, makeAlert(ent, topic))
			}
		}
	}
	return limitAlerts(foundAlerts, limit)
}

// FindAlertsForRoute returns the alerts on a route, as a whole or at one of its
// stops; with includeModes also those on its mode, and with includeTrips those
// on its trips.
func (f *Finder) FindAlertsForRoute(ctx context.Context, t *model.Route, limit *int, active *bool, includeModes bool, includeTrips bool) []*model.Alert {
	foundAlerts := []*model.Alert{}
	topics, _ := f.lc.GetFeedVersionRTFeeds(ctx, t.FeedVersionID)
	tnow := f.Clock.Now()
	for _, topic := range topics {
		a, ok := f.cache.GetSource(ctx, getTopicKey(topic, "realtime_alerts"))
		if a == nil || !ok {
			continue
		}
		// Looked up at most once per call, and only for a trip named by trip_id alone.
		tripRoutes := sync.OnceValue(func() map[string]string { return f.lc.GetTripRouteIDs(ctx, a, t.FeedVersionID) })
		for _, ent := range a.alerts {
			alert := ent.Alert
			if !checkAlertActivePeriod(tnow, active, alert) {
				continue
			}
			if alert == nil {
				continue
			}
			found := false
			for _, s := range alert.GetInformedEntity() {
				// The agency is checked first so that another agency's trip never
				// costs a trip lookup.
				if s == nil || !f.agencyMatches(ctx, s, t) {
					continue
				}
				if onRoute(s, t.RouteID.Val, t.RouteType.Int(), includeModes, includeTrips, tripRoutes) {
					found = true
					break
				}
			}
			if found {
				foundAlerts = append(foundAlerts, makeAlert(ent, topic))
			}
		}
	}
	return limitAlerts(foundAlerts, limit)
}

// agencyMatches reports whether a selector's agency_id, if it gives one, is the
// route's agency.
func (f *Finder) agencyMatches(ctx context.Context, s *pb.EntitySelector, t *model.Route) bool {
	if s.GetAgencyId() == "" {
		return true
	}
	agencyId, ok := f.lc.GetGtfsAgencyID(ctx, t.AgencyID.Int())
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

// onRoute reports whether a selector, its agency already checked, is on this
// route: it names the route, as a whole or at a stop; with includeModes it
// covers the route's mode; with includeTrips it names one of the route's trips,
// however the trip names its route.
func onRoute(s *pb.EntitySelector, routeId string, routeType int, includeModes bool, includeTrips bool, tripRoutes func() map[string]string) bool {
	if !modeAgrees(s, routeType) {
		return false
	}
	if namesTrip(s) {
		return includeTrips && selectorNamesRoute(s, routeId, tripRoutes)
	}
	if rid := s.GetRouteId(); rid != "" {
		return rid == routeId
	}
	return includeModes && matchesRouteType(s, routeType)
}

// selectorNamesRoute reports whether a selector names this route, directly or
// through one of its trips.
func selectorNamesRoute(s *pb.EntitySelector, routeId string, tripRoutes func() map[string]string) bool {
	if rid := s.GetRouteId(); rid != "" {
		return rid == routeId
	}
	return tripOnRoute(s.GetTrip(), routeId, tripRoutes)
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

// matchesRouteType reports whether a selector naming no route, stop or trip
// covers every route of this type.
func matchesRouteType(s *pb.EntitySelector, routeType int) bool {
	return isModeWide(s) && modeAgrees(s, routeType)
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
	return s.GetRouteId() != "" || s.GetStopId() != "" || namesTrip(s)
}

// namesTrip reports whether a selector narrows to a trip, by its trip_id or its
// route_id. An empty trip descriptor names no trip.
func namesTrip(s *pb.EntitySelector) bool {
	td := s.GetTrip()
	return td.GetTripId() != "" || td.GetRouteId() != ""
}

// tripAgrees reports whether the agency, route and mode a selector gives, if
// any, are this trip's.
func (f *Finder) tripAgrees(ctx context.Context, s *pb.EntitySelector, t *model.Trip) bool {
	rid, tdRid := s.GetRouteId(), s.GetTrip().GetRouteId()
	if s.GetAgencyId() == "" && rid == "" && tdRid == "" && s.RouteType == nil {
		return true
	}
	r, ok := f.lc.GetGtfsRoute(ctx, t.RouteID.Int())
	if !ok {
		return false
	}
	return agencyIdMatches(s, r.AgencyID) && (rid == "" || rid == r.RouteID) && (tdRid == "" || tdRid == r.RouteID) && modeAgrees(s, r.RouteType)
}

// inFeedVersion reports whether the agency and route a selector gives, if any,
// are in this feed version. A stop belongs to no single agency or route, so this
// only keeps out another feed's selectors that share the stop's stop_id.
func (f *Finder) inFeedVersion(ctx context.Context, s *pb.EntitySelector, fvid int) bool {
	if aid := s.GetAgencyId(); aid != "" && !f.lc.GetFeedVersionAgencyIDs(ctx, fvid).Contains(aid) {
		return false
	}
	if rid := s.GetRouteId(); rid != "" && !f.lc.GetFeedVersionRouteIDs(ctx, fvid).Contains(rid) {
		return false
	}
	return true
}

// matchesAgencyMode reports whether a mode-wide selector for this agency covers
// any of these route types.
func matchesAgencyMode(s *pb.EntitySelector, agencyId string, routeTypes []int) bool {
	if !isModeWide(s) || !agencyIdMatches(s, agencyId) {
		return false
	}
	mode := tt.BasicRouteType(int(s.GetRouteType()))
	return slices.ContainsFunc(routeTypes, func(rt int) bool { return tt.BasicRouteType(rt) == mode })
}

func (f *Finder) GetMessage(ctx context.Context, topic string, topicKey string) (*pb.FeedMessage, bool) {
	tk := getTopicKey(topic, topicKey)
	a, ok := f.cache.GetSource(ctx, tk)
	if a != nil && ok {
		return a.msg, ok
	}
	return nil, false
}

// FindAlertsForAgency returns the alerts on an agency, and also its mode-wide
// alerts on any of routeTypes.
func (f *Finder) FindAlertsForAgency(ctx context.Context, t *model.Agency, limit *int, active *bool, routeTypes []int) []*model.Alert {
	foundAlerts := []*model.Alert{}
	topics, _ := f.lc.GetFeedVersionRTFeeds(ctx, t.FeedVersionID)
	tnow := f.Clock.Now()
	for _, topic := range topics {
		a, ok := f.cache.GetSource(ctx, getTopicKey(topic, "realtime_alerts"))
		if a == nil || !ok {
			continue
		}
		for _, ent := range a.alerts {
			alert := ent.Alert
			if alert == nil {
				continue
			}
			if !checkAlertActivePeriod(tnow, active, alert) {
				continue
			}
			found := false
			for _, s := range alert.GetInformedEntity() {
				if s == nil {
					continue
				}
				if namesAgency(s, t.AgencyID.Val) || matchesAgencyMode(s, t.AgencyID.Val, routeTypes) {
					found = true
					break
				}
			}
			if found {
				foundAlerts = append(foundAlerts, makeAlert(ent, topic))
			}
		}
	}
	return limitAlerts(foundAlerts, limit)
}

// FindAlertsForStop returns the alerts on a stop, with or without a route, but
// not those on a trip at the stop.
func (f *Finder) FindAlertsForStop(ctx context.Context, t *model.Stop, limit *int, active *bool) []*model.Alert {
	foundAlerts := []*model.Alert{}
	topics, _ := f.lc.GetFeedVersionRTFeeds(ctx, t.FeedVersionID)
	tnow := f.Clock.Now()
	for _, topic := range topics {
		a, ok := f.cache.GetSource(ctx, getTopicKey(topic, "realtime_alerts"))
		if a == nil || !ok {
			continue
		}
		for _, ent := range a.alerts {
			alert := ent.Alert
			if !checkAlertActivePeriod(tnow, active, alert) {
				continue
			}
			if alert == nil {
				continue
			}
			found := false
			for _, s := range alert.GetInformedEntity() {
				if s == nil || namesTrip(s) || s.GetStopId() != t.StopID.Val {
					continue
				}
				if f.inFeedVersion(ctx, s, t.FeedVersionID) {
					found = true
					break
				}
			}
			if found {
				foundAlerts = append(foundAlerts, makeAlert(ent, topic))
			}
		}
	}
	return limitAlerts(foundAlerts, limit)
}

func (f *Finder) FindStopTimeUpdate(ctx context.Context, t *model.Trip, st *model.StopTime) (*model.RTStopTimeUpdate, bool) {
	seq := st.StopSequence.Int()
	// Resolve the trip in each RT feed once. Both passes below ask every topic
	// the same question, and a feed version can be associated with dozens of RT
	// feeds, so answering twice is most of the work.
	topics, _ := f.lc.GetFeedVersionRTFeeds(ctx, t.FeedVersionID)
	var rtTrips []*pb.TripUpdate
	for _, topic := range topics {
		if rtTrip, ok := f.getTrip(ctx, topic, t.TripID.Val); ok {
			rtTrips = append(rtTrips, rtTrip)
		}
	}
	// Attempt to match on stop sequence
	for _, rtTrip := range rtTrips {
		for _, ste := range rtTrip.StopTimeUpdate {
			if int(ste.GetStopSequence()) == seq {
				log.For(ctx).Trace().Str("trip_id", t.TripID.Val).Int("seq", seq).Msgf("found stop time update on trip_id/stop_sequence")
				return &model.RTStopTimeUpdate{TripUpdate: rtTrip, StopTimeUpdate: ste}, true
			}
		}
	}
	// Attempt to match on stop id
	for _, rtTrip := range rtTrips {
		// If no match on stop sequence, match on stop_id if stop is not visited twice
		check := map[string]int{}
		for _, ste := range rtTrip.StopTimeUpdate {
			check[ste.GetStopId()] += 1
		}
		// Get GTFS stop id for comparing with RT
		sid, ok := f.lc.GetGtfsStopID(st.StopID.Int())
		if !ok {
			continue
		}
		// Skip if this stop is visited twice and no stop sequence is matched (above)
		if check[sid] > 1 {
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
			if sid == ste.GetStopId() {
				log.For(ctx).Trace().Str("trip_id", t.TripID.Val).Str("stop_id", sid).Msgf("found stop time update on trip_id/stop_id")
				return &model.RTStopTimeUpdate{TripUpdate: rtTrip, StopTimeUpdate: ste, LastDelay: copyPtr(lastDelay)}, true
			}
		}
		// Matched on trip, but no match on stop sequence or stop_id
		return &model.RTStopTimeUpdate{TripUpdate: rtTrip, LastDelay: copyPtr(lastDelay)}, true
	}
	// log.For(ctx).Trace().Str("trip_id", t.TripID.Val).Int("seq", seq).Msgf("no stop time update found")
	return nil, false
}

// TODO: put this method on consumer and wrap, as with GetTrip
func (f *Finder) GetAddedTripsForStop(ctx context.Context, t *model.Stop) []*pb.TripUpdate {
	sid := t.StopID
	var ret []*pb.TripUpdate
	topics, _ := f.lc.GetFeedVersionRTFeeds(ctx, t.FeedVersionID)
	for _, topic := range topics {
		a, ok := f.cache.GetSource(ctx, getTopicKey(topic, "realtime_trip_updates"))
		if !ok {
			continue
		}
		// TODO: index more efficiently
		for _, trip := range a.entityByTrip {
			if trip.Trip.GetScheduleRelationship() != pb.TripDescriptor_ADDED {
				continue
			}
			for _, ste := range trip.StopTimeUpdate {
				if ste.GetStopId() == sid.Val {
					ret = append(ret, trip)
					break // continue to next trip
				}
			}
		}
	}
	return ret
}

func (f *Finder) MakeTrip(ctx context.Context, obj *model.Trip) (*model.Trip, error) {
	t := model.Trip{}
	t.FeedVersionID = obj.FeedVersionID
	t.TripID = obj.TripID
	t.RTTripID = obj.RTTripID
	if rtTrip := f.FindTrip(ctx, &t); rtTrip != nil {
		rtt := rtTrip.Trip
		rid, ok := f.lc.GetRouteID(obj.FeedVersionID, rtt.GetRouteId())
		if !ok {
			return nil, errors.New("not found")
		}
		t.RouteID.Set(strconv.Itoa(rid))
		t.DirectionID.SetInt(int(rtt.GetDirectionId()))
		return &t, nil
	}
	return nil, errors.New("not found")
}

func (f *Finder) getTrip(ctx context.Context, topic string, tid string) (*pb.TripUpdate, bool) {
	if tid == "" {
		return nil, false
	}
	a, ok := f.cache.GetSource(ctx, getTopicKey(topic, "realtime_trip_updates"))
	if !ok {
		return nil, false
	}
	trip, ok := a.GetTrip(tid)
	return trip, ok
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
