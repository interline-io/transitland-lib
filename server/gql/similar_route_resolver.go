package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
)

type similarRouteResolver struct{ *Resolver }

// Route loads the candidate by id, through the same permission check as any
// route lookup.
func (r *similarRouteResolver) Route(ctx context.Context, obj *model.SimilarRoute) (*model.Route, error) {
	return LoaderFor(ctx).RoutesByIDs.Load(ctx, obj.OtherRouteID)()
}

type routeStopCorrespondenceResolver struct{ *Resolver }

func (r *routeStopCorrespondenceResolver) Stop(ctx context.Context, obj *model.RouteStopCorrespondence) (*model.Stop, error) {
	return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.StopID)()
}

func (r *routeStopCorrespondenceResolver) OtherStop(ctx context.Context, obj *model.RouteStopCorrespondence) (*model.Stop, error) {
	return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.OtherStopID)()
}
