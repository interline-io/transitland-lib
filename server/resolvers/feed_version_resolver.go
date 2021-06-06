package resolvers

import (
	"context"

	"github.com/interline-io/transitland-lib/server/find"
	"github.com/interline-io/transitland-lib/server/model"
)

// FEED VERSION

type feedVersionResolver struct{ *Resolver }

func (r *feedVersionResolver) Agencies(ctx context.Context, obj *model.FeedVersion, limit *int, where *model.AgencyFilter) ([]*model.Agency, error) {
	return find.For(ctx).AgenciesByFeedVersionID.Load(model.AgencyParam{FeedVersionID: obj.ID, Limit: limit})
}

func (r *feedVersionResolver) Routes(ctx context.Context, obj *model.FeedVersion, limit *int, where *model.RouteFilter) ([]*model.Route, error) {
	return find.For(ctx).RoutesByFeedVersionID.Load(model.RouteParam{FeedVersionID: obj.ID, Limit: limit, Where: where})
}

func (r *feedVersionResolver) Stops(ctx context.Context, obj *model.FeedVersion, limit *int, where *model.StopFilter) ([]*model.Stop, error) {
	return find.For(ctx).StopsByFeedVersionID.Load(model.StopParam{FeedVersionID: obj.ID, Limit: limit, Where: where})
}

func (r *feedVersionResolver) Trips(ctx context.Context, obj *model.FeedVersion, limit *int, where *model.TripFilter) ([]*model.Trip, error) {
	// todo return find.For(ctx).TripsByFeedVersionID
	return nil, nil
}

func (r *feedVersionResolver) Feed(ctx context.Context, obj *model.FeedVersion) (*model.Feed, error) {
	return find.For(ctx).FeedsByID.Load(obj.FeedID)
}

func (r *feedVersionResolver) Files(ctx context.Context, obj *model.FeedVersion, limit *int) ([]*model.FeedVersionFileInfo, error) {
	return find.For(ctx).FeedVersionFileInfosByFeedVersionID.Load(model.FeedVersionFileInfoParam{FeedVersionID: obj.ID, Limit: limit})
}

func (r *feedVersionResolver) FeedVersionGtfsImport(ctx context.Context, obj *model.FeedVersion) (*model.FeedVersionGtfsImport, error) {
	return find.For(ctx).FeedVersionGtfsImportsByFeedVersionID.Load(obj.ID)
}

func (r *feedVersionResolver) ServiceLevels(ctx context.Context, obj *model.FeedVersion, limit *int, where *model.FeedVersionServiceLevelFilter) ([]*model.FeedVersionServiceLevel, error) {
	return find.For(ctx).FeedVersionServiceLevelsByFeedVersionID.Load(model.FeedVersionServiceLevelParam{FeedVersionID: obj.ID, Limit: limit, Where: where})
}

func (r *feedVersionResolver) FeedInfos(ctx context.Context, obj *model.FeedVersion, limit *int) ([]*model.FeedInfo, error) {
	return find.For(ctx).FeedInfosByFeedVersionID.Load(model.FeedInfoParam{FeedVersionID: obj.ID, Limit: limit})
}

// FEED VERSION GTFS IMPORT

type feedVersionGtfsImportResolver struct{ *Resolver }

func (r *feedVersionGtfsImportResolver) EntityCount(ctx context.Context, obj *model.FeedVersionGtfsImport) (interface{}, error) {
	return obj.EntityCount, nil
}

func (r *feedVersionGtfsImportResolver) WarningCount(ctx context.Context, obj *model.FeedVersionGtfsImport) (interface{}, error) {
	return obj.WarningCount, nil
}

func (r *feedVersionGtfsImportResolver) SkipEntityErrorCount(ctx context.Context, obj *model.FeedVersionGtfsImport) (interface{}, error) {
	return obj.SkipEntityErrorCount, nil
}

func (r *feedVersionGtfsImportResolver) SkipEntityReferenceCount(ctx context.Context, obj *model.FeedVersionGtfsImport) (interface{}, error) {
	return obj.SkipEntityReferenceCount, nil
}

func (r *feedVersionGtfsImportResolver) SkipEntityFilterCount(ctx context.Context, obj *model.FeedVersionGtfsImport) (interface{}, error) {
	return obj.SkipEntityFilterCount, nil
}

func (r *feedVersionGtfsImportResolver) SkipEntityMarkedCount(ctx context.Context, obj *model.FeedVersionGtfsImport) (interface{}, error) {
	return obj.SkipEntityMarkedCount, nil
}

func (r *feedStateResolver) FeedVersion(ctx context.Context, obj *model.FeedState) (*model.FeedVersion, error) {
	return find.For(ctx).FeedVersionsByID.Load(int(obj.FeedVersionID.Int))
}
