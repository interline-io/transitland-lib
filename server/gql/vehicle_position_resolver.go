package gql

import (
	"context"
	"errors"
	"sort"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlxy"
)

const (
	// A large operator runs well over the default 1,000 maximum, and vehicles
	// are served from the realtime cache rather than the database.
	RESOLVER_VEHICLE_POSITION_MAXLIMIT = 10_000
	// RESOLVER_VEHICLE_POSITION_SCOPE_MAXLIMIT bounds how many agencies and
	// routes a single vehicle_positions query resolves its filter to.
	RESOLVER_VEHICLE_POSITION_SCOPE_MAXLIMIT = 1_000
)

type vehiclePositionResolver struct{ *Resolver }

// The trip descriptor and the ids on it are optional: a GTFS-RT message may
// identify only the vehicle.
func rtTripID(td *model.RTTripDescriptor) string {
	if td == nil || td.TripID == nil {
		return ""
	}
	return *td.TripID
}

func rtRouteID(td *model.RTTripDescriptor) string {
	if td == nil || td.RouteID == nil {
		return ""
	}
	return *td.RouteID
}

func rtVehicleID(vd *model.RTVehicleDescriptor) string {
	if vd == nil || vd.ID == nil {
		return ""
	}
	return *vd.ID
}

func (r *vehiclePositionResolver) Trip(ctx context.Context, obj *model.VehiclePosition) (*model.Trip, error) {
	tripId := rtTripID(obj.TripDescriptor)
	if tripId == "" {
		return nil, nil
	}
	return LoaderFor(ctx).TripsByFeedVersionTripIDs.Load(ctx, model.FVEntityID{
		FeedVersionID: obj.FeedVersionID,
		EntityID:      tripId,
	})()
}

func (r *vehiclePositionResolver) Route(ctx context.Context, obj *model.VehiclePosition) (*model.Route, error) {
	if routeId := rtRouteID(obj.TripDescriptor); routeId != "" {
		return LoaderFor(ctx).RoutesByFeedVersionRouteIDs.Load(ctx, model.FVEntityID{
			FeedVersionID: obj.FeedVersionID,
			EntityID:      routeId,
		})()
	}
	// Many producers name only the trip; fall back to the matched trip's route.
	trip, err := r.Trip(ctx, obj)
	if err != nil || trip == nil {
		return nil, err
	}
	return LoaderFor(ctx).RoutesByIDs.Load(ctx, trip.RouteID.Int())()
}

func (r *vehiclePositionResolver) Stop(ctx context.Context, obj *model.VehiclePosition) (*model.Stop, error) {
	if obj.StopID == nil || *obj.StopID == "" {
		return nil, nil
	}
	return LoaderFor(ctx).StopsByFeedVersionStopIDs.Load(ctx, model.FVEntityID{
		FeedVersionID: obj.FeedVersionID,
		EntityID:      *obj.StopID,
	})()
}

