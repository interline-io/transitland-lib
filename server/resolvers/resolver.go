//go:generate go run github.com/99designs/gqlgen

package resolvers

import (
	"context"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"

	"github.com/interline-io/transitland-lib/server/auth"
	"github.com/interline-io/transitland-lib/server/config"
	"github.com/interline-io/transitland-lib/server/find"
	"github.com/interline-io/transitland-lib/server/generated/gqlgen"
	"github.com/interline-io/transitland-lib/server/model"
)

func atoi(v string) int {
	a, _ := strconv.Atoi(v)
	return a
}

// Resolver .
type Resolver struct {
	cfg config.Config
}

// Query .
func (r *Resolver) Query() gqlgen.QueryResolver { return &queryResolver{r} }

// Mutation .
func (r *Resolver) Mutation() gqlgen.MutationResolver { return &mutationResolver{r} }

// type helpers

// Agency .
func (r *Resolver) Agency() gqlgen.AgencyResolver { return &agencyResolver{r} }

// Feed .
func (r *Resolver) Feed() gqlgen.FeedResolver { return &feedResolver{r} }

// FeedState .
func (r *Resolver) FeedState() gqlgen.FeedStateResolver { return &feedStateResolver{r} }

// FeedVersion .
func (r *Resolver) FeedVersion() gqlgen.FeedVersionResolver { return &feedVersionResolver{r} }

// Route .
func (r *Resolver) Route() gqlgen.RouteResolver { return &routeResolver{r} }

// RouteStop .
func (r *Resolver) RouteStop() gqlgen.RouteStopResolver { return &routeStopResolver{r} }

// RouteHeadway .
func (r *Resolver) RouteHeadway() gqlgen.RouteHeadwayResolver { return &routeHeadwayResolver{r} }

// Stop .
func (r *Resolver) Stop() gqlgen.StopResolver { return &stopResolver{r} }

// Trip .
func (r *Resolver) Trip() gqlgen.TripResolver { return &tripResolver{r} }

// StopTime .
func (r *Resolver) StopTime() gqlgen.StopTimeResolver { return &stopTimeResolver{r} }

// Operator .
func (r *Resolver) Operator() gqlgen.OperatorResolver { return &operatorResolver{r} }

// FeedVersionGtfsImport .
func (r *Resolver) FeedVersionGtfsImport() gqlgen.FeedVersionGtfsImportResolver {
	return &feedVersionGtfsImportResolver{r}
}

// CensusGeography .
func (r *Resolver) CensusGeography() gqlgen.CensusGeographyResolver {
	return &censusGeographyResolver{r}
}

// CensusValue .
func (r *Resolver) CensusValue() gqlgen.CensusValueResolver {
	return &censusValueResolver{r}
}

// Pathway .
func (r *Resolver) Pathway() gqlgen.PathwayResolver {
	return &pathwayResolver{r}
}

////////////////////////// ROOT RESOLVER

// query root

type queryResolver struct{ *Resolver }

func (r *queryResolver) Agencies(ctx context.Context, limit *int, after *int, ids []int, where *model.AgencyFilter) ([]*model.Agency, error) {
	return find.FindAgencies(model.DB, limit, after, ids, where)
}

func (r *queryResolver) Routes(ctx context.Context, limit *int, after *int, ids []int, where *model.RouteFilter) ([]*model.Route, error) {
	return find.FindRoutes(model.DB, limit, after, ids, where)
}

func (r *queryResolver) Stops(ctx context.Context, limit *int, after *int, ids []int, where *model.StopFilter) ([]*model.Stop, error) {
	return find.FindStops(model.DB, limit, after, ids, where)
}

func (r *queryResolver) Trips(ctx context.Context, limit *int, after *int, ids []int, where *model.TripFilter) ([]*model.Trip, error) {
	return find.FindTrips(model.DB, limit, after, ids, where)
}

func (r *queryResolver) FeedVersions(ctx context.Context, limit *int, after *int, ids []int, where *model.FeedVersionFilter) ([]*model.FeedVersion, error) {
	return find.FindFeedVersions(model.DB, limit, after, ids, where)
}

func (r *queryResolver) Feeds(ctx context.Context, limit *int, after *int, ids []int, where *model.FeedFilter) ([]*model.Feed, error) {
	return find.FindFeeds(model.DB, limit, after, ids, where)
}

func (r *queryResolver) Operators(ctx context.Context, limit *int, after *int, ids []int, where *model.OperatorFilter) ([]*model.Operator, error) {
	return find.FindOperators(model.DB, limit, after, ids, where)
}

