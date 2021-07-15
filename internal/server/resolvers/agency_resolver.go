package resolvers

import (
	"context"

	"github.com/interline-io/transitland-lib/internal/server/find"
	"github.com/interline-io/transitland-lib/internal/server/model"
)

// AGENCY

type agencyResolver struct{ *Resolver }

func (r *agencyResolver) Routes(ctx context.Context, obj *model.Agency, limit *int, where *model.RouteFilter) ([]*model.Route, error) {
	return find.For(ctx).RoutesByAgencyID.Load(model.RouteParam{AgencyID: obj.ID, Limit: limit, Where: where})
}

func (r *agencyResolver) FeedVersion(ctx context.Context, obj *model.Agency) (*model.FeedVersion, error) {
	return find.For(ctx).FeedVersionsByID.Load(obj.FeedVersionID)
}

func (r *agencyResolver) Places(ctx context.Context, obj *model.Agency, limit *int, where *model.AgencyPlaceFilter) ([]*model.AgencyPlace, error) {
	return find.For(ctx).AgencyPlacesByAgencyID.Load(model.AgencyPlaceParam{AgencyID: obj.ID, Limit: limit, Where: where})
}

func (r *agencyResolver) Operator(ctx context.Context, obj *model.Agency) (*model.Operator, error) {
	return find.For(ctx).OperatorsByAgencyID.Load(obj.ID)
}
