package rtfinder

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/clock"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tldb"
)

// Cache provides a method for looking up and listening for changed RT data
type Cache interface {
	AddFeedMessage(context.Context, string, *pb.FeedMessage) error
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

func NewFinder(cache Cache, db tldb.Ext) *Finder {
	return &Finder{
		Clock: &clock.Real{},
		cache: cache,
		lc:    newLookupCache(db),
	}
}

func (f *Finder) AddData(ctx context.Context, topic string, data []byte) error {
	return f.cache.AddData(ctx, topic, data)
}

func (f *Finder) GetGtfsTripID(ctx context.Context, id int) (string, bool) {
	return f.lc.GetGtfsTripID(id)
}

func (f *Finder) StopTimezone(ctx context.Context, id int, known string) (*time.Location, bool) {
	return f.lc.StopTimezone(ctx, id, known)
}

func (f *Finder) FeedVersionTimezone(ctx context.Context, fvid int) (*time.Location, bool) {
	return f.lc.FeedVersionTimezone(ctx, fvid)
}

func (f *Finder) FindTrip(ctx context.Context, t *model.Trip) *pb.TripUpdate {
	topics, _ := f.lc.GetFeedVersionRTFeeds(t.FeedVersionID)
	for _, topic := range topics {
		if a, ok := f.getTrip(ctx, topic, t.TripID.Val); ok {
			return a
		}
	}
	return nil
}

func (f *Finder) FindAlertsForTrip(ctx context.Context, t *model.Trip, limit *int, active *bool) []*model.Alert {
	foundAlerts := []*model.Alert{}
	topics, _ := f.lc.GetFeedVersionRTFeeds(t.FeedVersionID)
	tnow := f.Clock.Now()
	for _, topic := range topics {
		a, ok := f.cache.GetSource(ctx, getTopicKey(topic, "realtime_alerts"))
		if a == nil || !ok {
			return foundAlerts
		}
		for _, alert := range a.alerts {
			if alert == nil {
				continue
			}
			if !checkAlertActivePeriod(tnow, active, alert) {
				continue
			}
			found := false
			for _, s := range alert.GetInformedEntity() {
				// trip must match
				// route, stop, agency are not checked
				if s == nil || s.Trip == nil {
					continue
				}
				if s.Trip.GetTripId() == t.TripID.Val {
					found = true
				}
			}
			if found {
				foundAlerts = append(foundAlerts, makeAlert(alert))
			}
		}
	}
	return limitAlerts(foundAlerts, limit)
}

func (f *Finder) FindAlertsForRoute(ctx context.Context, t *model.Route, limit *int, active *bool) []*model.Alert {
	foundAlerts := []*model.Alert{}
	topics, _ := f.lc.GetFeedVersionRTFeeds(t.FeedVersionID)
	tnow := f.Clock.Now()
	for _, topic := range topics {
		a, ok := f.cache.GetSource(ctx, getTopicKey(topic, "realtime_alerts"))
		if a == nil || !ok {
			continue
		}
		for _, alert := range a.alerts {
			if !checkAlertActivePeriod(tnow, active, alert) {
				continue
			}
			if alert == nil {
				continue
			}
			found := false
			for _, s := range alert.GetInformedEntity() {
				// trip and stop must be empty
				// route must match
				if s == nil || s.Trip != nil || s.GetStopId() != "" {
					continue
				}
				if s.GetRouteId() == t.RouteID.Val {
					found = true
				}
			}
			if found {
				foundAlerts = append(foundAlerts, makeAlert(alert))
			}
		}
	}
	return limitAlerts(foundAlerts, limit)
}

func (f *Finder) GetMessage(ctx context.Context, topic string, topicKey string) (*pb.FeedMessage, bool) {
	tk := getTopicKey(topic, topicKey)
	a, ok := f.cache.GetSource(ctx, tk)
	if a != nil && ok {
		return a.msg, ok
	}
	return nil, false
}

func (f *Finder) FindAlertsForAgency(ctx context.Context, t *model.Agency, limit *int, active *bool) []*model.Alert {
	foundAlerts := []*model.Alert{}
	topics, _ := f.lc.GetFeedVersionRTFeeds(t.FeedVersionID)
	tnow := f.Clock.Now()
	for _, topic := range topics {
		a, ok := f.cache.GetSource(ctx, getTopicKey(topic, "realtime_alerts"))
		if a == nil || !ok {
			continue
		}
		for _, alert := range a.alerts {
			if alert == nil {
				continue
			}
			if !checkAlertActivePeriod(tnow, active, alert) {
				continue
			}
			found := false
			for _, s := range alert.GetInformedEntity() {
				// trip, route, stop must be empty
				// agency must match
				if s == nil || s.Trip != nil || s.GetRouteId() != "" || s.GetStopId() != "" {
					continue
				}
				if s.GetAgencyId() == t.AgencyID.Val {
					found = true
				}
			}
			if found {
				foundAlerts = append(foundAlerts, makeAlert(alert))
			}
		}
	}
	return limitAlerts(foundAlerts, limit)
}

func (f *Finder) FindAlertsForStop(ctx context.Context, t *model.Stop, limit *int, active *bool) []*model.Alert {
	foundAlerts := []*model.Alert{}
	topics, _ := f.lc.GetFeedVersionRTFeeds(t.FeedVersionID)
	tnow := f.Clock.Now()
	for _, topic := range topics {
		a, ok := f.cache.GetSource(ctx, getTopicKey(topic, "realtime_alerts"))
		if a == nil || !ok {
			continue
		}
		for _, alert := range a.alerts {
			if !checkAlertActivePeriod(tnow, active, alert) {
				continue
			}
			if alert == nil {
				continue
			}
			found := false
			for _, s := range alert.GetInformedEntity() {
				// agency, route can be anything
				// trip must be empty
				if s == nil || s.Trip != nil {
					continue
				}
				if s.GetStopId() == t.StopID.Val {
					found = true
				}
			}
			if found {
				foundAlerts = append(foundAlerts, makeAlert(alert))
			}
		}
	}
	return limitAlerts(foundAlerts, limit)
}

func (f *Finder) FindStopTimeUpdate(ctx context.Context, t *model.Trip, st *model.StopTime) (*model.RTStopTimeUpdate, bool) {
	tid := t.TripID
	seq := st.StopSequence.Int()
	topics, _ := f.lc.GetFeedVersionRTFeeds(t.FeedVersionID)
	// Attempt to match on stop sequence
	for _, topic := range topics {
		// Match on trip
		rtTrip, rtok := f.getTrip(ctx, topic, tid.Val)
		if !rtok {
			continue
		}
		// Match on stop sequence
		for _, ste := range rtTrip.StopTimeUpdate {
			if int(ste.GetStopSequence()) == seq {
				log.For(ctx).Trace().Str("trip_id", t.TripID.Val).Int("seq", seq).Msgf("found stop time update on trip_id/stop_sequence")
				return &model.RTStopTimeUpdate{TripUpdate: rtTrip, StopTimeUpdate: ste}, true
			}
		}
	}
	// Attempt to match on stop id
	for _, topic := range topics {
		// Match on trip
		rtTrip, rtok := f.getTrip(ctx, topic, tid.Val)
		if !rtok {
			continue
		}
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
	topics, _ := f.lc.GetFeedVersionRTFeeds(t.FeedVersionID)
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

func makeAlert(a *pb.Alert) *model.Alert {
	r := model.Alert{}
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
			rttr.Start = &v
		}
		r.ActivePeriod = append(r.ActivePeriod, &rttr)
	}
	r.HeaderText = newTranslation(a.HeaderText)
	r.DescriptionText = newTranslation(a.DescriptionText)
	r.TtsHeaderText = newTranslation(a.TtsHeaderText)
	r.TtsDescriptionText = newTranslation(a.TtsDescriptionText)
	r.URL = newTranslation(a.Url)
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

func getTopicKey(topic string, t string) string {
	return fmt.Sprintf("rtdata:%s:%s", topic, t)
}

func copyPtr[T any, PT *T](v PT) PT {
	if v == nil {
		return nil
	}
	a := *v
	return &a
}
