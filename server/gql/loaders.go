package gql

// import graph gophers with your other imports
import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	dataloader "github.com/graph-gophers/dataloader/v7"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
)

type ctxKey string

const (
	loadersKey            = ctxKey("dataloaders")
	waitTime              = 2 * time.Millisecond
	maxBatch              = 100
	stopTimeBatchWaitTime = 10 * time.Millisecond
)

// Loaders wrap your data loaders to inject via middleware
type Loaders struct {
	AgenciesByFeedVersionIDs                                      *dataloader.Loader[agencyLoaderParam, []*model.Agency]
	AgenciesByIDs                                                 *dataloader.Loader[int, *model.Agency]
	AgenciesByOnestopIDs                                          *dataloader.Loader[agencyLoaderParam, []*model.Agency]
	AgencyPlacesByAgencyIDs                                       *dataloader.Loader[agencyPlaceLoaderParam, []*model.AgencyPlace]
	BookingRulesByFeedVersionIDs                                  *dataloader.Loader[bookingRuleLoaderParam, []*model.BookingRule]
	BookingRulesByIDs                                             *dataloader.Loader[int, *model.BookingRule]
	CalendarDatesByServiceIDs                                     *dataloader.Loader[calendarDateLoaderParam, []*model.CalendarDate]
	CalendarsByIDs                                                *dataloader.Loader[int, *model.Calendar]
	CensusDatasetLayersByDatasetIDs                               *dataloader.Loader[int, []*model.CensusLayer]
	CensusSourceLayersBySourceIDs                                 *dataloader.Loader[int, []*model.CensusLayer]
	CensusFieldsByTableIDs                                        *dataloader.Loader[censusFieldLoaderParam, []*model.CensusField]
	CensusGeographiesByDatasetIDs                                 *dataloader.Loader[censusDatasetGeographyLoaderParam, []*model.CensusGeography]
	CensusGeographiesBySourceIDs                                  *dataloader.Loader[censusSourceGeographyLoaderParam, []*model.CensusGeography]
	CensusGeographiesByEntityIDs                                  *dataloader.Loader[censusGeographyLoaderParam, []*model.CensusGeography]
	CensusSourcesByDatasetIDs                                     *dataloader.Loader[censusSourceLoaderParam, []*model.CensusSource]
	CensusGeographiesByLayerIDs                                   *dataloader.Loader[censusSourceGeographyLoaderParam, []*model.CensusGeography]
	CensusSourcesByIDs                                            *dataloader.Loader[int, *model.CensusSource]
	CensusLayersByIDs                                             *dataloader.Loader[int, *model.CensusLayer]
	CensusTableByIDs                                              *dataloader.Loader[int, *model.CensusTable]
	CensusValuesByGeographyIDs                                    *dataloader.Loader[censusValueLoaderParam, []*model.CensusValue]
	FeedFetchesByFeedIDs                                          *dataloader.Loader[feedFetchLoaderParam, []*model.FeedFetch]
	FeedInfosByFeedVersionIDs                                     *dataloader.Loader[feedInfoLoaderParam, []*model.FeedInfo]
	FeedsByIDs                                                    *dataloader.Loader[int, *model.Feed]
	FeedsByOperatorOnestopIDs                                     *dataloader.Loader[feedLoaderParam, []*model.Feed]
	FeedStatesByFeedIDs                                           *dataloader.Loader[int, *model.FeedState]
	FeedVersionFileInfosByFeedVersionIDs                          *dataloader.Loader[feedVersionFileInfoLoaderParam, []*model.FeedVersionFileInfo]
	FeedVersionGeometryByIDs                                      *dataloader.Loader[int, *tt.Polygon]
	FeedVersionGtfsImportByFeedVersionIDs                         *dataloader.Loader[int, *model.FeedVersionGtfsImport]
	FeedVersionsByFeedIDs                                         *dataloader.Loader[feedVersionLoaderParam, []*model.FeedVersion]
	FeedVersionsByIDs                                             *dataloader.Loader[int, *model.FeedVersion]
	FeedVersionServiceLevelsByFeedVersionIDs                      *dataloader.Loader[feedVersionServiceLevelLoaderParam, []*model.FeedVersionServiceLevel]
	FeedVersionServiceWindowByFeedVersionIDs                      *dataloader.Loader[int, *model.FeedVersionServiceWindow]
	FlexStopTimesByTripIDs                                        *dataloader.Loader[tripStopTimeLoaderParam, []*model.FlexStopTime]
	FlexStopTimesByStopIDs                                        *dataloader.Loader[stopTimeLoaderParam, []*model.FlexStopTime]
	FrequenciesByTripIDs                                          *dataloader.Loader[frequencyLoaderParam, []*model.Frequency]
	LevelsByIDs                                                   *dataloader.Loader[int, *model.Level]
	LevelsByParentStationIDs                                      *dataloader.Loader[levelLoaderParam, []*model.Level]
	LocationGroupsByFeedVersionIDs                                *dataloader.Loader[locationGroupLoaderParam, []*model.LocationGroup]
	LocationGroupsByIDs                                           *dataloader.Loader[int, *model.LocationGroup]
	LocationsByFeedVersionIDs                                     *dataloader.Loader[locationLoaderParam, []*model.Location]
	LocationsByIDs                                                *dataloader.Loader[int, *model.Location]
	OperatorsByAgencyIDs                                          *dataloader.Loader[int, *model.Operator]
	OperatorsByCOIFs                                              *dataloader.Loader[int, *model.Operator]
	OperatorsByFeedIDs                                            *dataloader.Loader[operatorLoaderParam, []*model.Operator]
	PathwaysByFromStopIDs                                         *dataloader.Loader[pathwayLoaderParam, []*model.Pathway]
	PathwaysByIDs                                                 *dataloader.Loader[int, *model.Pathway]
	PathwaysByToStopID                                            *dataloader.Loader[pathwayLoaderParam, []*model.Pathway]
	RouteAttributesByRouteIDs                                     *dataloader.Loader[int, *model.RouteAttribute]
	RouteGeometriesByRouteIDs                                     *dataloader.Loader[routeGeometryLoaderParam, []*model.RouteGeometry]
	RouteHeadwaysByRouteIDs                                       *dataloader.Loader[routeHeadwayLoaderParam, []*model.RouteHeadway]
	RoutesByAgencyIDs                                             *dataloader.Loader[routeLoaderParam, []*model.Route]
	RoutesByFeedVersionIDs                                        *dataloader.Loader[routeLoaderParam, []*model.Route]
	RoutesByIDs                                                   *dataloader.Loader[int, *model.Route]
	RouteStopPatternsByRouteIDs                                   *dataloader.Loader[routeStopPatternLoaderParam, []*model.RouteStopPattern]
	RouteStopsByRouteIDs                                          *dataloader.Loader[routeStopLoaderParam, []*model.RouteStop]
	RouteStopsByStopIDs                                           *dataloader.Loader[routeStopLoaderParam, []*model.RouteStop]
	SegmentPatternsByRouteIDs                                     *dataloader.Loader[segmentPatternLoaderParam, []*model.SegmentPattern]
	SegmentPatternsBySegmentIDs                                   *dataloader.Loader[segmentPatternLoaderParam, []*model.SegmentPattern]
	SegmentsByFeedVersionIDs                                      *dataloader.Loader[segmentLoaderParam, []*model.Segment]
	SegmentsByIDs                                                 *dataloader.Loader[int, *model.Segment]
	SegmentsByRouteIDs                                            *dataloader.Loader[segmentLoaderParam, []*model.Segment]
	ShapesByIDs                                                   *dataloader.Loader[int, *model.Shape]
	StopExternalReferencesByStopIDs                               *dataloader.Loader[int, *model.StopExternalReference]
	StopObservationsByStopIDs                                     *dataloader.Loader[stopObservationLoaderParam, []*model.StopObservation]
	StopPlacesByStopID                                            *dataloader.Loader[model.StopPlaceParam, *model.StopPlace]
	StopsByFeedVersionIDs                                         *dataloader.Loader[stopLoaderParam, []*model.Stop]
	StopsByIDs                                                    *dataloader.Loader[int, *model.Stop]
	StopsByLevelIDs                                               *dataloader.Loader[stopLoaderParam, []*model.Stop]
	StopsByParentStopIDs                                          *dataloader.Loader[stopLoaderParam, []*model.Stop]
	StopsByRouteIDs                                               *dataloader.Loader[stopLoaderParam, []*model.Stop]
	StopTimesByStopIDs                                            *dataloader.Loader[stopTimeLoaderParam, []*model.StopTime]
	StopTimesByTripIDs                                            *dataloader.Loader[tripStopTimeLoaderParam, []*model.StopTime]
	TargetStopsByStopIDs                                          *dataloader.Loader[int, *model.Stop]
	TripsByFeedVersionIDs                                         *dataloader.Loader[tripLoaderParam, []*model.Trip]
	TripsByIDs                                                    *dataloader.Loader[int, *model.Trip]
	TripsByRouteIDs                                               *dataloader.Loader[tripLoaderParam, []*model.Trip]
	ValidationReportErrorExemplarsByValidationReportErrorGroupIDs *dataloader.Loader[validationReportErrorExemplarLoaderParam, []*model.ValidationReportError]
	ValidationReportErrorGroupsByValidationReportIDs              *dataloader.Loader[validationReportErrorGroupLoaderParam, []*model.ValidationReportErrorGroup]
	ValidationReportsByFeedVersionIDs                             *dataloader.Loader[validationReportLoaderParam, []*model.ValidationReport]
}