// mutation root

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) ValidateGtfs(ctx context.Context, file *graphql.Upload, url *string, rturls []string) (*model.ValidationResult, error) {
	var src io.Reader
	if file != nil {
		src = file.File
	}
	return ValidateUpload(r.cfg, src, url, rturls, auth.ForContext(ctx))
}

func (r *mutationResolver) FeedVersionFetch(ctx context.Context, file *graphql.Upload, url *string, feed string) (*model.FeedVersionFetchResult, error) {
	var src io.Reader
	if file != nil {
		src = file.File
	}
	return Fetch(r.cfg, src, url, feed, auth.ForContext(ctx))
}

func (r *mutationResolver) FeedVersionImport(ctx context.Context, sha1 string) (*model.FeedVersionImportResult, error) {
	return Import(r.cfg, sha1, auth.ForContext(ctx))
}

func (r *mutationResolver) FeedVersionUpdate(ctx context.Context, id int, values model.FeedVersionSetInput) (*model.FeedVersion, error) {
	return UpdateFeedVersion(id, values, auth.ForContext(ctx))
}

func (r *mutationResolver) FeedVersionUnimport(ctx context.Context, id int) (*model.FeedVersionUnimportResult, error) {
	return UnimportFeedVersion(id)
}

func (r *mutationResolver) FeedVersionDelete(ctx context.Context, id int) (*model.FeedVersionDeleteResult, error) {
	return FeedVersionDelete(id)
}

////////////////////////// SUPPORT ENTITY RESOLVERS

// FEED

type feedResolver struct{ *Resolver }

func (r *feedResolver) FeedState(ctx context.Context, obj *model.Feed) (*model.FeedState, error) {
	return find.For(ctx).FeedStatesByFeedID.Load(obj.ID)
}

func (r *feedResolver) FeedVersions(ctx context.Context, obj *model.Feed, limit *int, where *model.FeedVersionFilter) ([]*model.FeedVersion, error) {
	return find.For(ctx).FeedVersionsByFeedID.Load(model.FeedVersionParam{
		FeedID: obj.ID,
		Limit:  limit,
		Where:  where,
	})
}

func (r *feedResolver) License(ctx context.Context, obj *model.Feed) (*model.FeedLicense, error) {
	return &model.FeedLicense{FeedLicense: obj.License}, nil
}

func (r *feedResolver) Languages(ctx context.Context, obj *model.Feed) ([]string, error) {
	return obj.Languages, nil
}

func (r *feedResolver) AssociatedFeeds(ctx context.Context, obj *model.Feed) ([]string, error) {
	return obj.AssociatedFeeds, nil
}

func (r *feedResolver) Urls(ctx context.Context, obj *model.Feed) (*model.FeedUrls, error) {
	return &model.FeedUrls{FeedUrls: obj.URLs}, nil
}

func (r *feedResolver) AssociatedOperators(ctx context.Context, obj *model.Feed) ([]*model.Operator, error) {
	return find.For(ctx).OperatorsByFeedID.Load(model.OperatorParam{FeedID: obj.ID})
}

func (r *feedResolver) Authorization(ctx context.Context, obj *model.Feed) (*model.FeedAuthorization, error) {
	return &model.FeedAuthorization{FeedAuthorization: obj.Authorization}, nil
}

// FEED STATE

type feedStateResolver struct{ *Resolver }

func (r *feedStateResolver) LastFetchedAt(ctx context.Context, obj *model.FeedState) (*time.Time, error) {
	// TODO: Add Custom Marshaler
	if obj.LastFetchedAt.Valid {
		return &obj.LastFetchedAt.Time, nil
	}
	return nil, nil
}

func (r *feedStateResolver) LastSuccessfulFetchAt(ctx context.Context, obj *model.FeedState) (*time.Time, error) {
	// TODO: Add Custom Marshaler
	if obj.LastSuccessfulFetchAt.Valid {
		return &obj.LastSuccessfulFetchAt.Time, nil
	}
	return nil, nil
}

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

////////////////////////// GTFS ENTITY RESOLVERS

// AGENCY

type agencyResolver struct{ *Resolver }

func (r *agencyResolver) Routes(ctx context.Context, obj *model.Agency, limit *int, where *model.RouteFilter) ([]*model.Route, error) {
	return find.For(ctx).RoutesByAgencyID.Load(model.RouteParam{
		AgencyID: obj.ID,
		Limit:    limit,
		Where:    where,
	})
}

func (r *agencyResolver) FeedVersion(ctx context.Context, obj *model.Agency) (*model.FeedVersion, error) {
	return find.For(ctx).FeedVersionsByID.Load(obj.FeedVersionID)
}

