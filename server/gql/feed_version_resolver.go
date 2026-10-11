package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
)

// FEED VERSION

type feedVersionResolver struct{ *Resolver }

func (r *feedVersionResolver) Cursor(ctx context.Context, obj *model.FeedVersion) (*model.Cursor, error) {
	c := model.NewCursor(0, obj.ID)
	return &c, nil
}

func (r *feedVersionResolver) Agencies(ctx context.Context, obj *model.FeedVersion, limit *int, where *model.AgencyFilter) ([]*model.Agency, error) {
	return LoaderFor(ctx).AgenciesByFeedVersionIDs.Load(ctx, agencyLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimit(limit)})()
}

func (r *feedVersionResolver) Routes(ctx context.Context, obj *model.FeedVersion, limit *int, where *model.RouteFilter) ([]*model.Route, error) {
	return LoaderFor(ctx).RoutesByFeedVersionIDs.Load(ctx, routeLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimit(limit), Where: where})()
}

func (r *feedVersionResolver) Stops(ctx context.Context, obj *model.FeedVersion, limit *int, where *model.StopFilter) ([]*model.Stop, error) {
	return LoaderFor(ctx).StopsByFeedVersionIDs.Load(ctx, stopLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimit(limit), Where: where})()
}

func (r *feedVersionResolver) Trips(ctx context.Context, obj *model.FeedVersion, limit *int, where *model.TripFilter) ([]*model.Trip, error) {
	return LoaderFor(ctx).TripsByFeedVersionIDs.Load(ctx, tripLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimit(limit), Where: where})()
}

func (r *feedVersionResolver) Calendars(ctx context.Context, obj *model.FeedVersion, limit *int, after *int, where *model.CalendarFilter) ([]*model.Calendar, error) {
	return LoaderFor(ctx).CalendarsByFeedVersionIDs.Load(ctx, feedVersionTableParam[*model.CalendarFilter]{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_GTFS_TABLE_MAXLIMIT), After: checkCursor(after), Where: where})()
}

func (r *feedVersionResolver) CalendarDates(ctx context.Context, obj *model.FeedVersion, limit *int, after *int, where *model.CalendarDateFilter) ([]*model.CalendarDate, error) {
	return LoaderFor(ctx).CalendarDatesByFeedVersionIDs.Load(ctx, feedVersionTableParam[*model.CalendarDateFilter]{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_GTFS_TABLE_MAXLIMIT), After: checkCursor(after), Where: where})()
}

func (r *feedVersionResolver) Frequencies(ctx context.Context, obj *model.FeedVersion, limit *int, after *int, where *model.FrequencyFilter) ([]*model.Frequency, error) {
	return LoaderFor(ctx).FrequenciesByFeedVersionIDs.Load(ctx, feedVersionTableParam[*model.FrequencyFilter]{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_GTFS_TABLE_MAXLIMIT), After: checkCursor(after), Where: where})()
}

func (r *feedVersionResolver) Transfers(ctx context.Context, obj *model.FeedVersion, limit *int, after *int, where *model.TransferFilter) ([]*model.Transfer, error) {
	return LoaderFor(ctx).TransfersByFeedVersionIDs.Load(ctx, feedVersionTableParam[*model.TransferFilter]{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_GTFS_TABLE_MAXLIMIT), After: checkCursor(after), Where: where})()
}

func (r *feedVersionResolver) Pathways(ctx context.Context, obj *model.FeedVersion, limit *int, after *int, where *model.PathwayFilter) ([]*model.Pathway, error) {
	return LoaderFor(ctx).PathwaysByFeedVersionIDs.Load(ctx, feedVersionTableParam[*model.PathwayFilter]{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_GTFS_TABLE_MAXLIMIT), After: checkCursor(after), Where: where})()
}

func (r *feedVersionResolver) Levels(ctx context.Context, obj *model.FeedVersion, limit *int, after *int, where *model.LevelFilter) ([]*model.Level, error) {
	return LoaderFor(ctx).LevelsByFeedVersionIDs.Load(ctx, feedVersionTableParam[*model.LevelFilter]{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_GTFS_TABLE_MAXLIMIT), After: checkCursor(after), Where: where})()
}

func (r *feedVersionResolver) LocationGroupStops(ctx context.Context, obj *model.FeedVersion, limit *int, after *int, where *model.LocationGroupStopFilter) ([]*model.LocationGroupStop, error) {
	return LoaderFor(ctx).LocationGroupStopsByFeedVersionIDs.Load(ctx, feedVersionTableParam[*model.LocationGroupStopFilter]{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_GTFS_TABLE_MAXLIMIT), After: checkCursor(after), Where: where})()
}

