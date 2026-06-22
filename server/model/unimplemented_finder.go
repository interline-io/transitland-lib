package model

import (
	"context"
	"errors"

	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
)

// ErrNotImplemented is returned by UnimplementedFinder methods.
var ErrNotImplemented = errors.New("not implemented")

// UnimplementedFinder implements the full Finder interface with stubs that
// return ErrNotImplemented (or empty results). Embed it in a partial Finder
// implementation — e.g. an in-memory or test finder — and override only the
// methods you support; the rest satisfy the interface without boilerplate.
type UnimplementedFinder struct{}

var _ Finder = UnimplementedFinder{}

// PermFinder

func (UnimplementedFinder) PermFilter(context.Context) *PermFilter { return nil }

// EntityFinder

func (UnimplementedFinder) FindAgencies(context.Context, *int, *Cursor, []int, *AgencyFilter) ([]*Agency, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FindRoutes(context.Context, *int, *Cursor, []int, *RouteFilter) ([]*Route, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FindStops(context.Context, *int, *Cursor, []int, *StopFilter) ([]*Stop, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FindTrips(context.Context, *int, *Cursor, []int, *TripFilter) ([]*Trip, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FindFeedVersions(context.Context, *int, *Cursor, []int, *FeedVersionFilter) ([]*FeedVersion, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FindFeeds(context.Context, *int, *Cursor, []int, *FeedFilter) ([]*Feed, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FindOperators(context.Context, *int, *Cursor, []int, *OperatorFilter) ([]*Operator, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FindPlaces(context.Context, *int, *Cursor, []int, *PlaceAggregationLevel, *PlaceFilter) ([]*Place, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FindCensusDatasets(context.Context, *int, *Cursor, []int, *CensusDatasetFilter) ([]*CensusDataset, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FindCensusValuesByDatasetID(context.Context, *int, CensusCursor, int, *CensusDatasetValueFilter) ([]*CensusValue, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) RouteStopBuffer(context.Context, *int, *float64, int) ([]*RouteStopBuffer, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FindFeedVersionServiceWindow(context.Context, int) (*ServiceWindow, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) DBX() tldb.Ext { return nil }

// EntityLoader

func (UnimplementedFinder) AgenciesByFeedVersionIDs(context.Context, *int, *AgencyFilter, []int) ([][]*Agency, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) AgenciesByIDs(context.Context, []int) ([]*Agency, []error) { return nil, nil }
func (UnimplementedFinder) AgenciesByOnestopIDs(context.Context, *int, *AgencyFilter, []string) ([][]*Agency, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) AgencyPlacesByAgencyIDs(context.Context, *int, *AgencyPlaceFilter, []int) ([][]*AgencyPlace, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) BookingRulesByFeedVersionIDs(context.Context, *int, *BookingRuleFilter, []int) ([][]*BookingRule, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) BookingRulesByIDs(context.Context, []int) ([]*BookingRule, []error) {
	return nil, nil
}
func (UnimplementedFinder) CalendarDatesByServiceIDs(context.Context, *int, *CalendarDateFilter, []int) ([][]*CalendarDate, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) CalendarsByIDs(context.Context, []int) ([]*Calendar, []error) {
	return nil, nil
}
func (UnimplementedFinder) CensusDatasetLayersByDatasetIDs(context.Context, []int) ([][]*CensusLayer, []error) {
	return nil, nil
}
func (UnimplementedFinder) CensusFieldsByTableIDs(context.Context, *int, []int) ([][]*CensusField, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) CensusGeographiesByDatasetIDs(context.Context, *int, *CensusDatasetGeographyFilter, []int) ([][]*CensusGeography, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) CensusGeographiesByEntityIDs(context.Context, *int, *CensusGeographyFilter, string, []int) ([][]*CensusGeography, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) CensusGeographiesByLayerIDs(context.Context, *int, *CensusSourceGeographyFilter, []int) ([][]*CensusGeography, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) CensusGeographiesBySourceIDs(context.Context, *int, *CensusSourceGeographyFilter, []int) ([][]*CensusGeography, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) CensusLayersByIDs(context.Context, []int) ([]*CensusLayer, []error) {
	return nil, nil
}
func (UnimplementedFinder) CensusSourcesByIDs(context.Context, []int) ([]*CensusSource, []error) {
	return nil, nil
}
func (UnimplementedFinder) CensusSourceLayersBySourceIDs(context.Context, []int) ([][]*CensusLayer, []error) {
	return nil, nil
}
func (UnimplementedFinder) CensusSourcesByDatasetIDs(context.Context, *int, *CensusSourceFilter, []int) ([][]*CensusSource, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) CensusTablesByDatasetIDs(context.Context, *int, *CensusTableFilter, []int) ([][]*CensusTable, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) CensusTableByIDs(context.Context, []int) ([]*CensusTable, []error) {
	return nil, nil
}
func (UnimplementedFinder) CensusValuesByGeographyIDs(context.Context, *int, string, []string, []string) ([][]*CensusValue, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FeedFetchesByFeedIDs(context.Context, *int, *FeedFetchFilter, []int) ([][]*FeedFetch, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FeedInfosByFeedVersionIDs(context.Context, *int, []int) ([][]*FeedInfo, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FeedsByIDs(context.Context, []int) ([]*Feed, []error) { return nil, nil }
func (UnimplementedFinder) FeedsByOperatorOnestopIDs(context.Context, *int, *FeedFilter, []string) ([][]*Feed, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FeedStatesByFeedIDs(context.Context, []int) ([]*FeedState, []error) {
	return nil, nil
}
func (UnimplementedFinder) FeedVersionFileInfosByFeedVersionIDs(context.Context, *int, []int) ([][]*FeedVersionFileInfo, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FeedVersionGeometryByIDs(context.Context, []int) ([]*tt.Polygon, []error) {
	return nil, nil
}
func (UnimplementedFinder) FeedVersionGtfsImportByFeedVersionIDs(context.Context, []int) ([]*FeedVersionGtfsImport, []error) {
	return nil, nil
}
func (UnimplementedFinder) FeedVersionsByFeedIDs(context.Context, *int, *FeedVersionFilter, []int) ([][]*FeedVersion, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FeedVersionsByIDs(context.Context, []int) ([]*FeedVersion, []error) {
	return nil, nil
}
func (UnimplementedFinder) FeedVersionServiceLevelsByFeedVersionIDs(context.Context, *int, *FeedVersionServiceLevelFilter, []int) ([][]*FeedVersionServiceLevel, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FeedVersionServiceWindowByFeedVersionIDs(context.Context, []int) ([]*FeedVersionServiceWindow, []error) {
	return nil, nil
}
func (UnimplementedFinder) FlexStopTimesByStopIDs(context.Context, *int, *StopTimeFilter, []FVPair) ([][]*FlexStopTime, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FlexStopTimesByLocationIDs(context.Context, *int, *StopTimeFilter, []FVPair) ([][]*FlexStopTime, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FlexStopTimesByLocationGroupIDs(context.Context, *int, *StopTimeFilter, []FVPair) ([][]*FlexStopTime, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FlexStopTimesByTripIDs(context.Context, *int, *TripStopTimeFilter, []FVPair) ([][]*FlexStopTime, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) FrequenciesByTripIDs(context.Context, *int, []int) ([][]*Frequency, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) LevelsByIDs(context.Context, []int) ([]*Level, []error) { return nil, nil }
func (UnimplementedFinder) LevelsByParentStationIDs(context.Context, *int, []int) ([][]*Level, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) LocationGroupsByFeedVersionIDs(context.Context, *int, *LocationGroupFilter, []int) ([][]*LocationGroup, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) LocationGroupsByIDs(context.Context, []int) ([]*LocationGroup, []error) {
	return nil, nil
}
func (UnimplementedFinder) LocationGroupsByStopIDs(context.Context, *int, []int) ([][]*LocationGroup, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) LocationsByFeedVersionIDs(context.Context, *int, *LocationFilter, []int) ([][]*Location, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) LocationsByIDs(context.Context, []int) ([]*Location, []error) {
	return nil, nil
}
func (UnimplementedFinder) OperatorsByAgencyIDs(context.Context, []int) ([]*Operator, []error) {
	return nil, nil
}
func (UnimplementedFinder) OperatorsByCOIFs(context.Context, []int) ([]*Operator, []error) {
	return nil, nil
}
func (UnimplementedFinder) OperatorsByFeedIDs(context.Context, *int, *OperatorFilter, []int) ([][]*Operator, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) PathwaysByFromStopIDs(context.Context, *int, *PathwayFilter, []int) ([][]*Pathway, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) PathwaysByIDs(context.Context, []int) ([]*Pathway, []error) {
	return nil, nil
}
func (UnimplementedFinder) PathwaysByToStopIDs(context.Context, *int, *PathwayFilter, []int) ([][]*Pathway, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) RouteAttributesByRouteIDs(context.Context, []int) ([]*RouteAttribute, []error) {
	return nil, nil
}
func (UnimplementedFinder) RouteGeometriesByRouteIDs(context.Context, *int, []int) ([][]*RouteGeometry, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) RouteHeadwaysByRouteIDs(context.Context, *int, []int) ([][]*RouteHeadway, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) RoutesByAgencyIDs(context.Context, *int, *RouteFilter, []int) ([][]*Route, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) RoutesByFeedVersionIDs(context.Context, *int, *RouteFilter, []int) ([][]*Route, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) RoutesByIDs(context.Context, []int) ([]*Route, []error) { return nil, nil }
func (UnimplementedFinder) RouteStopPatternsByRouteIDs(context.Context, *int, []int) ([][]*RouteStopPattern, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) RouteStopsByRouteIDs(context.Context, *int, []int) ([][]*RouteStop, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) RouteStopsByStopIDs(context.Context, *int, []int) ([][]*RouteStop, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) SegmentPatternsByRouteIDs(context.Context, *int, *SegmentPatternFilter, []int) ([][]*SegmentPattern, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) SegmentPatternsBySegmentIDs(context.Context, *int, *SegmentPatternFilter, []int) ([][]*SegmentPattern, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) SegmentsByFeedVersionIDs(context.Context, *int, *SegmentFilter, []int) ([][]*Segment, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) SegmentsByIDs(context.Context, []int) ([]*Segment, []error) {
	return nil, nil
}
func (UnimplementedFinder) SegmentsByRouteIDs(context.Context, *int, *SegmentFilter, []int) ([][]*Segment, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) ShapesByIDs(context.Context, []int) ([]*Shape, []error) { return nil, nil }
func (UnimplementedFinder) StopExternalReferencesByStopIDs(context.Context, []int) ([]*StopExternalReference, []error) {
	return nil, nil
}
func (UnimplementedFinder) StopObservationsByStopIDs(context.Context, *int, *StopObservationFilter, []int) ([][]*StopObservation, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) StopPlacesByStopID(context.Context, []StopPlaceParam) ([]*StopPlace, []error) {
	return nil, nil
}
func (UnimplementedFinder) StopsByFeedVersionIDs(context.Context, *int, *StopFilter, []int) ([][]*Stop, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) StopsByIDs(context.Context, []int) ([]*Stop, []error) { return nil, nil }
func (UnimplementedFinder) StopsByLevelIDs(context.Context, *int, *StopFilter, []int) ([][]*Stop, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) StopsByLocationGroupIDs(context.Context, *int, []int) ([][]*Stop, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) StopsByParentStopIDs(context.Context, *int, *StopFilter, []int) ([][]*Stop, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) StopsByRouteIDs(context.Context, *int, *StopFilter, []int) ([][]*Stop, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) StopTimesByStopIDs(context.Context, *int, *StopTimeFilter, []FVPair) ([][]*StopTime, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) StopTimesByTripIDs(context.Context, *int, *TripStopTimeFilter, []FVPair) ([][]*StopTime, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) TargetStopsByStopIDs(context.Context, []int) ([]*Stop, []error) {
	return nil, nil
}
func (UnimplementedFinder) TripsByFeedVersionIDs(context.Context, *int, *TripFilter, []int) ([][]*Trip, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) TripsByIDs(context.Context, []int) ([]*Trip, []error) { return nil, nil }
func (UnimplementedFinder) TripsByRouteIDs(context.Context, *int, *TripFilter, []FVPair) ([][]*Trip, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) ValidationReportErrorExemplarsByValidationReportErrorGroupIDs(context.Context, *int, []int) ([][]*ValidationReportError, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) ValidationReportErrorGroupsByValidationReportIDs(context.Context, *int, []int) ([][]*ValidationReportErrorGroup, error) {
	return nil, ErrNotImplemented
}
func (UnimplementedFinder) ValidationReportsByFeedVersionIDs(context.Context, *int, *ValidationReportFilter, []int) ([][]*ValidationReport, error) {
	return nil, ErrNotImplemented
}

// EntityMutator

func (UnimplementedFinder) StopCreate(context.Context, StopSetInput) (int, error) {
	return 0, ErrNotImplemented
}
func (UnimplementedFinder) StopUpdate(context.Context, StopSetInput) (int, error) {
	return 0, ErrNotImplemented
}
func (UnimplementedFinder) StopDelete(context.Context, int) error { return ErrNotImplemented }
func (UnimplementedFinder) PathwayCreate(context.Context, PathwaySetInput) (int, error) {
	return 0, ErrNotImplemented
}
func (UnimplementedFinder) PathwayUpdate(context.Context, PathwaySetInput) (int, error) {
	return 0, ErrNotImplemented
}
func (UnimplementedFinder) PathwayDelete(context.Context, int) error { return ErrNotImplemented }
func (UnimplementedFinder) LevelCreate(context.Context, LevelSetInput) (int, error) {
	return 0, ErrNotImplemented
}
func (UnimplementedFinder) LevelUpdate(context.Context, LevelSetInput) (int, error) {
	return 0, ErrNotImplemented
}
func (UnimplementedFinder) LevelDelete(context.Context, int) error { return ErrNotImplemented }
