package model

import (
	"context"
	"io"
	"time"

	"github.com/interline-io/transitland-lib/internal/gbfs"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/interline-io/transitland-mw/auth/authz"
)

type FVPair struct {
	FeedVersionID int
	EntityID      int
}

// Finder provides all necessary database methods
type Finder interface {
	PermFinder
	EntityFinder
	EntityLoader
	EntityMutator
}

type PermFinder interface {
	PermFilter(context.Context) *PermFilter
}

// Finder handles basic queries
type EntityFinder interface {
	FindAgencies(context.Context, *int, *Cursor, []int, *AgencyFilter) ([]*Agency, error)
	FindRoutes(context.Context, *int, *Cursor, []int, *RouteFilter) ([]*Route, error)
	FindStops(context.Context, *int, *Cursor, []int, *StopFilter) ([]*Stop, error)
	FindTrips(context.Context, *int, *Cursor, []int, *TripFilter) ([]*Trip, error)
	FindFeedVersions(context.Context, *int, *Cursor, []int, *FeedVersionFilter) ([]*FeedVersion, error)
	FindFeeds(context.Context, *int, *Cursor, []int, *FeedFilter) ([]*Feed, error)
	FindOperators(context.Context, *int, *Cursor, []int, *OperatorFilter) ([]*Operator, error)
	FindPlaces(context.Context, *int, *Cursor, []int, *PlaceAggregationLevel, *PlaceFilter) ([]*Place, error)
	FindCensusDatasets(context.Context, *int, *Cursor, []int, *CensusDatasetFilter) ([]*CensusDataset, error)
	RouteStopBuffer(context.Context, *int, *float64, int) ([]*RouteStopBuffer, error)
	FindFeedVersionServiceWindow(context.Context, int) (*ServiceWindow, error)
	DBX() tldb.Ext // escape hatch, for now
}