// NewLoaders instantiates data loaders for the middleware
func NewLoaders(dbf model.Finder, batchSize int, stopTimeBatchSize int) *Loaders {
	if batchSize == 0 {
		batchSize = maxBatch
	}
	if stopTimeBatchSize == 0 {
		stopTimeBatchSize = maxBatch
	}

	loaders := &Loaders{
		AgenciesByFeedVersionIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.AgenciesByFeedVersionIDs,
			func(p agencyLoaderParam) (int, *model.AgencyFilter, *int) {
				return p.FeedVersionID, p.Where, p.Limit
			},
		),
		AgenciesByIDs: withWaitAndCapacity(waitTime, batchSize, dbf.AgenciesByIDs),
		AgenciesByOnestopIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.AgenciesByOnestopIDs,
			func(p agencyLoaderParam) (string, *model.AgencyFilter, *int) {
				a := ""
				if p.OnestopID != nil {
					a = *p.OnestopID
				}
				return a, p.Where, p.Limit
			},
		),
		AgencyPlacesByAgencyIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.AgencyPlacesByAgencyIDs,
			func(p agencyPlaceLoaderParam) (int, *model.AgencyPlaceFilter, *int) {
				return p.AgencyID, p.Where, p.Limit
			},
		),
		BookingRulesByFeedVersionIDs: withWaitAndCapacityGroup(waitTime, batchSize,
			paramGroupAdapter(dbf.BookingRulesByFeedVersionIDs),
			func(p bookingRuleLoaderParam) (int, bool, *int) {
				return p.FeedVersionID, false, p.Limit
			},
		),
		BookingRulesByIDs: withWaitAndCapacity(waitTime, batchSize, dbf.BookingRulesByIDs),
		CalendarDatesByServiceIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.CalendarDatesByServiceIDs,
			func(p calendarDateLoaderParam) (int, *model.CalendarDateFilter, *int) {
				return p.ServiceID, p.Where, p.Limit
			},
		),
		CalendarsByIDs:                  withWaitAndCapacity(waitTime, batchSize, dbf.CalendarsByIDs),
		CensusDatasetLayersByDatasetIDs: withWaitAndCapacity(waitTime, batchSize, dbf.CensusDatasetLayersByDatasetIDs),
		CensusSourceLayersBySourceIDs:   withWaitAndCapacity(waitTime, batchSize, dbf.CensusSourceLayersBySourceIDs),
		CensusFieldsByTableIDs: withWaitAndCapacityGroup(waitTime, batchSize,
			paramGroupAdapter(dbf.CensusFieldsByTableIDs),
			func(p censusFieldLoaderParam) (int, bool, *int) {
				return p.TableID, false, p.Limit
			},
		),
		CensusGeographiesByDatasetIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.CensusGeographiesByDatasetIDs,
			func(p censusDatasetGeographyLoaderParam) (int, *model.CensusDatasetGeographyFilter, *int) {
				return p.DatasetID, p.Where, p.Limit
			},
		),
		CensusGeographiesByEntityIDs: withWaitAndCapacityGroup(waitTime, batchSize,
			func(ctx context.Context, limit *int, param *censusGeographyLoaderParam, keys []int) (ents [][]*model.CensusGeography, err error) {
				return dbf.CensusGeographiesByEntityIDs(ctx, limit, param.Where, param.EntityType, keys)
			},
			func(p censusGeographyLoaderParam) (int, *censusGeographyLoaderParam, *int) {
				rp := censusGeographyLoaderParam{
					EntityType: p.EntityType,
					Where:      p.Where,
				}
				return p.EntityID, &rp, p.Limit
			},
		),
		CensusSourcesByDatasetIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.CensusSourcesByDatasetIDs,
			func(p censusSourceLoaderParam) (int, *model.CensusSourceFilter, *int) {
				return p.DatasetID, p.Where, p.Limit
			},
		),
		CensusTableByIDs:   withWaitAndCapacity(waitTime, batchSize, dbf.CensusTableByIDs),
		CensusLayersByIDs:  withWaitAndCapacity(waitTime, batchSize, dbf.CensusLayersByIDs),
		CensusSourcesByIDs: withWaitAndCapacity(waitTime, batchSize, dbf.CensusSourcesByIDs),
		CensusGeographiesBySourceIDs: withWaitAndCapacityGroup(waitTime, batchSize,
			dbf.CensusGeographiesBySourceIDs,
			func(p censusSourceGeographyLoaderParam) (int, *model.CensusSourceGeographyFilter, *int) {
				return p.SourceID, p.Where, p.Limit
			},
		),
		CensusGeographiesByLayerIDs: withWaitAndCapacityGroup(waitTime, batchSize,
			dbf.CensusGeographiesByLayerIDs,
			func(p censusSourceGeographyLoaderParam) (int, *model.CensusSourceGeographyFilter, *int) {
				return p.LayerID, p.Where, p.Limit
			},
		),
		CensusValuesByGeographyIDs: withWaitAndCapacityGroup(waitTime, batchSize,
			func(ctx context.Context, limit *int, param *censusValueLoaderParam, keys []string) ([][]*model.CensusValue, error) {
				var tnames []string
				for _, t := range strings.Split(param.TableNames, ",") {
					tnames = append(tnames, strings.ToLower(strings.TrimSpace(t)))
				}
				if param.Dataset == nil {
					return nil, nil
				}
				return dbf.CensusValuesByGeographyIDs(ctx, limit, *param.Dataset, tnames, keys)
			},
			func(p censusValueLoaderParam) (string, *censusValueLoaderParam, *int) {
				return p.Geoid, &censusValueLoaderParam{TableNames: p.TableNames, Dataset: p.Dataset}, p.Limit
			},
		),
		FeedFetchesByFeedIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.FeedFetchesByFeedIDs,
			func(p feedFetchLoaderParam) (int, *model.FeedFetchFilter, *int) {
				return p.FeedID, p.Where, p.Limit
			},
		),
		FeedInfosByFeedVersionIDs: withWaitAndCapacityGroup(waitTime, batchSize,
			paramGroupAdapter(dbf.FeedInfosByFeedVersionIDs),
			func(p feedInfoLoaderParam) (int, bool, *int) {
				return p.FeedVersionID, false, p.Limit
			},
		),
		FeedsByIDs: withWaitAndCapacity(waitTime, batchSize, dbf.FeedsByIDs),
		FeedsByOperatorOnestopIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.FeedsByOperatorOnestopIDs,
			func(p feedLoaderParam) (string, *model.FeedFilter, *int) {
				return p.OperatorOnestopID, p.Where, p.Limit
			},
		),
		FeedStatesByFeedIDs: withWaitAndCapacity(waitTime, batchSize, dbf.FeedStatesByFeedIDs),
		FeedVersionFileInfosByFeedVersionIDs: withWaitAndCapacityGroup(waitTime, batchSize,
			paramGroupAdapter(dbf.FeedVersionFileInfosByFeedVersionIDs),
			func(p feedVersionFileInfoLoaderParam) (int, bool, *int) {
				return p.FeedVersionID, false, p.Limit
			},
		),
		FeedVersionGeometryByIDs:              withWaitAndCapacity(waitTime, batchSize, dbf.FeedVersionGeometryByIDs),
		FeedVersionGtfsImportByFeedVersionIDs: withWaitAndCapacity(waitTime, batchSize, dbf.FeedVersionGtfsImportByFeedVersionIDs),
		FeedVersionsByFeedIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.FeedVersionsByFeedIDs,
			func(p feedVersionLoaderParam) (int, *model.FeedVersionFilter, *int) {
				return p.FeedID, p.Where, p.Limit
			},
		),
		FeedVersionsByIDs: withWaitAndCapacity(waitTime, batchSize, dbf.FeedVersionsByIDs),
		FeedVersionServiceLevelsByFeedVersionIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.FeedVersionServiceLevelsByFeedVersionIDs,
			func(p feedVersionServiceLevelLoaderParam) (int, *model.FeedVersionServiceLevelFilter, *int) {
				return p.FeedVersionID, p.Where, p.Limit
			},
		),

		FeedVersionServiceWindowByFeedVersionIDs: withWaitAndCapacity(waitTime, maxBatch, dbf.FeedVersionServiceWindowByFeedVersionIDs),
		FlexStopTimesByTripIDs: withWaitAndCapacityGroup(waitTime, stopTimeBatchSize, dbf.FlexStopTimesByTripIDs,
			func(p tripStopTimeLoaderParam) (model.FVPair, *model.TripStopTimeFilter, *int) {
				return model.FVPair{FeedVersionID: p.FeedVersionID, EntityID: p.TripID}, p.Where, p.Limit
			},
		),
		FlexStopTimesByStopIDs: withWaitAndCapacityGroup(waitTime, stopTimeBatchSize, dbf.FlexStopTimesByStopIDs,
			func(p stopTimeLoaderParam) (model.FVPair, *model.StopTimeFilter, *int) {
				return model.FVPair{FeedVersionID: p.FeedVersionID, EntityID: p.StopID}, p.Where, p.Limit
			},
		),
		FrequenciesByTripIDs: withWaitAndCapacityGroup(waitTime, batchSize,
			paramGroupAdapter(dbf.FrequenciesByTripIDs),
			func(p frequencyLoaderParam) (int, bool, *int) {
				return p.TripID, false, p.Limit
			},
		),

		LevelsByIDs: withWaitAndCapacity(waitTime, batchSize, dbf.LevelsByIDs),
		LevelsByParentStationIDs: withWaitAndCapacityGroup(waitTime, batchSize,
			paramGroupAdapter(dbf.LevelsByParentStationIDs),
			func(p levelLoaderParam) (int, bool, *int) {
				return p.ParentStationID, false, p.Limit
			},
		),

		LocationGroupsByFeedVersionIDs: withWaitAndCapacityGroup(waitTime, batchSize,
			paramGroupAdapter(dbf.LocationGroupsByFeedVersionIDs),
			func(p locationGroupLoaderParam) (int, bool, *int) {
				return p.FeedVersionID, false, p.Limit
			},
		),
		LocationGroupsByIDs: withWaitAndCapacity(waitTime, batchSize, dbf.LocationGroupsByIDs),
		LocationsByFeedVersionIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.LocationsByFeedVersionIDs,
			func(p locationLoaderParam) (int, *model.LocationFilter, *int) {
				return p.FeedVersionID, p.Where, p.Limit
			},
		),
		LocationsByIDs: withWaitAndCapacity(waitTime, batchSize, dbf.LocationsByIDs),

		OperatorsByAgencyIDs: withWaitAndCapacity(waitTime, batchSize, dbf.OperatorsByAgencyIDs),
		OperatorsByCOIFs:     withWaitAndCapacity(waitTime, batchSize, dbf.OperatorsByCOIFs),
		OperatorsByFeedIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.OperatorsByFeedIDs,
			func(p operatorLoaderParam) (int, *model.OperatorFilter, *int) {
				return p.FeedID, p.Where, p.Limit
			},
		),
		PathwaysByIDs: withWaitAndCapacity(waitTime, batchSize, dbf.PathwaysByIDs),
		PathwaysByFromStopIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.PathwaysByFromStopIDs,
			func(p pathwayLoaderParam) (int, *model.PathwayFilter, *int) {
				return p.FromStopID, p.Where, p.Limit
			},
		),
		PathwaysByToStopID: withWaitAndCapacityGroup(waitTime, batchSize, dbf.PathwaysByToStopIDs,
			func(p pathwayLoaderParam) (int, *model.PathwayFilter, *int) {
				return p.ToStopID, p.Where, p.Limit
			},
		),
		RouteAttributesByRouteIDs: withWaitAndCapacity(waitTime, batchSize, dbf.RouteAttributesByRouteIDs),
		RouteGeometriesByRouteIDs: withWaitAndCapacityGroup(waitTime, batchSize,
			paramGroupAdapter(dbf.RouteGeometriesByRouteIDs),
			func(p routeGeometryLoaderParam) (int, bool, *int) {
				return p.RouteID, false, p.Limit
			},
		),
		RouteHeadwaysByRouteIDs: withWaitAndCapacityGroup(waitTime, batchSize,
			paramGroupAdapter(dbf.RouteHeadwaysByRouteIDs),
			func(p routeHeadwayLoaderParam) (int, bool, *int) {
				return p.RouteID, false, p.Limit
			},
		),
		RoutesByAgencyIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.RoutesByAgencyIDs,
			func(p routeLoaderParam) (int, *model.RouteFilter, *int) {
				return p.AgencyID, p.Where, p.Limit
			},
		),
		RoutesByFeedVersionIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.RoutesByFeedVersionIDs,
			func(p routeLoaderParam) (int, *model.RouteFilter, *int) {
				return p.FeedVersionID, p.Where, p.Limit
			},
		),
		RoutesByIDs: withWaitAndCapacity(waitTime, batchSize, dbf.RoutesByIDs),
		RouteStopPatternsByRouteIDs: withWaitAndCapacityGroup(waitTime, batchSize,
			paramGroupAdapter(dbf.RouteStopPatternsByRouteIDs),
			func(p routeStopPatternLoaderParam) (int, bool, *int) {
				return p.RouteID, false, nil
			},
		),
		RouteStopsByRouteIDs: withWaitAndCapacityGroup(waitTime, batchSize,
			paramGroupAdapter(dbf.RouteStopsByRouteIDs),
			func(p routeStopLoaderParam) (int, bool, *int) {
				return p.RouteID, false, p.Limit
			},
		),
		RouteStopsByStopIDs: withWaitAndCapacityGroup(waitTime, batchSize,
			paramGroupAdapter(dbf.RouteStopsByStopIDs),
			func(p routeStopLoaderParam) (int, bool, *int) {
				return p.StopID, false, p.Limit
			},
		),
		SegmentPatternsByRouteIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.SegmentPatternsByRouteIDs,
			func(p segmentPatternLoaderParam) (int, *model.SegmentPatternFilter, *int) {
				return p.RouteID, p.Where, p.Limit
			},
		),
		SegmentPatternsBySegmentIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.SegmentPatternsBySegmentIDs,
			func(p segmentPatternLoaderParam) (int, *model.SegmentPatternFilter, *int) {
				return p.SegmentID, p.Where, p.Limit
			},
		),
		SegmentsByFeedVersionIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.SegmentsByFeedVersionIDs,
			func(p segmentLoaderParam) (int, *model.SegmentFilter, *int) {
				return p.FeedVersionID, p.Where, p.Limit
			},
		),
		SegmentsByIDs: withWaitAndCapacity(waitTime, batchSize, dbf.SegmentsByIDs),
		SegmentsByRouteIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.SegmentsByRouteIDs,
			func(p segmentLoaderParam) (int, *model.SegmentFilter, *int) {
				return p.RouteID, p.Where, p.Limit
			},
		),
		ShapesByIDs:                     withWaitAndCapacity(waitTime, batchSize, dbf.ShapesByIDs),
		StopExternalReferencesByStopIDs: withWaitAndCapacity(waitTime, batchSize, dbf.StopExternalReferencesByStopIDs),
		StopObservationsByStopIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.StopObservationsByStopIDs,
			func(p stopObservationLoaderParam) (int, *model.StopObservationFilter, *int) {
				return p.StopID, p.Where, p.Limit
			},
		),
		StopPlacesByStopID: withWaitAndCapacity(waitTime, batchSize, dbf.StopPlacesByStopID),
		StopsByFeedVersionIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.StopsByFeedVersionIDs,
			func(p stopLoaderParam) (int, *model.StopFilter, *int) {
				return p.FeedVersionID, p.Where, p.Limit
			},
		),
		StopsByIDs: withWaitAndCapacity(waitTime, batchSize, dbf.StopsByIDs),
		StopsByLevelIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.StopsByLevelIDs,
			func(p stopLoaderParam) (int, *model.StopFilter, *int) {
				return p.LevelID, p.Where, p.Limit
			},
		),
		StopsByParentStopIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.StopsByParentStopIDs,
			func(p stopLoaderParam) (int, *model.StopFilter, *int) {
				return p.ParentStopID, p.Where, p.Limit
			},
		),
		StopsByRouteIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.StopsByRouteIDs,
			func(p stopLoaderParam) (int, *model.StopFilter, *int) {
				return p.RouteID, p.Where, p.Limit
			},
		),
		StopTimesByStopIDs: withWaitAndCapacityGroup(stopTimeBatchWaitTime, stopTimeBatchSize, dbf.StopTimesByStopIDs,
			func(p stopTimeLoaderParam) (model.FVPair, *model.StopTimeFilter, *int) {
				return model.FVPair{FeedVersionID: p.FeedVersionID, EntityID: p.StopID}, p.Where, p.Limit
			},
		),
		StopTimesByTripIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.StopTimesByTripIDs,
			func(p tripStopTimeLoaderParam) (model.FVPair, *model.TripStopTimeFilter, *int) {
				return model.FVPair{FeedVersionID: p.FeedVersionID, EntityID: p.TripID}, p.Where, p.Limit
			},
		),
		TargetStopsByStopIDs: withWaitAndCapacity(waitTime, batchSize, dbf.TargetStopsByStopIDs),
		TripsByFeedVersionIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.TripsByFeedVersionIDs,
			func(p tripLoaderParam) (int, *model.TripFilter, *int) {
				return p.FeedVersionID, p.Where, p.Limit
			},
		),
		TripsByIDs: withWaitAndCapacity(waitTime, batchSize, dbf.TripsByIDs),
		TripsByRouteIDs: withWaitAndCapacityGroup(waitTime, batchSize, dbf.TripsByRouteIDs,
			func(p tripLoaderParam) (model.FVPair, *model.TripFilter, *int) {
				return model.FVPair{EntityID: p.RouteID, FeedVersionID: p.FeedVersionID}, p.Where, p.Limit
			},
		),
		ValidationReportErrorExemplarsByValidationReportErrorGroupIDs: withWaitAndCapacityGroup(waitTime, batchSize,
			paramGroupAdapter(dbf.ValidationReportErrorExemplarsByValidationReportErrorGroupIDs),
			func(p validationReportErrorExemplarLoaderParam) (int, bool, *int) {
				return p.ValidationReportGroupID, false, p.Limit
			},
		),
		ValidationReportErrorGroupsByValidationReportIDs: withWaitAndCapacityGroup(waitTime, batchSize,
			paramGroupAdapter(dbf.ValidationReportErrorGroupsByValidationReportIDs),
			func(p validationReportErrorGroupLoaderParam) (int, bool, *int) {
				return p.ValidationReportID, false, p.Limit
			},
		),
		ValidationReportsByFeedVersionIDs: withWaitAndCapacityGroup(waitTime, batchSize,
			dbf.ValidationReportsByFeedVersionIDs,
			func(p validationReportLoaderParam) (int, *model.ValidationReportFilter, *int) {
				return p.FeedVersionID, p.Where, p.Limit
			},
		),
	}
	return loaders
}

func loaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This is per request scoped loaders/cache
		// Is this OK to use as a long term cache?
		ctx := r.Context()
		cfg := model.ForContext(ctx)
		loaders := NewLoaders(cfg.Finder, cfg.LoaderBatchSize, cfg.LoaderStopTimeBatchSize)
		nextCtx := context.WithValue(ctx, loadersKey, loaders)
		r = r.WithContext(nextCtx)
		next.ServeHTTP(w, r)
	})
}

// LoaderFor returns the dataloader for a given context
func LoaderFor(ctx context.Context) *Loaders {
	return ctx.Value(loadersKey).(*Loaders)
}

// withWaitAndCapacity is a helper that sets a default with time, with less manually specifying type params
func withWaitAndCapacity[
	T any,
	ParamT comparable,
](
	d time.Duration,
	size int,
	cb func(context.Context, []ParamT) ([]T, []error),
) *dataloader.Loader[ParamT, T] {
	return dataloader.NewBatchedLoader(
		unwrapResult(cb),
		dataloader.WithWait[ParamT, T](d),
		dataloader.WithBatchCapacity[ParamT, T](size),
	)
}

// withWaitAndCapacityGroup is a helper that sets a default with time, with less manually specifying type params
func withWaitAndCapacityGroup[
	T any,
	ParamT comparable,
	W any,
	K comparable,
](
	d time.Duration,
	size int,
	queryFunc func(context.Context, *int, W, []K) ([][]T, error),
	paramFunc func(ParamT) (K, W, *int),
) *dataloader.Loader[ParamT, []T] {
	return dataloader.NewBatchedLoader(
		unwrapResult(paramGroupQuery(paramFunc, queryFunc)),
		dataloader.WithWait[ParamT, []T](d),
		dataloader.WithBatchCapacity[ParamT, []T](size),
	)
}

