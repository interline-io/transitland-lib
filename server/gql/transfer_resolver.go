package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
)

// TRANSFER

type transferResolver struct{ *Resolver }

func (r *transferResolver) FeedVersion(ctx context.Context, obj *model.Transfer) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

func (r *transferResolver) FromStop(ctx context.Context, obj *model.Transfer) (*model.Stop, error) {
	if !obj.FromStopID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.FromStopID.Int())()
}

func (r *transferResolver) ToStop(ctx context.Context, obj *model.Transfer) (*model.Stop, error) {
	if !obj.ToStopID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.ToStopID.Int())()
}

func (r *transferResolver) FromRoute(ctx context.Context, obj *model.Transfer) (*model.Route, error) {
	if !obj.FromRouteID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).RoutesByIDs.Load(ctx, obj.FromRouteID.Int())()
}

func (r *transferResolver) ToRoute(ctx context.Context, obj *model.Transfer) (*model.Route, error) {
	if !obj.ToRouteID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).RoutesByIDs.Load(ctx, obj.ToRouteID.Int())()
}

func (r *transferResolver) FromTrip(ctx context.Context, obj *model.Transfer) (*model.Trip, error) {
	if !obj.FromTripID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).TripsByIDs.Load(ctx, obj.FromTripID.Int())()
}

func (r *transferResolver) ToTrip(ctx context.Context, obj *model.Transfer) (*model.Trip, error) {
	if !obj.ToTripID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).TripsByIDs.Load(ctx, obj.ToTripID.Int())()
}
