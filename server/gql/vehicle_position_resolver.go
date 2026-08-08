package gql

import (
	"context"

	"github.com/interline-io/log"
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
func checkVehiclePositionGeo(cfg model.Config, where *model.VehiclePositionFilter) error {
	if where == nil {
		return nil
	}
	return checkGeo(cfg.MaxRadius, nil, where.Bbox)
}

// vehiclePositionLimit applies this field's own default in place of the
// API-wide one, which a map viewport would overrun immediately.
func vehiclePositionLimit(limit *int) *int {
	if limit == nil {
		a := RESOLVER_VEHICLE_POSITION_DEFAULT_LIMIT
		return &a
	}
	return resolverCheckLimitMax(limit, RESOLVER_VEHICLE_POSITION_MAXLIMIT)
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
	if err := checkVehiclePositionGeo(cfg, &where); err != nil {
		return nil, err
	}
	agencies, err := vehiclePositionAgencies(ctx, where)
	if err != nil {
		return nil, err
	}

	// Each agency is asked without a limit, because which vehicles survive one
	// is decided across the whole viewport rather than per agency. Agencies
	// whose static feeds share a realtime feed each see the same vehicles, so
	// one bus can arrive here once per agency before being deduplicated.
	var found []*model.VehiclePosition
	for _, agency := range agencies {
		found = append(found, cfg.RTFinder.FindVehiclePositionsForAgency(ctx, agency, nil, &where)...)
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
		// The result is missing whatever agencies fell off the end, which no
		// part of the response says.
		log.For(ctx).Warn().Int("scope_limit", RESOLVER_VEHICLE_POSITION_SCOPE_MAXLIMIT).Msg("vehicle_positions: agency scope limit reached, results are incomplete")
	}
	return agencies, err
}

func (r *agencyResolver) VehiclePositions(ctx context.Context, obj *model.Agency, limit *int, where *model.VehiclePositionFilter) ([]*model.VehiclePosition, error) {
	cfg := model.ForContext(ctx)
	if err := checkVehiclePositionGeo(cfg, where); err != nil {
		return nil, err
	}
	return cfg.RTFinder.FindVehiclePositionsForAgency(ctx, obj, vehiclePositionLimit(limit), where), nil
}

func (r *routeResolver) VehiclePositions(ctx context.Context, obj *model.Route, limit *int, where *model.VehiclePositionFilter) ([]*model.VehiclePosition, error) {
	cfg := model.ForContext(ctx)
	if err := checkVehiclePositionGeo(cfg, where); err != nil {
		return nil, err
	}
	return cfg.RTFinder.FindVehiclePositionsForRoute(ctx, obj, vehiclePositionLimit(limit), where), nil
}

func (r *tripResolver) VehiclePosition(ctx context.Context, obj *model.Trip, where *model.VehiclePositionFilter) (*model.VehiclePosition, error) {
	cfg := model.ForContext(ctx)
	if err := checkVehiclePositionGeo(cfg, where); err != nil {
		return nil, err
	}
	return cfg.RTFinder.FindVehiclePositionForTrip(ctx, obj, where), nil
}
