package rtfinder

import (
	"context"
	"sync"
	"time"

	"github.com/interline-io/transitland-lib/internal/set"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
)

// vehiclePositionTopicKey is the GTFS-RT source URL type carrying vehicle positions.
const vehiclePositionTopicKey = string(model.FeedSourceURLTypesRealtimeVehiclePositions)

// vehiclePositionMatch decides whether a vehicle from a realtime feed belongs
// to the entity being asked about.
type vehiclePositionMatch func(topic string, v *pb.VehiclePosition) vpMatch

// vpMatch is how strongly a vehicle was attributed, which is what resolves a
// vehicle two agencies both claim to one of them.
type vpMatch int

const (
	vpNoMatch vpMatch = iota
	// Claimed only because the realtime feed belongs to a single operator.
	vpMatchByFeed
	// Matched against the feed version's own GTFS ids.
	vpMatchByEntityID
)

// FindVehiclePositionsForAgency returns cached vehicle positions operated by an agency.
//
// GTFS-RT carries no agency identifier, so a vehicle is attributed by matching
// its trip descriptor's route_id against the agency's routes. A message naming
// no route can only be attributed when the realtime feed belongs to this agency
// alone: one operator on the feed, one agency in the feed version. Regional
// feeds such as 511's are associated with dozens of operators, and without that
// check every one of them would claim every vehicle in the feed.
func (f *Finder) FindVehiclePositionsForAgency(ctx context.Context, a *model.Agency, limit *int, where *model.VehiclePositionFilter) []*model.VehiclePosition {
	// Each lookup is deferred and run at most once: most agencies a bounding
	// box resolves to have no vehicle inside it and never need any of them.
	routeIds := sync.OnceValue(func() set.Set[string] { return f.lc.GetAgencyRouteIDs(ctx, a.ID) })
	singleAgency := sync.OnceValue(func() bool {
		agencyCount, ok := f.lc.GetFeedVersionAgencyCount(ctx, a.FeedVersionID)
		return ok && agencyCount == 1
	})
	exclusive := map[string]bool{}
	match := func(topic string, v *pb.VehiclePosition) vpMatch {
		if routeIds().Contains(v.GetTrip().GetRouteId()) {
			return vpMatchByEntityID
		}
		if !singleAgency() {
			return vpNoMatch
		}
		ex, seen := exclusive[topic]
		if !seen {
			ex = f.feedIsExclusive(ctx, topic)
			exclusive[topic] = ex
		}
		if !ex {
			return vpNoMatch
		}
		return vpMatchByFeed
	}
	return limitVehiclePositions(f.findVehiclePositions(ctx, a.FeedVersionID, where, match), limit)
}

// FindVehiclePositionsForRoute returns cached vehicle positions running a route,
// matched on the trip descriptor's route_id, or on its trip_id when the message
// names no route.
func (f *Finder) FindVehiclePositionsForRoute(ctx context.Context, r *model.Route, limit *int, where *model.VehiclePositionFilter) []*model.VehiclePosition {
	routeId := r.RouteID.Val
	if routeId == "" {
		return nil
	}
	tripIds := sync.OnceValue(func() set.Set[string] { return f.lc.GetRouteTripIDs(ctx, r.ID) })
	match := func(_ string, v *pb.VehiclePosition) vpMatch {
		matched := false
		if rid := v.GetTrip().GetRouteId(); rid != "" {
			matched = rid == routeId
		} else {
			matched = tripIds().Contains(v.GetTrip().GetTripId())
		}
		if !matched {
			return vpNoMatch
		}
		return vpMatchByEntityID
	}
	return limitVehiclePositions(f.findVehiclePositions(ctx, r.FeedVersionID, where, match), limit)
}

// FindVehiclePositionForTrip returns the cached vehicle position running a trip,
// matched on the trip descriptor's trip_id. Where more than one vehicle claims
// the trip, the most recently reported wins.
func (f *Finder) FindVehiclePositionForTrip(ctx context.Context, t *model.Trip, where *model.VehiclePositionFilter) *model.VehiclePosition {
	tripId := t.TripID.Val
	if tripId == "" {
		return nil
	}
	match := func(_ string, v *pb.VehiclePosition) vpMatch {
		if v.GetTrip().GetTripId() != tripId {
			return vpNoMatch
		}
		return vpMatchByEntityID
	}
	found := model.OrderVehiclePositions(f.findVehiclePositions(ctx, t.FeedVersionID, where, match), ptr(1))
	if len(found) == 0 {
		return nil
	}
	return found[0]
}

// feedIsExclusive reports whether a realtime feed serves a single operator.
func (f *Finder) feedIsExclusive(ctx context.Context, topic string) bool {
	operatorCount, ok := f.lc.GetFeedOperatorCount(ctx, topic)
	return ok && operatorCount <= 1
}