func (r *queryResolver) VehiclePositions(ctx context.Context, limit *int, where model.VehiclePositionFilter) ([]*model.VehiclePosition, error) {
	ctx = addMetric(ctx, "vehiclePositions")
	cfg := model.ForContext(ctx)
	byAgency := where.Bbox != nil || len(where.AgencyOnestopIds) > 0 || len(where.FeedOnestopIds) > 0
	if !byAgency && len(where.RouteOnestopIds) == 0 {
		return nil, errors.New("vehicle_positions requires at least one of bbox, agency_onestop_ids, feed_onestop_ids, or route_onestop_ids")
	}
	if err := checkGeo(cfg.MaxRadius, nil, where.Bbox); err != nil {
		return nil, err
	}

	// Routes named by Onestop ID restrict which vehicles are returned, and
	// select the agencies to read when nothing else does.
	var routeIds map[string]bool
	var routeAgencyIds []int
	if len(where.RouteOnestopIds) > 0 {
		routes, err := cfg.Finder.FindRoutes(ctx, ptr(RESOLVER_VEHICLE_POSITION_SCOPE_MAXLIMIT), nil, nil, &model.RouteFilter{OnestopIds: where.RouteOnestopIds})
		if err != nil {
			return nil, err
		}
		routeIds = map[string]bool{}
		seenAgency := map[int]bool{}
		for _, route := range routes {
			routeIds[route.RouteID.Val] = true
			if agencyId := route.AgencyID.Int(); agencyId > 0 && !seenAgency[agencyId] {
				seenAgency[agencyId] = true
				routeAgencyIds = append(routeAgencyIds, agencyId)
			}
		}
	}

	var agencies []*model.Agency
	var err error
	if byAgency {
		agencies, err = vehiclePositionAgencies(ctx, where)
	} else {
		agencies, err = vehiclePositionRouteAgencies(ctx, routeAgencyIds)
	}
	if err != nil {
		return nil, err
	}

	tripIds := map[string]bool{}
	for _, tripId := range where.TripIds {
		tripIds[tripId] = true
	}
	var bbox *tlxy.BoundingBox
	if where.Bbox != nil {
		bbox = &tlxy.BoundingBox{
			MinLon: where.Bbox.MinLon,
			MinLat: where.Bbox.MinLat,
			MaxLon: where.Bbox.MaxLon,
			MaxLat: where.Bbox.MaxLat,
		}
	}

	var ret []*model.VehiclePosition
	for _, agency := range agencies {
		for _, ent := range cfg.RTFinder.FindVehiclePositionsForAgency(ctx, agency, nil) {
			if routeIds != nil && !routeIds[rtRouteID(ent.TripDescriptor)] {
				continue
			}
			if len(tripIds) > 0 && !tripIds[rtTripID(ent.TripDescriptor)] {
				continue
			}
			if bbox != nil {
				if ent.Position == nil || !ent.Position.Valid || !bbox.Contains(ent.Position.ToPoint()) {
					continue
				}
			}
			ret = append(ret, ent)
		}
	}
	// Vehicles arrive in feed order, which a client polling for changes cannot
	// rely on staying put.
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].FeedOnestopID != ret[j].FeedOnestopID {
			return ret[i].FeedOnestopID < ret[j].FeedOnestopID
		}
		if a, b := rtVehicleID(ret[i].Vehicle), rtVehicleID(ret[j].Vehicle); a != b {
			return a < b
		}
		return rtTripID(ret[i].TripDescriptor) < rtTripID(ret[j].TripDescriptor)
	})
	if lim := resolverCheckLimitMax(limit, RESOLVER_VEHICLE_POSITION_MAXLIMIT); lim != nil && len(ret) > *lim {
		ret = ret[0:*lim]
	}
	return ret, nil
}

// vehiclePositionAgencies resolves the agency, feed and bbox filters to the
// agencies whose realtime feeds are read.
func vehiclePositionAgencies(ctx context.Context, where model.VehiclePositionFilter) ([]*model.Agency, error) {
	agencyFilter := &model.AgencyFilter{
		OnestopIds:     where.AgencyOnestopIds,
		FeedOnestopIds: where.FeedOnestopIds,
	}
	if where.Bbox != nil {
		agencyFilter.Location = &model.AgencyLocationFilter{Bbox: where.Bbox}
	}
	return model.ForContext(ctx).Finder.FindAgencies(ctx, ptr(RESOLVER_VEHICLE_POSITION_SCOPE_MAXLIMIT), nil, nil, agencyFilter)
}

// vehiclePositionRouteAgencies loads the agencies operating the routes a
// route-only filter resolved to.
func vehiclePositionRouteAgencies(ctx context.Context, agencyIds []int) ([]*model.Agency, error) {
	if len(agencyIds) == 0 {
		return nil, nil
	}
	agencies, errs := model.ForContext(ctx).Finder.AgenciesByIDs(ctx, agencyIds)
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	var ret []*model.Agency
	for _, agency := range agencies {
		if agency != nil {
			ret = append(ret, agency)
		}
	}
	return ret, nil
}

func (r *agencyResolver) VehiclePositions(ctx context.Context, obj *model.Agency, limit *int) ([]*model.VehiclePosition, error) {
	return model.ForContext(ctx).RTFinder.FindVehiclePositionsForAgency(ctx, obj, resolverCheckLimitMax(limit, RESOLVER_VEHICLE_POSITION_MAXLIMIT)), nil
}

func (r *routeResolver) VehiclePositions(ctx context.Context, obj *model.Route, limit *int) ([]*model.VehiclePosition, error) {
	return model.ForContext(ctx).RTFinder.FindVehiclePositionsForRoute(ctx, obj, resolverCheckLimitMax(limit, RESOLVER_VEHICLE_POSITION_MAXLIMIT)), nil
}

func (r *tripResolver) VehiclePosition(ctx context.Context, obj *model.Trip) (*model.VehiclePosition, error) {
	return model.ForContext(ctx).RTFinder.FindVehiclePositionForTrip(ctx, obj), nil
}