func (r *feedVersionResolver) StopAreas(ctx context.Context, obj *model.FeedVersion, limit *int, after *int, where *model.StopAreaFilter) ([]*model.StopArea, error) {
	return LoaderFor(ctx).StopAreasByFeedVersionIDs.Load(ctx, feedVersionTableParam[*model.StopAreaFilter]{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_GTFS_TABLE_MAXLIMIT), After: checkCursor(after), Where: where})()
}

func (r *feedVersionResolver) Translations(ctx context.Context, obj *model.FeedVersion, limit *int, after *int, where *model.TranslationFilter) ([]*model.Translation, error) {
	return LoaderFor(ctx).TranslationsByFeedVersionIDs.Load(ctx, feedVersionTableParam[*model.TranslationFilter]{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_GTFS_TABLE_MAXLIMIT), After: checkCursor(after), Where: where})()
}

func (r *feedVersionResolver) Attributions(ctx context.Context, obj *model.FeedVersion, limit *int, after *int, where *model.AttributionFilter) ([]*model.Attribution, error) {
	return LoaderFor(ctx).AttributionsByFeedVersionIDs.Load(ctx, feedVersionTableParam[*model.AttributionFilter]{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_GTFS_TABLE_MAXLIMIT), After: checkCursor(after), Where: where})()
}

func (r *feedVersionResolver) Feed(ctx context.Context, obj *model.FeedVersion) (*model.Feed, error) {
	return LoaderFor(ctx).FeedsByIDs.Load(ctx, obj.FeedID)()
}

func (r *feedVersionResolver) Geometry(ctx context.Context, obj *model.FeedVersion) (*tt.Polygon, error) {
	// Defer feed geometry loading
	return LoaderFor(ctx).FeedVersionGeometryByIDs.Load(ctx, obj.ID)()
}

func (r *feedVersionResolver) Files(ctx context.Context, obj *model.FeedVersion, limit *int) ([]*model.FeedVersionFileInfo, error) {
	return LoaderFor(ctx).FeedVersionFileInfosByFeedVersionIDs.Load(ctx, feedVersionFileInfoLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimit(limit)})()
}

func (r *feedVersionResolver) FeedVersionGtfsImport(ctx context.Context, obj *model.FeedVersion) (*model.FeedVersionGtfsImport, error) {
	return LoaderFor(ctx).FeedVersionGtfsImportByFeedVersionIDs.Load(ctx, obj.ID)()
}

func (r *feedVersionResolver) ServiceWindow(ctx context.Context, obj *model.FeedVersion) (*model.FeedVersionServiceWindow, error) {
	return LoaderFor(ctx).FeedVersionServiceWindowByFeedVersionIDs.Load(ctx, obj.ID)()
}

func (r *feedVersionResolver) ServiceLevels(ctx context.Context, obj *model.FeedVersion, limit *int, where *model.FeedVersionServiceLevelFilter) ([]*model.FeedVersionServiceLevel, error) {
	return LoaderFor(ctx).FeedVersionServiceLevelsByFeedVersionIDs.Load(ctx, feedVersionServiceLevelLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimit(limit), Where: where})()
}

func (r *feedVersionResolver) FeedInfos(ctx context.Context, obj *model.FeedVersion, limit *int) ([]*model.FeedInfo, error) {
	return LoaderFor(ctx).FeedInfosByFeedVersionIDs.Load(ctx, feedInfoLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimit(limit)})()
}

func (r *feedVersionResolver) Locations(ctx context.Context, obj *model.FeedVersion, limit *int, where *model.LocationFilter) ([]*model.Location, error) {
	return LoaderFor(ctx).LocationsByFeedVersionIDs.Load(ctx, locationLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_LOCATION_MAXLIMIT), Where: where})()
}

func (r *feedVersionResolver) BookingRules(ctx context.Context, obj *model.FeedVersion, limit *int, where *model.BookingRuleFilter) ([]*model.BookingRule, error) {
	return LoaderFor(ctx).BookingRulesByFeedVersionIDs.Load(ctx, bookingRuleLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimit(limit), Where: where})()
}

func (r *feedVersionResolver) FareAttributes(ctx context.Context, obj *model.FeedVersion, limit *int, after *int) ([]*model.FareAttribute, error) {
	return LoaderFor(ctx).FareAttributesByFeedVersionIDs.Load(ctx, fareLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_FARE_MAXLIMIT), After: checkCursor(after)})()
}

func (r *feedVersionResolver) FareRules(ctx context.Context, obj *model.FeedVersion, limit *int, after *int) ([]*model.FareRule, error) {
	return LoaderFor(ctx).FareRulesByFeedVersionIDs.Load(ctx, fareLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_FARE_MAXLIMIT), After: checkCursor(after)})()
}

func (r *feedVersionResolver) FareMedia(ctx context.Context, obj *model.FeedVersion, limit *int, after *int) ([]*model.FareMedia, error) {
	return LoaderFor(ctx).FareMediaByFeedVersionIDs.Load(ctx, fareLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_FARE_MAXLIMIT), After: checkCursor(after)})()
}

