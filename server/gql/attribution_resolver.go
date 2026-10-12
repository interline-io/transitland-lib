package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
)

// ATTRIBUTION

type attributionResolver struct{ *Resolver }

func (r *attributionResolver) FeedVersion(ctx context.Context, obj *model.Attribution) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

func (r *attributionResolver) Agency(ctx context.Context, obj *model.Attribution) (*model.Agency, error) {
	if !obj.AgencyID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).AgenciesByIDs.Load(ctx, obj.AgencyID.Int())()
}

func (r *attributionResolver) Route(ctx context.Context, obj *model.Attribution) (*model.Route, error) {
	if !obj.RouteID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).RoutesByIDs.Load(ctx, obj.RouteID.Int())()
}

func (r *attributionResolver) Trip(ctx context.Context, obj *model.Attribution) (*model.Trip, error) {
	if !obj.TripID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).TripsByIDs.Load(ctx, obj.TripID.Int())()
}
