package rtfinder

import (
	"context"
	"time"

	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
)

// vehiclePositionTopicKey is the GTFS-RT source URL type carrying vehicle positions.
const vehiclePositionTopicKey = "realtime_vehicle_positions"

// FindVehiclePositionsForAgency returns cached vehicle positions operated by an agency.
//
// GTFS-RT carries no agency identifier, so in a feed with more than one agency a
// vehicle is attributed by matching its trip descriptor's route_id against the
// agency's routes; vehicles reporting no route_id are not returned. In a
// single-agency feed every vehicle belongs to the agency and is returned as-is.
func (f *Finder) FindVehiclePositionsForAgency(ctx context.Context, a *model.Agency, limit *int) []*model.VehiclePosition {
	match := func(_ *pb.VehiclePosition) bool { return true }
	if f.lc.GetFeedVersionAgencyCount(a.FeedVersionID) > 1 {
		routeIds := f.lc.GetAgencyRouteIDs(a.ID)
		match = func(v *pb.VehiclePosition) bool {
			return routeIds[v.GetTrip().GetRouteId()]
		}
	}
	return limitVehiclePositions(f.findVehiclePositions(ctx, a.FeedVersionID, match), limit)
}

// FindVehiclePositionsForRoute returns cached vehicle positions running a route,
// matched on the trip descriptor's route_id.
func (f *Finder) FindVehiclePositionsForRoute(ctx context.Context, r *model.Route, limit *int) []*model.VehiclePosition {
	routeId := r.RouteID.Val
	if routeId == "" {
		return nil
	}
	match := func(v *pb.VehiclePosition) bool {
		return v.GetTrip().GetRouteId() == routeId
	}
	return limitVehiclePositions(f.findVehiclePositions(ctx, r.FeedVersionID, match), limit)
}

// FindVehiclePositionForTrip returns the cached vehicle position running a trip,
// matched on the trip descriptor's trip_id.
func (f *Finder) FindVehiclePositionForTrip(ctx context.Context, t *model.Trip) *model.VehiclePosition {
	tripId := t.TripID.Val
	if tripId == "" {
		return nil
	}
	match := func(v *pb.VehiclePosition) bool {
		return v.GetTrip().GetTripId() == tripId
	}
	found := f.findVehiclePositions(ctx, t.FeedVersionID, match)
	if len(found) == 0 {
		return nil
	}
	return found[0]
}

// findVehiclePositions collects matching vehicle positions from every realtime
// feed associated with a feed version.
func (f *Finder) findVehiclePositions(ctx context.Context, fvid int, match func(*pb.VehiclePosition) bool) []*model.VehiclePosition {
	var ret []*model.VehiclePosition
	topics, _ := f.lc.GetFeedVersionRTFeeds(fvid)
	for _, topic := range topics {
		src, ok := f.cache.GetSource(ctx, getTopicKey(topic, vehiclePositionTopicKey))
		if !ok || src == nil {
			continue
		}
		for _, v := range src.GetVehiclePositions() {
			if v == nil || !match(v) {
				continue
			}
			ret = append(ret, makeVehiclePosition(v, topic, fvid))
		}
	}
	return ret
}

func limitVehiclePositions(ents []*model.VehiclePosition, limit *int) []*model.VehiclePosition {
	if limit != nil && len(ents) > *limit {
		return ents[0:*limit]
	}
	return ents
}

func makeVehiclePosition(v *pb.VehiclePosition, feedOnestopID string, fvid int) *model.VehiclePosition {
	r := model.VehiclePosition{
		FeedOnestopID: feedOnestopID,
		FeedVersionID: fvid,
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