func (r *feedVersionResolver) FareProducts(ctx context.Context, obj *model.FeedVersion, limit *int, after *int) ([]*model.FareProduct, error) {
	return LoaderFor(ctx).FareProductsByFeedVersionIDs.Load(ctx, fareLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_FARE_MAXLIMIT), After: checkCursor(after)})()
}

func (r *feedVersionResolver) FareLegRules(ctx context.Context, obj *model.FeedVersion, limit *int, after *int) ([]*model.FareLegRule, error) {
	return LoaderFor(ctx).FareLegRulesByFeedVersionIDs.Load(ctx, fareLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_FARE_MAXLIMIT), After: checkCursor(after)})()
}

func (r *feedVersionResolver) FareLegJoinRules(ctx context.Context, obj *model.FeedVersion, limit *int, after *int) ([]*model.FareLegJoinRule, error) {
	return LoaderFor(ctx).FareLegJoinRulesByFeedVersionIDs.Load(ctx, fareLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_FARE_MAXLIMIT), After: checkCursor(after)})()
}

func (r *feedVersionResolver) FareTransferRules(ctx context.Context, obj *model.FeedVersion, limit *int, after *int) ([]*model.FareTransferRule, error) {
	return LoaderFor(ctx).FareTransferRulesByFeedVersionIDs.Load(ctx, fareLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_FARE_MAXLIMIT), After: checkCursor(after)})()
}

func (r *feedVersionResolver) RiderCategories(ctx context.Context, obj *model.FeedVersion, limit *int, after *int) ([]*model.RiderCategory, error) {
	return LoaderFor(ctx).RiderCategoriesByFeedVersionIDs.Load(ctx, fareLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_FARE_MAXLIMIT), After: checkCursor(after)})()
}

func (r *feedVersionResolver) Timeframes(ctx context.Context, obj *model.FeedVersion, limit *int, after *int) ([]*model.Timeframe, error) {
	return LoaderFor(ctx).TimeframesByFeedVersionIDs.Load(ctx, fareLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_FARE_MAXLIMIT), After: checkCursor(after)})()
}

func (r *feedVersionResolver) Areas(ctx context.Context, obj *model.FeedVersion, limit *int, after *int) ([]*model.Area, error) {
	return LoaderFor(ctx).AreasByFeedVersionIDs.Load(ctx, fareLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_FARE_MAXLIMIT), After: checkCursor(after)})()
}

func (r *feedVersionResolver) Networks(ctx context.Context, obj *model.FeedVersion, limit *int, after *int) ([]*model.Network, error) {
	return LoaderFor(ctx).NetworksByFeedVersionIDs.Load(ctx, fareLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_FARE_MAXLIMIT), After: checkCursor(after)})()
}

func (r *feedVersionResolver) RouteNetworks(ctx context.Context, obj *model.FeedVersion, limit *int, after *int) ([]*model.RouteNetwork, error) {
	return LoaderFor(ctx).RouteNetworksByFeedVersionIDs.Load(ctx, fareLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_FARE_MAXLIMIT), After: checkCursor(after)})()
}

func (r *feedVersionResolver) LocationGroups(ctx context.Context, obj *model.FeedVersion, limit *int, where *model.LocationGroupFilter) ([]*model.LocationGroup, error) {
	return LoaderFor(ctx).LocationGroupsByFeedVersionIDs.Load(ctx, locationGroupLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimit(limit), Where: where})()
}

// FEED VALIDATION REPORT

func (r *feedVersionResolver) ValidationReports(ctx context.Context, obj *model.FeedVersion, limit *int, where *model.ValidationReportFilter) ([]*model.ValidationReport, error) {
	return LoaderFor(ctx).ValidationReportsByFeedVersionIDs.Load(ctx, validationReportLoaderParam{FeedVersionID: obj.ID, Where: where, Limit: resolverCheckLimit(limit)})()
}

// SEGMENTS

func (r *feedVersionResolver) Segments(ctx context.Context, obj *model.FeedVersion, limit *int) ([]*model.Segment, error) {
	return LoaderFor(ctx).SegmentsByFeedVersionIDs.Load(ctx, segmentLoaderParam{FeedVersionID: obj.ID, Limit: resolverCheckLimit(limit)})()
}

func (r *feedVersionResolver) Shapes(ctx context.Context, obj *model.FeedVersion, limit *int, after *int, where *model.ShapeFilter) ([]*model.Shape, error) {
	return model.ForContext(ctx).Finder.FindShapesByFeedVersion(ctx, obj.ID, resolverCheckLimitMax(limit, RESOLVER_SHAPE_MAXLIMIT), checkCursor(after), where)
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
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, int(obj.FeedVersionID.Val))()
}