// findVehiclePositions collects matching vehicle positions from every realtime
// feed associated with a feed version.
func (f *Finder) findVehiclePositions(ctx context.Context, fvid int, where *model.VehiclePositionFilter, match vehiclePositionMatch) []*model.VehiclePosition {
	var bbox *tlxy.BoundingBox
	if where != nil && where.Bbox != nil {
		bbox = ptr(tlxy.BoundingBox(*where.Bbox))
	}
	var ret []*model.VehiclePosition
	topics, _ := f.lc.GetFeedVersionRTFeeds(ctx, fvid)
	for _, topic := range topics {
		src, ok := f.cache.GetSource(ctx, getTopicKey(topic, vehiclePositionTopicKey))
		if !ok || src == nil {
			continue
		}
		for _, ent := range src.GetVehiclePositions() {
			if ent.Position == nil || !withinBbox(bbox, ent.Position) {
				continue
			}
			m := match(topic, ent.Position)
			if m == vpNoMatch {
				continue
			}
			ret = append(ret, makeVehiclePosition(ent, topic, fvid, m == vpMatchByEntityID))
		}
	}
	return ret
}

// withinBbox tests the raw message, so vehicles outside the search area are
// dropped before anything is built for them.
func withinBbox(bbox *tlxy.BoundingBox, v *pb.VehiclePosition) bool {
	if bbox == nil {
		return true
	}
	p := v.Position
	if p == nil {
		return false
	}
	return bbox.Contains(tlxy.Point{Lon: float64(p.GetLongitude()), Lat: float64(p.GetLatitude())})
}

// limitVehiclePositions orders and truncates a result set, skipping both when
// there is no limit to apply and the caller will order the result itself.
func limitVehiclePositions(ents []*model.VehiclePosition, limit *int) []*model.VehiclePosition {
	if limit == nil {
		return ents
	}
	return model.OrderVehiclePositions(ents, limit)
}

func makeVehiclePosition(ent VehiclePositionEntity, rtFeedOnestopID string, fvid int, matchedByEntityID bool) *model.VehiclePosition {
	v := ent.Position
	r := model.VehiclePosition{
		ID:                ent.ID,
		RtFeedOnestopID:   rtFeedOnestopID,
		FeedVersionID:     fvid,
		MatchedByEntityID: matchedByEntityID,
	}
	if p := v.Position; p != nil {
		pt := tt.NewPoint(float64(p.GetLongitude()), float64(p.GetLatitude()))
		r.Position = &pt
		if p.Bearing != nil {
			r.Bearing = ptr(float64(p.GetBearing()))
		}
		if p.Speed != nil {
			r.Speed = ptr(float64(p.GetSpeed()))
		}
	}
	if vd := v.Vehicle; vd != nil {
		r.Vehicle = &model.RTVehicleDescriptor{
			ID:           pstr(vd.GetId()),
			Label:        pstr(vd.GetLabel()),
			LicensePlate: pstr(vd.GetLicensePlate()),
		}
	}
	if td := v.Trip; td != nil {
		r.TripDescriptor = makeTripDescriptor(td)
	}
	if v.StopId != nil {
		r.StopID = pstr(v.GetStopId())
	}
	if v.CurrentStopSequence != nil {
		r.CurrentStopSequence = ptr(int(v.GetCurrentStopSequence()))
	}
	if v.CurrentStatus != nil {
		r.CurrentStatus = pstr(v.CurrentStatus.String())
	}
	if v.CongestionLevel != nil {
		r.CongestionLevel = pstr(v.CongestionLevel.String())
	}
	if v.OccupancyStatus != nil {
		r.OccupancyStatus = pstr(v.OccupancyStatus.String())
	}
	if v.OccupancyPercentage != nil {
		r.OccupancyPercentage = ptr(int(v.GetOccupancyPercentage()))
	}
	if v.Timestamp != nil {
		r.Timestamp = ptr(time.Unix(int64(v.GetTimestamp()), 0).In(time.UTC))
	}
	return &r
}

func makeTripDescriptor(td *pb.TripDescriptor) *model.RTTripDescriptor {
	r := model.RTTripDescriptor{
		TripID:  pstr(td.GetTripId()),
		RouteID: pstr(td.GetRouteId()),
	}
	if td.DirectionId != nil {
		r.DirectionID = ptr(int(td.GetDirectionId()))
	}
	if td.ScheduleRelationship != nil {
		r.ScheduleRelationship = pstr(td.ScheduleRelationship.String())
	}
	// Times and dates are free-form strings in GTFS-RT; skip values that do not parse.
	if v := td.GetStartTime(); v != "" {
		if s, err := tt.NewSecondsFromString(v); err == nil {
			r.StartTime = &s
		}
	}
	if v := td.GetStartDate(); v != "" {
		if d, err := tt.ParseDate(v); err == nil {
			r.StartDate = &d
		}
	}
	return &r
}

func ptr[T any](v T) *T {
	return &v
}
