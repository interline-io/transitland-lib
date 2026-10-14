package gql

import (
	"context"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/model"
)

const (
	// Vehicles are served from the realtime cache rather than the database, so
	// the ceiling is a backstop rather than a budget: a client asking for
	// everything in a viewport should get everything.
	RESOLVER_VEHICLE_POSITION_MAXLIMIT = 100_000
	// RESOLVER_VEHICLE_POSITION_DEFAULT_LIMIT is what an unlimited query
	// returns. The API-wide default of 100 is far too small for a map viewport.
	RESOLVER_VEHICLE_POSITION_DEFAULT_LIMIT = 1_000
	// RESOLVER_VEHICLE_POSITION_SCOPE_MAXLIMIT bounds how many agencies a
	// single vehicle_positions search resolves its bounding box to.
	RESOLVER_VEHICLE_POSITION_SCOPE_MAXLIMIT = 1_000
)

// checkVehiclePositionGeo bounds the search area on the nested fields, where
// the filter is optional.
func checkVehiclePositionGeo(ctx context.Context, where *model.VehiclePositionFilter) error {
	if where == nil {
		return nil
	}
	return checkGeo(model.ForContext(ctx).MaxRadius, nil, where.Bbox)
}

func vehiclePositionLimit(limit *int) *int {
	return resolverCheckLimitDefault(limit, RESOLVER_VEHICLE_POSITION_DEFAULT_LIMIT, RESOLVER_VEHICLE_POSITION_MAXLIMIT)
}

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

func (r *vehiclePositionResolver) Trip(ctx context.Context, obj *model.VehiclePosition) (*model.Trip, error) {
	td := obj.TripDescriptor
	tripId := rtTripID(td)
	if tripId == "" {
		return nil, nil
	}
	// A DUPLICATED vehicle runs a new copy of the trip it names, which has no
	// trip of its own yet.
	rel := ""
	if td.ScheduleRelationship != nil {
		rel = *td.ScheduleRelationship
	}
	if rel == pb.TripDescriptor_DUPLICATED.String() {
		return nil, nil
	}
	trip, err := LoaderFor(ctx).TripsByFeedVersionTripIDs.Load(ctx, model.FVEntityID{
		FeedVersionID: obj.FeedVersionID,
		EntityID:      tripId,
	})()
	if err != nil {
		return nil, err
	}
	if rel == pb.TripDescriptor_ADDED.String() {
		// Naming a scheduled trip, an ADDED vehicle runs a copy of it, the form
		// duplicates took before DUPLICATED. Otherwise its trip was added in real
		// time, and is on the route the vehicle names.
		if trip != nil {
			return nil, nil
		}
		ptd := &pb.TripDescriptor{TripId: td.TripID, RouteId: td.RouteID}
		if td.DirectionID != nil {
			ptd.DirectionId = ptr(uint32(*td.DirectionID))
		}
		if trip, err = model.ForContext(ctx).RTFinder.MakeTrip(ctx, obj.FeedVersionID, ptd); err != nil {
			log.For(ctx).Debug().Err(err).Str("trip_id", tripId).Str("route_id", rtRouteID(td)).Msg("added vehicle's trip left out: route not in feed version")
			return nil, nil
		}
	}
	// The run the vehicle names; one naming none is the trip's current run.
	if td.StartDate != nil {
		return tripRun(trip, *td.StartDate), nil
	}
	return trip, nil
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
	if err := checkVehiclePositionGeo(ctx, &where); err != nil {
		return nil, err
	}
	agencies, err := vehiclePositionAgencies(ctx, where)
	if err != nil {
		return nil, err
	}

	// Each agency is asked without a limit: which vehicles survive one is
	// decided across the whole viewport rather than per agency.
	rtFinder := model.ForContext(ctx).RTFinder
	var found []*model.VehiclePosition
	for _, agency := range agencies {
		found = append(found, rtFinder.FindVehiclePositionsForAgency(ctx, agency, nil, &where)...)
	}
	return model.OrderVehiclePositions(found, vehiclePositionLimit(limit)), nil
}

// vehiclePositionAgencies resolves the search area to the agencies whose
// realtime feeds are read. An agency's geometry is the hull of its stops, so
// any agency with a vehicle inside the box also intersects the box.
func vehiclePositionAgencies(ctx context.Context, where model.VehiclePositionFilter) ([]*model.Agency, error) {
	agencyFilter := &model.AgencyFilter{
		Location: &model.AgencyLocationFilter{Bbox: where.Bbox},
	}
	agencies, err := model.ForContext(ctx).Finder.FindAgencies(ctx, ptr(RESOLVER_VEHICLE_POSITION_SCOPE_MAXLIMIT), nil, nil, agencyFilter)
	if len(agencies) >= RESOLVER_VEHICLE_POSITION_SCOPE_MAXLIMIT {
		log.For(ctx).Warn().Int("scope_limit", RESOLVER_VEHICLE_POSITION_SCOPE_MAXLIMIT).Msg("vehicle_positions: agency scope limit reached, results are incomplete")
	}
	return agencies, err
}

func (r *agencyResolver) VehiclePositions(ctx context.Context, obj *model.Agency, limit *int, where *model.VehiclePositionFilter) ([]*model.VehiclePosition, error) {
	if err := checkVehiclePositionGeo(ctx, where); err != nil {
		return nil, err
	}
	return model.ForContext(ctx).RTFinder.FindVehiclePositionsForAgency(ctx, obj, vehiclePositionLimit(limit), where), nil
}

func (r *routeResolver) VehiclePositions(ctx context.Context, obj *model.Route, limit *int, where *model.VehiclePositionFilter) ([]*model.VehiclePosition, error) {
	if err := checkVehiclePositionGeo(ctx, where); err != nil {
		return nil, err
	}
	return model.ForContext(ctx).RTFinder.FindVehiclePositionsForRoute(ctx, obj, vehiclePositionLimit(limit), where), nil
}

func (r *tripResolver) VehiclePosition(ctx context.Context, obj *model.Trip, where *model.VehiclePositionFilter) (*model.VehiclePosition, error) {
	if err := checkVehiclePositionGeo(ctx, where); err != nil {
		return nil, err
	}
	return model.ForContext(ctx).RTFinder.FindVehiclePositionForTrip(ctx, obj, where), nil
}