type EntityLoader interface {
	AgenciesByFeedVersionIDs(ctx context.Context, limit *int, where *AgencyFilter, feedVersionIds []int) ([][]*Agency, error)
	AgenciesByIDs(context.Context, []int) ([]*Agency, []error)
	AgenciesByOnestopIDs(context.Context, *int, *AgencyFilter, []string) ([][]*Agency, error)
	AgencyPlacesByAgencyIDs(context.Context, *int, *AgencyPlaceFilter, []int) ([][]*AgencyPlace, error)
	CalendarDatesByServiceIDs(context.Context, *int, *CalendarDateFilter, []int) ([][]*CalendarDate, error)
	CalendarsByIDs(context.Context, []int) ([]*Calendar, []error)
	CensusDatasetLayersByDatasetIDs(context.Context, []int) ([][]*CensusLayer, []error)
	CensusFieldsByTableIDs(context.Context, *int, []int) ([][]*CensusField, error)
	CensusGeographiesByDatasetIDs(context.Context, *int, *CensusDatasetGeographyFilter, []int) ([][]*CensusGeography, error)
	CensusGeographiesByEntityIDs(context.Context, *int, *CensusGeographyFilter, string, []int) ([][]*CensusGeography, error)
	CensusGeographiesByLayerIDs(context.Context, *int, *CensusSourceGeographyFilter, []int) ([][]*CensusGeography, error)
	CensusGeographiesBySourceIDs(context.Context, *int, *CensusSourceGeographyFilter, []int) ([][]*CensusGeography, error)
	CensusLayersByIDs(context.Context, []int) ([]*CensusLayer, []error)
	CensusSourcesByIDs(context.Context, []int) ([]*CensusSource, []error)
	CensusSourceLayersBySourceIDs(context.Context, []int) ([][]*CensusLayer, []error)
	CensusSourcesByDatasetIDs(context.Context, *int, *CensusSourceFilter, []int) ([][]*CensusSource, error)
	CensusTableByIDs(context.Context, []int) ([]*CensusTable, []error)
	CensusValuesByGeographyIDs(context.Context, *int, []string, []string) ([][]*CensusValue, error)
	FeedFetchesByFeedIDs(context.Context, *int, *FeedFetchFilter, []int) ([][]*FeedFetch, error)
	FeedInfosByFeedVersionIDs(context.Context, *int, []int) ([][]*FeedInfo, error)
	FeedsByIDs(context.Context, []int) ([]*Feed, []error)
	FeedsByOperatorOnestopIDs(context.Context, *int, *FeedFilter, []string) ([][]*Feed, error)
	FeedStatesByFeedIDs(context.Context, []int) ([]*FeedState, []error)
	FeedVersionFileInfosByFeedVersionIDs(context.Context, *int, []int) ([][]*FeedVersionFileInfo, error)
	FeedVersionGeometryByIDs(context.Context, []int) ([]*tt.Polygon, []error)
	FeedVersionGtfsImportByFeedVersionIDs(context.Context, []int) ([]*FeedVersionGtfsImport, []error)
	FeedVersionsByFeedIDs(context.Context, *int, *FeedVersionFilter, []int) ([][]*FeedVersion, error)
	FeedVersionsByIDs(context.Context, []int) ([]*FeedVersion, []error)
	FeedVersionServiceLevelsByFeedVersionIDs(context.Context, *int, *FeedVersionServiceLevelFilter, []int) ([][]*FeedVersionServiceLevel, error)
	FeedVersionServiceWindowByFeedVersionIDs(context.Context, []int) ([]*FeedVersionServiceWindow, []error)
	FrequenciesByTripIDs(context.Context, *int, []int) ([][]*Frequency, error)
	LevelsByIDs(context.Context, []int) ([]*Level, []error)
	LevelsByParentStationIDs(context.Context, *int, []int) ([][]*Level, error)
	OperatorsByAgencyIDs(context.Context, []int) ([]*Operator, []error)
	OperatorsByCOIFs(context.Context, []int) ([]*Operator, []error)
	OperatorsByFeedIDs(context.Context, *int, *OperatorFilter, []int) ([][]*Operator, error)
	PathwaysByFromStopIDs(context.Context, *int, *PathwayFilter, []int) ([][]*Pathway, error)
	PathwaysByIDs(context.Context, []int) ([]*Pathway, []error)
	PathwaysByToStopIDs(context.Context, *int, *PathwayFilter, []int) ([][]*Pathway, error)
	RouteAttributesByRouteIDs(context.Context, []int) ([]*RouteAttribute, []error)
	RouteGeometriesByRouteIDs(context.Context, *int, []int) ([][]*RouteGeometry, error)
	RouteHeadwaysByRouteIDs(context.Context, *int, []int) ([][]*RouteHeadway, error)
	RoutesByAgencyIDs(context.Context, *int, *RouteFilter, []int) ([][]*Route, error)
	RoutesByFeedVersionIDs(context.Context, *int, *RouteFilter, []int) ([][]*Route, error)
	RoutesByIDs(context.Context, []int) ([]*Route, []error)
	RouteStopPatternsByRouteIDs(context.Context, *int, []int) ([][]*RouteStopPattern, error)
	RouteStopsByRouteIDs(context.Context, *int, []int) ([][]*RouteStop, error)
	RouteStopsByStopIDs(context.Context, *int, []int) ([][]*RouteStop, error)
	SegmentPatternsByRouteIDs(context.Context, *int, *SegmentPatternFilter, []int) ([][]*SegmentPattern, error)
	SegmentPatternsBySegmentIDs(context.Context, *int, *SegmentPatternFilter, []int) ([][]*SegmentPattern, error)
	SegmentsByFeedVersionIDs(context.Context, *int, *SegmentFilter, []int) ([][]*Segment, error)
	SegmentsByIDs(context.Context, []int) ([]*Segment, []error)
	SegmentsByRouteIDs(context.Context, *int, *SegmentFilter, []int) ([][]*Segment, error)
	ShapesByIDs(context.Context, []int) ([]*Shape, []error)
	StopExternalReferencesByStopIDs(context.Context, []int) ([]*StopExternalReference, []error)
	StopObservationsByStopIDs(context.Context, *int, *StopObservationFilter, []int) ([][]*StopObservation, error)
	StopPlacesByStopID(context.Context, []StopPlaceParam) ([]*StopPlace, []error)
	StopsByFeedVersionIDs(context.Context, *int, *StopFilter, []int) ([][]*Stop, error)
	StopsByIDs(context.Context, []int) ([]*Stop, []error)
	StopsByLevelIDs(context.Context, *int, *StopFilter, []int) ([][]*Stop, error)
	StopsByParentStopIDs(context.Context, *int, *StopFilter, []int) ([][]*Stop, error)
	StopsByRouteIDs(context.Context, *int, *StopFilter, []int) ([][]*Stop, error)
	StopTimesByStopIDs(context.Context, *int, *StopTimeFilter, []FVPair) ([][]*StopTime, error)
	StopTimesByTripIDs(context.Context, *int, *TripStopTimeFilter, []FVPair) ([][]*StopTime, error)
	TargetStopsByStopIDs(context.Context, []int) ([]*Stop, []error)
	TripsByFeedVersionIDs(context.Context, *int, *TripFilter, []int) ([][]*Trip, error)
	TripsByIDs(context.Context, []int) ([]*Trip, []error)
	TripsByRouteIDs(context.Context, *int, *TripFilter, []FVPair) ([][]*Trip, error)
	ValidationReportErrorExemplarsByValidationReportErrorGroupIDs(context.Context, *int, []int) ([][]*ValidationReportError, error)
	ValidationReportErrorGroupsByValidationReportIDs(context.Context, *int, []int) ([][]*ValidationReportErrorGroup, error)
	ValidationReportsByFeedVersionIDs(context.Context, *int, *ValidationReportFilter, []int) ([][]*ValidationReport, error)
}