// unwrap function adapts existing Finder methods to dataloader Result type
func unwrapResult[
	T any,
	ParamT comparable,
](
	cb func(context.Context, []ParamT) ([]T, []error),
) func(context.Context, []ParamT) []*dataloader.Result[T] {
	x := func(ctx context.Context, ps []ParamT) []*dataloader.Result[T] {
		a, errs := cb(ctx, ps)
		if len(a) != len(ps) {
			log.For(ctx).Trace().Msgf("error in dataloader, result len %d did not match param length %d", len(a), len(ps))
			return nil
		}
		ret := make([]*dataloader.Result[T], len(ps))
		for idx := range ps {
			var err error
			if idx < len(errs) {
				err = errs[idx]
			}
			var data T
			if idx < len(a) {
				data = a[idx]
			}
			ret[idx] = &dataloader.Result[T]{Data: data, Error: err}
		}
		return ret
	}
	return x
}

////////////

func paramGroupAdapter[
	K comparable,
	T any,
](inner func(context.Context, *int, []K) ([][]T, error)) func(context.Context, *int, bool, []K) ([][]T, error) {
	return func(ctx context.Context, limit *int, where bool, keys []K) ([][]T, error) {
		return inner(ctx, limit, keys)
	}
}

func paramGroupQuery[
	K comparable,
	ParamT any,
	W any,
	T any,
](
	paramFunc func(ParamT) (K, W, *int),
	queryFunc func(context.Context, *int, W, []K) ([][]T, error),
) func(context.Context, []ParamT) ([][]T, []error) {
	return func(ctx context.Context, params []ParamT) ([][]T, []error) {
		// Create return value
		ret := make([][]T, len(params))
		errs := make([]error, len(params))

		// Group params by JSON representation
		type paramGroupItem[K comparable, M any] struct {
			Limit *int
			Where M
		}
		type paramGroup[K comparable, M any] struct {
			Index []int
			Keys  []K
			Limit *int
			Where M
		}
		paramGroups := map[string]paramGroup[K, W]{}
		for i, param := range params {
			// Get values from supplied func
			key, where, limit := paramFunc(param)

			// Convert to paramGroupItem
			item := paramGroupItem[K, W]{
				Limit: limit,
				Where: where,
			}

			// Use the JSON representation of Where and Limit as the key
			jj, err := json.Marshal(paramGroupItem[K, W]{Where: item.Where, Limit: item.Limit})
			if err != nil {
				// TODO: log and expand error
				errs[i] = err
				continue
			}
			paramGroupKey := string(jj)

			// Add index and key
			a, ok := paramGroups[paramGroupKey]
			if !ok {
				a = paramGroup[K, W]{Where: item.Where, Limit: item.Limit}
			}
			a.Index = append(a.Index, i)
			a.Keys = append(a.Keys, key)
			paramGroups[paramGroupKey] = a
		}

		// Process each param group
		for _, pgroup := range paramGroups {
			// Run query function
			ents, err := queryFunc(ctx, pgroup.Limit, pgroup.Where, pgroup.Keys)
			if err != nil {
				panic(err)
			}

			// Group using keyFunc and merge into output
			// This limit is just for safety; other limits should be set at the query/resolver level
			limit := 1000
			if a := resolverCheckLimitMax(pgroup.Limit, 100_000); a != nil {
				limit = *a
			}
			for resultIdx, idx := range pgroup.Index {
				a := ents[resultIdx]
				if len(a) > limit {
					a = a[0:limit]
				}
				ret[idx] = a
			}
		}
		return ret, errs
	}
}
