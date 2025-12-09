package gql

import (
	"github.com/interline-io/transitland-lib/server/model"
)

// This file contains parameters that can be passed to methods for finding/selecting/grouping entities
// These are distinct from WHERE graphql input filters, which are available to users.

type frequencyLoaderParam struct {
	TripID int
	Limit  *int
}

type feedVersionFileInfoLoaderParam struct {
	FeedVersionID int
	Limit         *int
}

type feedVersionLoaderParam struct {
	FeedID int
	Limit  *int
	Where  *model.FeedVersionFilter
}

type feedVersionServiceLevelLoaderParam struct {
	FeedVersionID int
	Limit         *int
	Where         *model.FeedVersionServiceLevelFilter
}

type feedInfoLoaderParam struct {
	FeedVersionID int
	Limit         *int
}

type pathwayLoaderParam struct {
	FeedVersionID int
	FromStopID    int
	ToStopID      int
	Limit         *int
	Where         *model.PathwayFilter
}

type stopTimeLoaderParam struct {
	TripID          int
	StopID          int
	LocationID      int
	LocationGroupID int
	FeedVersionID   int
	Limit           *int
	Where           *model.StopTimeFilter
}

type tripStopTimeLoaderParam struct {
	TripID        int
	FeedVersionID int
	Limit         *int
	StartTime     *int
	EndTime       *int
	Where         *model.TripStopTimeFilter
}

type agencyLoaderParam struct {
	FeedVersionID int
	Limit         *int
	OnestopID     *string
	Where         *model.AgencyFilter
}

type routeLoaderParam struct {
	AgencyID      int
	FeedVersionID int
	Limit         *int
	Where         *model.RouteFilter
}

type routeStopLoaderParam struct {
	RouteID int
	StopID  int
	Limit   *int
}

type routeHeadwayLoaderParam struct {
	RouteID int
	Limit   *int
}

type routeGeometryLoaderParam struct {
	RouteID int
	Limit   *int
}

type tripLoaderParam struct {
	FeedVersionID int
	RouteID       int
	Limit         *int
	ServiceWindow *model.ServiceWindow
	Where         *model.TripFilter
}

type stopLoaderParam struct {
	FeedVersionID int
	ParentStopID  int
	AgencyID      int
	LevelID       int
	Limit         *int
	Where         *model.StopFilter
	RouteID       int
}

type levelLoaderParam struct {
	ParentStationID int
	Limit           *int
}

type feedLoaderParam struct {
	OperatorOnestopID string
	Limit             *int
	Where             *model.FeedFilter
}

type feedFetchLoaderParam struct {
	FeedID int
	Limit  *int
	Where  *model.FeedFetchFilter
}

type agencyPlaceLoaderParam struct {
	AgencyID int
	Limit    *int
	Where    *model.AgencyPlaceFilter
}

type operatorLoaderParam struct {
	FeedID int
	Limit  *int
	Where  *model.OperatorFilter
}

type stopObservationLoaderParam struct {
	StopID int
	Limit  *int
	Where  *model.StopObservationFilter
}

type calendarDateLoaderParam struct {
	ServiceID int
	Limit     *int
	Where     *model.CalendarDateFilter
}

type censusGeographyLoaderParam struct {
	EntityType string
	EntityID   int
	DatasetID  int
	SourceID   int
	LayerID    int
	Limit      *int
	Where      *model.CensusGeographyFilter
}

type censusDatasetGeographyLoaderParam struct {
	DatasetID int
	Limit     *int
	Where     *model.CensusDatasetGeographyFilter
}

type censusSourceGeographyLoaderParam struct {
	SourceID int
	LayerID  int
	Limit    *int
	Where    *model.CensusSourceGeographyFilter
}

type censusValueLoaderParam struct {
	Dataset    *string
	Geoid      string
	TableNames string // these have to be comma joined for now, []string cant be used as map key
	Limit      *int
}

type censusFieldLoaderParam struct {
	Limit   *int
	TableID int
}

type censusSourceLoaderParam struct {
	DatasetID int
	Limit     *int
	Where     *model.CensusSourceFilter
}

type routeStopPatternLoaderParam struct {
	RouteID int
}

type segmentPatternLoaderParam struct {
	SegmentID int
	RouteID   int
	Limit     *int
	Where     *model.SegmentPatternFilter
}

type segmentLoaderParam struct {
	FeedVersionID int
	RouteID       int
	Layer         string
	Limit         *int
	Where         *model.SegmentFilter
}

type validationReportLoaderParam struct {
	FeedVersionID int
	Limit         *int
	Where         *model.ValidationReportFilter
}

type validationReportErrorExemplarLoaderParam struct {
	ValidationReportGroupID int
	Limit                   *int
}

type validationReportErrorGroupLoaderParam struct {
	ValidationReportID int
	Limit              *int
}

type bookingRuleLoaderParam struct {
	FeedVersionID int
	Limit         *int
	Where         *model.BookingRuleFilter
}

type locationGroupLoaderParam struct {
	FeedVersionID int
	Limit         *int
	Where         *model.LocationGroupFilter
}

type stopsByLocationGroupLoaderParam struct {
	LocationGroupID int
	Limit           *int
}

type locationLoaderParam struct {
	FeedVersionID int
	Limit         *int
	Where         *model.LocationFilter
}
