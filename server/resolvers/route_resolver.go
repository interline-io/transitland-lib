package resolvers

import (
	"context"

	"github.com/interline-io/transitland-lib/server/find"
	"github.com/interline-io/transitland-lib/server/model"
)

// ROUTE

type routeResolver struct{ *Resolver }

func (r *routeResolver) Geometries(ctx context.Context, obj *model.Route, limit *int) ([]*model.RouteGeometry, error) {
	return find.For(ctx).RouteGeometriesByRouteID.Load(model.RouteGeometryParam{RouteID: obj.ID, Limit: limit})
}

func (r *routeResolver) Trips(ctx context.Context, obj *model.Route, limit *int, where *model.TripFilter) ([]*model.Trip, error) {
	return find.For(ctx).TripsByRouteID.Load(model.TripParam{RouteID: obj.ID, Limit: limit, Where: where})
}

func (r *routeResolver) Agency(ctx context.Context, obj *model.Route) (*model.Agency, error) {
	return find.For(ctx).AgenciesByID.Load(atoi(obj.AgencyID))
}

func (r *routeResolver) FeedVersion(ctx context.Context, obj *model.Route) (*model.FeedVersion, error) {
	return find.For(ctx).FeedVersionsByID.Load(obj.FeedVersionID)
}

func (r *routeResolver) RouteStops(ctx context.Context, obj *model.Route, limit *int) ([]*model.RouteStop, error) {
	return find.For(ctx).RouteStopsByRouteID.Load(model.RouteStopParam{RouteID: obj.ID, Limit: limit})
}

func (r *routeResolver) Headways(ctx context.Context, obj *model.Route, limit *int) ([]*model.RouteHeadway, error) {
	return find.For(ctx).RouteHeadwaysByRouteID.Load(model.RouteHeadwayParam{RouteID: obj.ID, Limit: limit})
}

func (r *routeResolver) RouteStopBuffer(ctx context.Context, obj *model.Route, radius *float64) (*model.RouteStopBuffer, error) {
	// TODO: remove n+1 (which is tricky, what if multiple radius specified in different parts of query)
	ents := []*model.RouteStopBuffer{}
	q := find.RouteStopBufferSelect(model.RouteStopBufferParam{Radius: radius, EntityID: obj.ID})
	find.MustSelect(model.DB, q, &ents)
	if len(ents) > 0 {
		return ents[0], nil
	}
	return nil, nil
}

// ROUTE HEADWAYS

type routeHeadwayResolver struct{ *Resolver }

func (r *routeHeadwayResolver) Stop(ctx context.Context, obj *model.RouteHeadway) (*model.Stop, error) {
	return find.For(ctx).StopsByID.Load(obj.SelectedStopID)
}

// ROUTE STOP

type routeStopResolver struct{ *Resolver }

func (r *routeStopResolver) Route(ctx context.Context, obj *model.RouteStop) (*model.Route, error) {
	return find.For(ctx).RoutesByID.Load(obj.RouteID)
}

func (r *routeStopResolver) Stop(ctx context.Context, obj *model.RouteStop) (*model.Stop, error) {
	return find.For(ctx).StopsByID.Load(obj.StopID)
}

func (r *routeStopResolver) Agency(ctx context.Context, obj *model.RouteStop) (*model.Agency, error) {
	return find.For(ctx).AgenciesByID.Load(obj.AgencyID)
}
