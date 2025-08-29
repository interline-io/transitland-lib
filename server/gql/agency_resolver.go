package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
)

// AGENCY

type agencyResolver struct{ *Resolver }

func (r *agencyResolver) Cursor(ctx context.Context, obj *model.Agency) (*model.Cursor, error) {
	c := model.NewCursor(obj.FeedVersionID, obj.ID)
	return &c, nil
}

func (r *agencyResolver) Routes(ctx context.Context, obj *model.Agency, limit *int, where *model.RouteFilter) ([]*model.Route, error) {
	return LoaderFor(ctx).RoutesByAgencyIDs.Load(ctx, routeLoaderParam{AgencyID: obj.ID, Limit: checkLimit(limit), Where: where})()
}

func (r *agencyResolver) FeedVersion(ctx context.Context, obj *model.Agency) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

func (r *agencyResolver) Places(ctx context.Context, obj *model.Agency, limit *int, where *model.AgencyPlaceFilter) ([]*model.AgencyPlace, error) {
	return LoaderFor(ctx).AgencyPlacesByAgencyIDs.Load(ctx, agencyPlaceLoaderParam{AgencyID: obj.ID, Limit: checkLimit(limit), Where: where})()
}

func (r *agencyResolver) Operator(ctx context.Context, obj *model.Agency) (*model.Operator, error) {
	if obj.CoifID == nil {
		return nil, nil
	}
	return LoaderFor(ctx).OperatorsByCOIFs.Load(ctx, *obj.CoifID)()
}

func (r *agencyResolver) Alerts(ctx context.Context, obj *model.Agency, active *bool, limit *int) ([]*model.Alert, error) {
	rtAlerts := model.ForContext(ctx).RTFinder.FindAlertsForAgency(ctx, obj, checkLimit(limit), active)
	return rtAlerts, nil
}
