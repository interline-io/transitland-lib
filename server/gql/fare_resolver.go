package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
)

// GTFS Fares v1 and v2 resolvers.

type fareAttributeResolver struct{ *Resolver }

func (r *fareAttributeResolver) FeedVersion(ctx context.Context, obj *model.FareAttribute) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

type fareRuleResolver struct{ *Resolver }

func (r *fareRuleResolver) FeedVersion(ctx context.Context, obj *model.FareRule) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

type fareMediaResolver struct{ *Resolver }

func (r *fareMediaResolver) FeedVersion(ctx context.Context, obj *model.FareMedia) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

type fareProductResolver struct{ *Resolver }

func (r *fareProductResolver) FeedVersion(ctx context.Context, obj *model.FareProduct) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

type fareLegRuleResolver struct{ *Resolver }

func (r *fareLegRuleResolver) FeedVersion(ctx context.Context, obj *model.FareLegRule) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

type fareLegJoinRuleResolver struct{ *Resolver }

func (r *fareLegJoinRuleResolver) FeedVersion(ctx context.Context, obj *model.FareLegJoinRule) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

type fareTransferRuleResolver struct{ *Resolver }

func (r *fareTransferRuleResolver) FeedVersion(ctx context.Context, obj *model.FareTransferRule) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

type riderCategoryResolver struct{ *Resolver }

func (r *riderCategoryResolver) FeedVersion(ctx context.Context, obj *model.RiderCategory) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

type timeframeResolver struct{ *Resolver }

func (r *timeframeResolver) FeedVersion(ctx context.Context, obj *model.Timeframe) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

type areaResolver struct{ *Resolver }

func (r *areaResolver) FeedVersion(ctx context.Context, obj *model.Area) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

type networkResolver struct{ *Resolver }

func (r *networkResolver) FeedVersion(ctx context.Context, obj *model.Network) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

type routeNetworkResolver struct{ *Resolver }

func (r *routeNetworkResolver) FeedVersion(ctx context.Context, obj *model.RouteNetwork) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

type stopAreaResolver struct{ *Resolver }

func (r *stopAreaResolver) FeedVersion(ctx context.Context, obj *model.StopArea) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

func (r *fareAttributeResolver) Agency(ctx context.Context, obj *model.FareAttribute) (*model.Agency, error) {
	if !obj.AgencyID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).AgenciesByIDs.Load(ctx, obj.AgencyID.Int())()
}

// FareAttribute resolves fare_rules.fare_id, which the importer stores as the
// referenced fare_attributes row id.
func (r *fareRuleResolver) FareAttribute(ctx context.Context, obj *model.FareRule) (*model.FareAttribute, error) {
	if !obj.FareID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).FareAttributesByIDs.Load(ctx, obj.FareID.Int())()
}

func (r *fareRuleResolver) Route(ctx context.Context, obj *model.FareRule) (*model.Route, error) {
	if !obj.RouteID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).RoutesByIDs.Load(ctx, obj.RouteID.Int())()
}

func (r *fareProductResolver) RiderCategoryID(ctx context.Context, obj *model.FareProduct) (*string, error) {
	return obj.RiderCategoryID.Ptr(), nil
}

// FareMedia resolves fare_products.fare_media_id, which the importer stores as
// the referenced fare_media row id.
func (r *fareProductResolver) FareMedia(ctx context.Context, obj *model.FareProduct) (*model.FareMedia, error) {
	if !obj.FareMediaID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).FareMediaByIDs.Load(ctx, obj.FareMediaID.Int())()
}

func (r *timeframeResolver) Service(ctx context.Context, obj *model.Timeframe) (*model.Calendar, error) {
	if !obj.ServiceID.Valid {
		return nil, nil
	}
	return LoaderFor(ctx).CalendarsByIDs.Load(ctx, obj.ServiceID.Int())()
}

func (r *areaResolver) StopAreas(ctx context.Context, obj *model.Area, limit *int) ([]*model.StopArea, error) {
	return LoaderFor(ctx).StopAreasByAreaIDs.Load(ctx, stopAreaLoaderParam{AreaID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_FARE_MAXLIMIT)})()
}

func (r *stopAreaResolver) Area(ctx context.Context, obj *model.StopArea) (*model.Area, error) {
	return LoaderFor(ctx).AreasByIDs.Load(ctx, obj.AreaID.Int())()
}

func (r *stopAreaResolver) Stop(ctx context.Context, obj *model.StopArea) (*model.Stop, error) {
	return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.StopID.Int())()
}

func (r *routeNetworkResolver) Network(ctx context.Context, obj *model.RouteNetwork) (*model.Network, error) {
	return LoaderFor(ctx).NetworksByIDs.Load(ctx, obj.NetworkID.Int())()
}

func (r *routeNetworkResolver) Route(ctx context.Context, obj *model.RouteNetwork) (*model.Route, error) {
	return LoaderFor(ctx).RoutesByIDs.Load(ctx, obj.RouteID.Int())()
}
