package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
)

// ROUTE

type routeResolver struct{ *Resolver }

func (r *routeResolver) Cursor(ctx context.Context, obj *model.Route) (*model.Cursor, error) {
	c := model.NewCursor(obj.FeedVersionID, obj.ID)
	return &c, nil
}

func (r *routeResolver) Geometry(ctx context.Context, obj *model.Route) (*tt.Geometry, error) {
	if obj.Geometry.Valid {
		return &obj.Geometry, nil
	}
	// Defer geometry loading
	geoms, err := LoaderFor(ctx).RouteGeometriesByRouteIDs.Load(ctx, routeGeometryLoaderParam{RouteID: obj.ID})()
	if err != nil {
		return nil, err
	}
	if len(geoms) > 0 {
		return geoms[0].CombinedGeometry, nil
	}
	return nil, nil
}

func (r *routeResolver) Geometries(ctx context.Context, obj *model.Route, limit *int) ([]*model.RouteGeometry, error) {
	return LoaderFor(ctx).RouteGeometriesByRouteIDs.Load(ctx, routeGeometryLoaderParam{RouteID: obj.ID, Limit: resolverCheckLimit(limit)})()
}

func (r *routeResolver) Trips(ctx context.Context, obj *model.Route, limit *int, where *model.TripFilter) ([]*model.Trip, error) {
	return LoaderFor(ctx).TripsByRouteIDs.Load(ctx, tripLoaderParam{RouteID: obj.ID, FeedVersionID: obj.FeedVersionID, Limit: resolverCheckLimit(limit), Where: where})()
}

func (r *routeResolver) Agency(ctx context.Context, obj *model.Route) (*model.Agency, error) {
	return LoaderFor(ctx).AgenciesByIDs.Load(ctx, obj.AgencyID.Int())()
}

func (r *routeResolver) FeedVersion(ctx context.Context, obj *model.Route) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

func (r *routeResolver) Stops(ctx context.Context, obj *model.Route, limit *int, where *model.StopFilter) ([]*model.Stop, error) {
	return LoaderFor(ctx).StopsByRouteIDs.Load(ctx, stopLoaderParam{RouteID: obj.ID, Limit: resolverCheckLimit(limit), Where: where})()
}

func (r *routeResolver) RouteStops(ctx context.Context, obj *model.Route, limit *int) ([]*model.RouteStop, error) {
	return LoaderFor(ctx).RouteStopsByRouteIDs.Load(ctx, routeStopLoaderParam{RouteID: obj.ID, Limit: resolverCheckLimit(limit)})()
}

func (r *routeResolver) Headways(ctx context.Context, obj *model.Route, limit *int) ([]*model.RouteHeadway, error) {
	return LoaderFor(ctx).RouteHeadwaysByRouteIDs.Load(ctx, routeHeadwayLoaderParam{RouteID: obj.ID, Limit: resolverCheckLimit(limit)})()
}

func (r *routeResolver) RouteStopBuffer(ctx context.Context, obj *model.Route, radius *float64) (*model.RouteStopBuffer, error) {
	// TODO: remove n+1 (which is tricky, what if multiple radius specified in different parts of query)
	ents, err := model.ForContext(ctx).Finder.RouteStopBuffer(ctx, nil, radius, obj.ID)
	if err != nil {
		return nil, err
	}
	if len(ents) > 0 {
		return ents[0], nil
	}
	return nil, nil
}

func (r *routeResolver) Alerts(ctx context.Context, obj *model.Route, active *bool, limit *int) ([]*model.Alert, error) {
	return model.ForContext(ctx).RTFinder.FindAlertsForRoute(ctx, obj, resolverCheckLimit(limit), active), nil
}

func (r *routeResolver) Patterns(ctx context.Context, obj *model.Route) ([]*model.RouteStopPattern, error) {
	return LoaderFor(ctx).RouteStopPatternsByRouteIDs.Load(ctx, routeStopPatternLoaderParam{RouteID: obj.ID})()
}

func (r *routeResolver) RouteAttribute(ctx context.Context, obj *model.Route) (*model.RouteAttribute, error) {
	return LoaderFor(ctx).RouteAttributesByRouteIDs.Load(ctx, obj.ID)()
}

func (r *routeResolver) SegmentPatterns(ctx context.Context, obj *model.Route, limit *int, where *model.SegmentPatternFilter) ([]*model.SegmentPattern, error) {
	return LoaderFor(ctx).SegmentPatternsByRouteIDs.Load(ctx, segmentPatternLoaderParam{RouteID: obj.ID, Where: where, Limit: resolverCheckLimit(limit)})()
}

func (r *routeResolver) Segments(ctx context.Context, obj *model.Route, limit *int, where *model.SegmentFilter) ([]*model.Segment, error) {
	return LoaderFor(ctx).SegmentsByRouteIDs.Load(ctx, segmentLoaderParam{RouteID: obj.ID, Where: where, Limit: resolverCheckLimit(limit)})()
}

// ROUTE HEADWAYS

type routeHeadwayResolver struct{ *Resolver }

func (r *routeHeadwayResolver) Stop(ctx context.Context, obj *model.RouteHeadway) (*model.Stop, error) {
	return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.SelectedStopID)()
}

func (r *routeHeadwayResolver) Departures(ctx context.Context, obj *model.RouteHeadway) ([]*tt.Seconds, error) {
	var ret []*tt.Seconds
	for _, v := range obj.DepartureInts.Val {
		w := tt.NewSeconds(int(v))
		ret = append(ret, &w)
	}
	return ret, nil
}

// ROUTE STOP

type routeStopResolver struct{ *Resolver }

func (r *routeStopResolver) Route(ctx context.Context, obj *model.RouteStop) (*model.Route, error) {
	return LoaderFor(ctx).RoutesByIDs.Load(ctx, obj.RouteID)()
}

func (r *routeStopResolver) Stop(ctx context.Context, obj *model.RouteStop) (*model.Stop, error) {
	return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.StopID)()
}

func (r *routeStopResolver) Agency(ctx context.Context, obj *model.RouteStop) (*model.Agency, error) {
	return LoaderFor(ctx).AgenciesByIDs.Load(ctx, obj.AgencyID)()
}

// ROUTE PATTERN

type routePatternResolver struct{ *Resolver }

func (r *routePatternResolver) Trips(ctx context.Context, obj *model.RouteStopPattern, limit *int) ([]*model.Trip, error) {
	// TODO: N+1 query
	trips, err := model.ForContext(ctx).Finder.FindTrips(ctx, resolverCheckLimit(limit), nil, nil, &model.TripFilter{StopPatternID: &obj.StopPatternID, RouteIds: []int{obj.RouteID}})
	return trips, err
}

// func (r *routePatternResolver) Stops(ctx context.Context, obj *model.RouteStopPattern) ([]*model.Stop, error) {
// 	return nil, nil
// }