type EntityMutator interface {
	StopCreate(ctx context.Context, input StopSetInput) (int, error)
	StopUpdate(ctx context.Context, input StopSetInput) (int, error)
	StopDelete(ctx context.Context, id int) error
	PathwayCreate(ctx context.Context, input PathwaySetInput) (int, error)
	PathwayUpdate(ctx context.Context, input PathwaySetInput) (int, error)
	PathwayDelete(ctx context.Context, id int) error
	LevelCreate(ctx context.Context, input LevelSetInput) (int, error)
	LevelUpdate(ctx context.Context, input LevelSetInput) (int, error)
	LevelDelete(ctx context.Context, id int) error
}

// RTFinder manages and looks up RT data
type RTFinder interface {
	AddData(context.Context, string, []byte) error
	FindTrip(context.Context, *Trip) *pb.TripUpdate
	MakeTrip(context.Context, *Trip) (*Trip, error)
	FindAlertsForTrip(context.Context, *Trip, *int, *bool) []*Alert
	FindAlertsForStop(context.Context, *Stop, *int, *bool) []*Alert
	FindAlertsForRoute(context.Context, *Route, *int, *bool) []*Alert
	FindAlertsForAgency(context.Context, *Agency, *int, *bool) []*Alert
	GetAddedTripsForStop(context.Context, *Stop) []*pb.TripUpdate
	FindStopTimeUpdate(context.Context, *Trip, *StopTime) (*RTStopTimeUpdate, bool)
	// lookup cache methods
	StopTimezone(context.Context, int, string) (*time.Location, bool)
	GetGtfsTripID(context.Context, int) (string, bool)
	GetMessage(context.Context, string, string) (*pb.FeedMessage, bool)
}

// GbfsFinder manages and looks up GBFS data
type GbfsFinder interface {
	AddData(context.Context, string, gbfs.GbfsFeed) error
	FindBikes(context.Context, *int, *GbfsBikeRequest) ([]*GbfsFreeBikeStatus, error)
	FindDocks(context.Context, *int, *GbfsDockRequest) ([]*GbfsStationInformation, error)
}

type Checker interface {
	authz.CheckerServer
}

type Actions interface {
	StaticFetch(context.Context, string, io.Reader, string) (*FeedVersionFetchResult, error)
	RTFetch(context.Context, string, string, string, string) error
	GbfsFetch(context.Context, string, string) error
	ValidateUpload(context.Context, io.Reader, *string, []string) (*ValidationReport, error)
	FeedVersionUnimport(context.Context, int) (*FeedVersionUnimportResult, error)
	FeedVersionImport(context.Context, int) (*FeedVersionImportResult, error)
	FeedVersionUpdate(context.Context, FeedVersionSetInput) (int, error)
	FeedVersionDelete(context.Context, int) (*FeedVersionDeleteResult, error)
}