func (r *agencyResolver) Places(ctx context.Context, obj *model.Agency, limit *int, where *model.AgencyPlaceFilter) ([]*model.AgencyPlace, error) {
	return find.For(ctx).AgencyPlacesByAgencyID.Load(model.AgencyPlaceParam{AgencyID: obj.ID, Limit: limit, Where: where})
}

// OPERATOR

type operatorResolver struct{ *Resolver }

func (r *operatorResolver) Agency(ctx context.Context, obj *model.Operator) (*model.Agency, error) {
	if obj.AgencyID != nil {
		return find.For(ctx).AgenciesByID.Load(*obj.AgencyID)
	}
	return nil, nil
}

func (r *operatorResolver) OperatorTags(ctx context.Context, obj *model.Operator) (interface{}, error) {
	return obj.OperatorTags, nil
}

func (r *operatorResolver) OperatorAssociatedFeeds(ctx context.Context, obj *model.Operator) (interface{}, error) {
	return obj.OperatorAssociatedFeeds, nil
}

func (r *operatorResolver) PlacesCache(ctx context.Context, obj *model.Operator) ([]string, error) {
	if obj.PlacesCache != nil {
		return *obj.PlacesCache, nil
	}
	return []string{}, nil
}

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

// STOP

type stopResolver struct{ *Resolver }

func (r *stopResolver) FeedVersion(ctx context.Context, obj *model.Stop) (*model.FeedVersion, error) {
	return find.For(ctx).FeedVersionsByID.Load(obj.FeedVersionID)
}

func (r *stopResolver) Level(ctx context.Context, obj *model.Stop) (*model.Level, error) {
	if !obj.LevelID.Valid {
		return nil, nil
	}
	return find.For(ctx).LevelsByID.Load(atoi(obj.LevelID.Key))
}

func (r *stopResolver) Parent(ctx context.Context, obj *model.Stop) (*model.Stop, error) {
	if !obj.ParentStation.Valid {
		return nil, nil
	}
	return find.For(ctx).StopsByID.Load(atoi(obj.ParentStation.Key))
}

func (r *stopResolver) Children(ctx context.Context, obj *model.Stop, limit *int) ([]*model.Stop, error) {
	return find.For(ctx).StopsByParentStopID.Load(model.StopParam{ParentStopID: obj.ID, Limit: limit})
}

func (r *stopResolver) RouteStops(ctx context.Context, obj *model.Stop, limit *int) ([]*model.RouteStop, error) {
	return find.For(ctx).RouteStopsByStopID.Load(model.RouteStopParam{StopID: obj.ID, Limit: limit})
}

func (r *stopResolver) PathwaysFromStop(ctx context.Context, obj *model.Stop, limit *int) ([]*model.Pathway, error) {
	return find.For(ctx).PathwaysByFromStopID.Load(model.PathwayParam{FromStopID: obj.ID, Limit: limit})
}

func (r *stopResolver) PathwaysToStop(ctx context.Context, obj *model.Stop, limit *int) ([]*model.Pathway, error) {
	return find.For(ctx).PathwaysByToStopID.Load(model.PathwayParam{ToStopID: obj.ID, Limit: limit})
}

func (r *stopResolver) StopTimes(ctx context.Context, obj *model.Stop, limit *int, where *model.StopTimeFilter) ([]*model.StopTime, error) {
	return find.For(ctx).StopTimesByStopID.Load(model.StopTimeParam{StopID: obj.ID, Limit: limit, Where: where})

}

// TRIP

type tripResolver struct{ *Resolver }

func (r *tripResolver) Route(ctx context.Context, obj *model.Trip) (*model.Route, error) {
	return find.For(ctx).RoutesByID.Load(atoi(obj.RouteID))
}

func (r *tripResolver) FeedVersion(ctx context.Context, obj *model.Trip) (*model.FeedVersion, error) {
	return find.For(ctx).FeedVersionsByID.Load(obj.FeedVersionID)
}

func (r *tripResolver) Shape(ctx context.Context, obj *model.Trip) (*model.Shape, error) {
	if !obj.ShapeID.Valid {
		return nil, nil
	}
	return find.For(ctx).ShapesByID.Load(obj.ShapeID.Int())
}

func (r *tripResolver) Calendar(ctx context.Context, obj *model.Trip) (*model.Calendar, error) {
	return find.For(ctx).CalendarsByID.Load(atoi(obj.ServiceID))
}

func (r *tripResolver) StopTimes(ctx context.Context, obj *model.Trip, limit *int) ([]*model.StopTime, error) {
	return find.For(ctx).StopTimesByTripID.Load(model.StopTimeParam{TripID: obj.ID, Limit: limit})
}

