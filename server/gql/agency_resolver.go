package gql

import (
	"context"
	"slices"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
)

// AGENCY

type agencyResolver struct{ *Resolver }

func (r *agencyResolver) Cursor(ctx context.Context, obj *model.Agency) (*model.Cursor, error) {
	c := model.NewCursor(obj.FeedVersionID, obj.ID)
	return &c, nil
}

func (r *agencyResolver) Routes(ctx context.Context, obj *model.Agency, limit *int, where *model.RouteFilter) ([]*model.Route, error) {
	return LoaderFor(ctx).RoutesByAgencyIDs.Load(ctx, routeLoaderParam{AgencyID: obj.ID, Limit: resolverCheckLimit(limit), Where: where})()
}

func (r *agencyResolver) RouteTypes(ctx context.Context, obj *model.Agency) ([]int, error) {
	return LoaderFor(ctx).RouteTypesByAgencyIDs.Load(ctx, obj.ID)()
}

// RouteTypesBasic is RouteTypes folded onto the basic types they stand for.
//
// Taken from the same load rather than asked of the database again: the fold is a
// pure function of the answer that load already holds, so the two fields cost one
// query between them.
func (r *agencyResolver) RouteTypesBasic(ctx context.Context, obj *model.Agency) ([]int, error) {
	routeTypes, err := LoaderFor(ctx).RouteTypesByAgencyIDs.Load(ctx, obj.ID)()
	if err != nil {
		return nil, err
	}
	// Empty rather than nil, so an agency running nothing answers [] and not null.
	basic := []int{}
	for _, rt := range routeTypes {
		if b := tt.BasicRouteType(rt); !slices.Contains(basic, b) {
			basic = append(basic, b)
		}
	}
	slices.Sort(basic)
	return basic, nil
}

func (r *agencyResolver) Stops(ctx context.Context, obj *model.Agency, limit *int, after *int, where *model.AgencyStopFilter) ([]*model.Stop, error) {
	return LoaderFor(ctx).StopsByAgencyIDs.Load(ctx, agencyStopLoaderParam{AgencyID: obj.ID, Limit: resolverCheckLimit(limit), After: checkCursor(after), Where: where})()
}

func (r *agencyResolver) FeedVersion(ctx context.Context, obj *model.Agency) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

func (r *agencyResolver) Places(ctx context.Context, obj *model.Agency, limit *int, where *model.AgencyPlaceFilter) ([]*model.AgencyPlace, error) {
	return LoaderFor(ctx).AgencyPlacesByAgencyIDs.Load(ctx, agencyPlaceLoaderParam{AgencyID: obj.ID, Limit: resolverCheckLimit(limit), Where: where})()
}

func (r *agencyResolver) Operator(ctx context.Context, obj *model.Agency) (*model.Operator, error) {
	if obj.CoifID == nil {
		return nil, nil
	}
	return LoaderFor(ctx).OperatorsByCOIFs.Load(ctx, *obj.CoifID)()
}

func (r *agencyResolver) Alerts(ctx context.Context, obj *model.Agency, active *bool, limit *int) ([]*model.Alert, error) {
	rtAlerts := model.ForContext(ctx).RTFinder.FindAlertsForAgency(ctx, obj, resolverCheckLimit(limit), active)
	return rtAlerts, nil
}
