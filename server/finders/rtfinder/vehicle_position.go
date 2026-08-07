package rtfinder

import (
	"context"
	"sort"
	"time"

	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
)

// vehiclePositionTopicKey is the GTFS-RT source URL type carrying vehicle positions.
const vehiclePositionTopicKey = "realtime_vehicle_positions"

// vehiclePositionMatch decides whether a vehicle from a realtime feed belongs
// to the entity being asked about.
type vehiclePositionMatch func(topic string, v *pb.VehiclePosition) bool

// FindVehiclePositionsForAgency returns cached vehicle positions operated by an agency.
//
// GTFS-RT carries no agency identifier, so a vehicle is attributed by matching
// its trip descriptor's route_id against the agency's routes. A message naming
// no route can only be attributed when the realtime feed belongs to this agency
// alone: one operator on the feed, one agency in the feed version. Regional
// feeds such as 511's are associated with dozens of operators, and without that
// check every one of them would claim every vehicle in the feed.
func (f *Finder) FindVehiclePositionsForAgency(ctx context.Context, a *model.Agency, limit *int, where *model.VehiclePositionFilter) []*model.VehiclePosition {
	agencyCount, ok := f.lc.GetFeedVersionAgencyCount(ctx, a.FeedVersionID)
	singleAgency := ok && agencyCount <= 1
	var routeIds map[string]bool
	match := func(topic string, v *pb.VehiclePosition) bool {
		if singleAgency && f.feedIsExclusive(ctx, topic) {
			return true
		}
		if routeIds == nil {
			routeIds = f.lc.GetAgencyRouteIDs(ctx, a.ID)
		}
		return routeIds[v.GetTrip().GetRouteId()]
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
	var tripIds map[string]bool
	match := func(_ string, v *pb.VehiclePosition) bool {
		if rid := v.GetTrip().GetRouteId(); rid != "" {
			return rid == routeId
		}
		if tripIds == nil {
			tripIds = f.lc.GetRouteTripIDs(ctx, r.ID)
		}
		return tripIds[v.GetTrip().GetTripId()]
	}
	return limitVehiclePositions(f.findVehiclePositions(ctx, r.FeedVersionID, where, match), limit)
}

// FindVehiclePositionForTrip returns the cached vehicle position running a trip,
// matched on the trip descriptor's trip_id.
func (f *Finder) FindVehiclePositionForTrip(ctx context.Context, t *model.Trip, where *model.VehiclePositionFilter) *model.VehiclePosition {
	tripId := t.TripID.Val
	if tripId == "" {
		return nil
	}
	match := func(_ string, v *pb.VehiclePosition) bool {
		return v.GetTrip().GetTripId() == tripId
	}
	found := f.findVehiclePositions(ctx, t.FeedVersionID, where, match)
	if len(found) == 0 {
		return nil
	}
	sortVehiclePositions(found)
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
	if where != nil {
		bbox = &tlxy.BoundingBox{
			MinLon: where.Bbox.MinLon,
			MinLat: where.Bbox.MinLat,
			MaxLon: where.Bbox.MaxLon,
			MaxLat: where.Bbox.MaxLat,
		}
	}
	var ret []*model.VehiclePosition
	topics, _ := f.lc.GetFeedVersionRTFeeds(fvid)
	for _, topic := range topics {
		src, ok := f.cache.GetSource(ctx, getTopicKey(topic, vehiclePositionTopicKey))
		if !ok || src == nil {
			continue
		}
		for _, ent := range src.GetVehiclePositions() {
			if ent.Position == nil || !withinBbox(bbox, ent.Position) || !match(topic, ent.Position) {
				continue
			}
			ret = append(ret, makeVehiclePosition(ent, topic, fvid))
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

// limitVehiclePositions sorts and truncates a result set. Vehicles arrive in
// feed order, which a client polling for changes cannot rely on staying put,
// and which would otherwise decide arbitrarily what a limit cuts.
func limitVehiclePositions(ents []*model.VehiclePosition, limit *int) []*model.VehiclePosition {
	sortVehiclePositions(ents)
	if limit != nil && len(ents) > *limit {
		return ents[0:*limit]
	}
	return ents
}

func sortVehiclePositions(ents []*model.VehiclePosition) {
	sort.Slice(ents, func(i, j int) bool {
		if ents[i].RtFeedOnestopID != ents[j].RtFeedOnestopID {
			return ents[i].RtFeedOnestopID < ents[j].RtFeedOnestopID
		}
		return ents[i].ID < ents[j].ID
	})
}

func makeVehiclePosition(ent VehiclePositionEntity, rtFeedOnestopID string, fvid int) *model.VehiclePosition {
	v := ent.Position
	r := model.VehiclePosition{
		ID:              ent.ID,
		RtFeedOnestopID: rtFeedOnestopID,
		FeedVersionID:   fvid,
	}
	if p := v.Position; p != nil {
		pt := tt.NewPoint(float64(p.GetLongitude()), float64(p.GetLatitude()))
		r.Position = &pt
		if p.Bearing != nil {
			r.Bearing = pval(float64(p.GetBearing()))
		}
		if p.Speed != nil {
			r.Speed = pval(float64(p.GetSpeed()))
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
		r.CurrentStopSequence = pval(int(v.GetCurrentStopSequence()))
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
		r.OccupancyPercentage = pval(int(v.GetOccupancyPercentage()))
	}
	if v.Timestamp != nil {
		r.Timestamp = pval(time.Unix(int64(v.GetTimestamp()), 0).In(time.UTC))
	}
	return &r
}

func makeTripDescriptor(td *pb.TripDescriptor) *model.RTTripDescriptor {
	r := model.RTTripDescriptor{
		TripID:  pstr(td.GetTripId()),
		RouteID: pstr(td.GetRouteId()),
	}
	if td.DirectionId != nil {
		r.DirectionID = pval(int(td.GetDirectionId()))
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

func pval[T any](v T) *T {
	return &v
}