func (r *tripResolver) Frequencies(ctx context.Context, obj *model.Trip, limit *int) ([]*model.Frequency, error) {
	return find.For(ctx).FrequenciesByTripID.Load(model.FrequencyParam{TripID: obj.ID, Limit: limit})
}

// STOP TIME

type stopTimeResolver struct{ *Resolver }

func (r *stopTimeResolver) Stop(ctx context.Context, obj *model.StopTime) (*model.Stop, error) {
	return find.For(ctx).StopsByID.Load(atoi(obj.StopID))
}

func (r *stopTimeResolver) Trip(ctx context.Context, obj *model.StopTime) (*model.Trip, error) {
	return find.For(ctx).TripsByID.Load(atoi(obj.TripID))
}

func (r *stopTimeResolver) PickupType(ctx context.Context, obj *model.StopTime) (*int, error) {
	return nil, nil
}

func (r *stopTimeResolver) DropOffType(ctx context.Context, obj *model.StopTime) (*int, error) {
	return nil, nil
}

func (r *stopTimeResolver) Interpolated(ctx context.Context, obj *model.StopTime) (*int, error) {
	return nil, nil
}

func (r *stopTimeResolver) StopHeadsign(ctx context.Context, obj *model.StopTime) (*string, error) {
	return nil, nil
}

func (r *stopTimeResolver) Timepoint(ctx context.Context, obj *model.StopTime) (*int, error) {
	return nil, nil
}

// CALENDAR

type calendarResolver struct{ *Resolver }

func (r *calendarResolver) AddedDates(ctx context.Context, obj *model.Calendar) ([]*model.CalendarDate, error) {
	return nil, nil
}

func (r *calendarResolver) RemovedDates(ctx context.Context, obj *model.Calendar) ([]*model.CalendarDate, error) {
	return nil, nil
}

func (r *calendarResolver) Timepoint(ctx context.Context, obj *model.Calendar) ([]*model.CalendarDate, error) {
	return nil, nil
}

// PATHWAYS

type pathwayResolver struct{ *Resolver }

func (r *pathwayResolver) FromStop(ctx context.Context, obj *model.Pathway) (*model.Stop, error) {
	return find.For(ctx).StopsByID.Load(atoi(obj.FromStopID))
}

func (r *pathwayResolver) ToStop(ctx context.Context, obj *model.Pathway) (*model.Stop, error) {
	return find.For(ctx).StopsByID.Load(atoi(obj.ToStopID))
}

////////////////////////// CENSUS RESOLVERS

type censusGeographyResolver struct{ *Resolver }

func (r *censusGeographyResolver) Values(ctx context.Context, obj *model.CensusGeography, tableNames []string, limit *int) (ents []*model.CensusValue, err error) {
	// dataloader cant easily pass []string
	return find.For(ctx).CensusValuesByGeographyID.Load(model.CensusValueParam{TableNames: strings.Join(tableNames, ","), Limit: limit, GeographyID: obj.ID})
}

type censusValueResolver struct{ *Resolver }

func (r *censusValueResolver) Table(ctx context.Context, obj *model.CensusValue) (*model.CensusTable, error) {
	return find.For(ctx).CensusTableByID.Load(obj.TableID)
}

func (r *censusValueResolver) Values(ctx context.Context, obj *model.CensusValue) (interface{}, error) {
	return obj.TableValues, nil
}

// add geography resolvers to agency, route, stop

func (r *agencyResolver) CensusGeographies(ctx context.Context, obj *model.Agency, layerName string, radius *float64, limit *int) (ents []*model.CensusGeography, err error) {
	return find.For(ctx).CensusGeographiesByEntityID.Load(model.CensusGeographyParam{
		EntityType: "agency",
		EntityID:   obj.ID,
		Radius:     radius,
		LayerName:  layerName,
		Limit:      limit,
	})
}

func (r *routeResolver) CensusGeographies(ctx context.Context, obj *model.Route, layerName string, radius *float64, limit *int) (ents []*model.CensusGeography, err error) {
	return find.For(ctx).CensusGeographiesByEntityID.Load(model.CensusGeographyParam{
		EntityType: "route",
		EntityID:   obj.ID,
		Radius:     radius,
		LayerName:  layerName,
		Limit:      limit,
	})
}

func (r *stopResolver) CensusGeographies(ctx context.Context, obj *model.Stop, layerName string, radius *float64, limit *int) (ents []*model.CensusGeography, err error) {
	return find.For(ctx).CensusGeographiesByEntityID.Load(model.CensusGeographyParam{
		EntityType: "stop",
		EntityID:   obj.ID,
		Radius:     radius,
		LayerName:  layerName,
		Limit:      limit,
	})
}
