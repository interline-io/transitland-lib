package model

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"

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

// notImplBatch is the stub return for a batched ("...ByIDs") loader: a result
// and error slice sized to the keys, with ErrNotImplemented at every index. The
// dataloader adapter requires len(results)==len(keys) (see server/gql/loaders.go
// unwrapResult), so a per-key error surfaces an unimplemented loader clearly
// rather than as an opaque length-mismatch or a misleading "not found".
func notImplBatch[T any, K any](keys []K) ([]T, []error) {
	err := finderUnimpl(2)
	data := make([]T, len(keys))
	errs := make([]error, len(keys))
	for i := range errs {
		errs[i] = err
	}
	return data, errs
}

// notImplErr is the not-implemented return for a non-batched stub.
func notImplErr() error {
	return finderUnimpl(2)
}

// finderUnimpl builds the not-implemented error for the Finder method `up` frames
// above it, naming it (e.g. "not implemented: RouteGeometriesByRouteIDs") so an
// unimplemented loader identifies itself rather than surfacing a bare "not
// implemented". It wraps ErrNotImplemented so errors.Is still matches.
func finderUnimpl(up int) error {
	name := "unknown"
	if pc, _, _, ok := runtime.Caller(up); ok {
		full := runtime.FuncForPC(pc).Name()
		name = full[strings.LastIndex(full, ".")+1:]
	}
	return fmt.Errorf("%w: %s", ErrNotImplemented, name)
}

// PermFinder

func (UnimplementedFinder) PermFilter(context.Context) *PermFilter { return nil }

// EntityFinder

func (UnimplementedFinder) FindAgencies(context.Context, *int, *Cursor, []int, *AgencyFilter) ([]*Agency, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FindRoutes(context.Context, *int, *Cursor, []int, *RouteFilter) ([]*Route, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FindStops(context.Context, *int, *Cursor, []int, *StopFilter) ([]*Stop, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FindTrips(context.Context, *int, *Cursor, []int, *TripFilter) ([]*Trip, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FindFeedVersions(context.Context, *int, *Cursor, []int, *FeedVersionFilter) ([]*FeedVersion, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FindFeeds(context.Context, *int, *Cursor, []int, *FeedFilter) ([]*Feed, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FindOperators(context.Context, *int, *Cursor, []int, *OperatorFilter) ([]*Operator, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FindPlaces(context.Context, *int, *Cursor, []int, *PlaceAggregationLevel, *PlaceFilter) ([]*Place, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FindCensusDatasets(context.Context, *int, *Cursor, []int, *CensusDatasetFilter) ([]*CensusDataset, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FindCensusValuesByDatasetID(context.Context, *int, CensusCursor, int, *CensusDatasetValueFilter) ([]*CensusValue, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) RouteStopBuffer(context.Context, *int, *float64, int) ([]*RouteStopBuffer, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FindFeedVersionServiceWindow(context.Context, int) (*ServiceWindow, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) DBX() tldb.Ext { return nil }

// EntityLoader

func (UnimplementedFinder) AgenciesByFeedVersionIDs(context.Context, *int, *AgencyFilter, []int) ([][]*Agency, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) AgenciesByIDs(_ context.Context, ids []int) ([]*Agency, []error) {
	return notImplBatch[*Agency](ids)
}
func (UnimplementedFinder) AgenciesByOnestopIDs(context.Context, *int, *AgencyFilter, []string) ([][]*Agency, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) AgencyPlacesByAgencyIDs(context.Context, *int, *AgencyPlaceFilter, []int) ([][]*AgencyPlace, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) BookingRulesByFeedVersionIDs(context.Context, *int, *BookingRuleFilter, []int) ([][]*BookingRule, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) BookingRulesByIDs(_ context.Context, ids []int) ([]*BookingRule, []error) {
	return notImplBatch[*BookingRule](ids)
}
func (UnimplementedFinder) CalendarDatesByServiceIDs(context.Context, *int, *CalendarDateFilter, []int) ([][]*CalendarDate, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) CalendarsByIDs(_ context.Context, ids []int) ([]*Calendar, []error) {
	return notImplBatch[*Calendar](ids)
}
func (UnimplementedFinder) CensusDatasetLayersByDatasetIDs(_ context.Context, ids []int) ([][]*CensusLayer, []error) {
	return notImplBatch[[]*CensusLayer](ids)
}
func (UnimplementedFinder) CensusFieldsByTableIDs(context.Context, *int, []int) ([][]*CensusField, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) CensusGeographiesByDatasetIDs(context.Context, *int, *CensusDatasetGeographyFilter, []int) ([][]*CensusGeography, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) CensusGeographiesByEntityIDs(context.Context, *int, *CensusGeographyFilter, string, []int) ([][]*CensusGeography, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) CensusGeographiesByLayerIDs(context.Context, *int, *CensusSourceGeographyFilter, []int) ([][]*CensusGeography, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) CensusGeographiesBySourceIDs(context.Context, *int, *CensusSourceGeographyFilter, []int) ([][]*CensusGeography, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) CensusLayersByIDs(_ context.Context, ids []int) ([]*CensusLayer, []error) {
	return notImplBatch[*CensusLayer](ids)
}
func (UnimplementedFinder) CensusSourcesByIDs(_ context.Context, ids []int) ([]*CensusSource, []error) {
	return notImplBatch[*CensusSource](ids)
}
func (UnimplementedFinder) CensusSourceLayersBySourceIDs(_ context.Context, ids []int) ([][]*CensusLayer, []error) {
	return notImplBatch[[]*CensusLayer](ids)
}
func (UnimplementedFinder) CensusSourcesByDatasetIDs(context.Context, *int, *CensusSourceFilter, []int) ([][]*CensusSource, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) CensusTablesByDatasetIDs(context.Context, *int, *CensusTableFilter, []int) ([][]*CensusTable, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) CensusTableByIDs(_ context.Context, ids []int) ([]*CensusTable, []error) {
	return notImplBatch[*CensusTable](ids)
}
func (UnimplementedFinder) CensusValuesByGeographyIDs(context.Context, *int, string, []string, []string) ([][]*CensusValue, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FeedFetchesByFeedIDs(context.Context, *int, *FeedFetchFilter, []int) ([][]*FeedFetch, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FeedInfosByFeedVersionIDs(context.Context, *int, []int) ([][]*FeedInfo, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FeedsByIDs(_ context.Context, ids []int) ([]*Feed, []error) {
	return notImplBatch[*Feed](ids)
}
func (UnimplementedFinder) FeedsByOperatorOnestopIDs(context.Context, *int, *FeedFilter, []string) ([][]*Feed, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FeedStatesByFeedIDs(_ context.Context, ids []int) ([]*FeedState, []error) {
	return notImplBatch[*FeedState](ids)
}
func (UnimplementedFinder) FeedVersionFileInfosByFeedVersionIDs(context.Context, *int, []int) ([][]*FeedVersionFileInfo, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FeedVersionGeometryByIDs(_ context.Context, ids []int) ([]*tt.Polygon, []error) {
	return notImplBatch[*tt.Polygon](ids)
}
func (UnimplementedFinder) FeedVersionGtfsImportByFeedVersionIDs(_ context.Context, ids []int) ([]*FeedVersionGtfsImport, []error) {
	return notImplBatch[*FeedVersionGtfsImport](ids)
}
func (UnimplementedFinder) FeedVersionsByFeedIDs(context.Context, *int, *FeedVersionFilter, []int) ([][]*FeedVersion, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FeedVersionsByIDs(_ context.Context, ids []int) ([]*FeedVersion, []error) {
	return notImplBatch[*FeedVersion](ids)
}
func (UnimplementedFinder) FeedVersionServiceLevelsByFeedVersionIDs(context.Context, *int, *FeedVersionServiceLevelFilter, []int) ([][]*FeedVersionServiceLevel, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FeedVersionServiceWindowByFeedVersionIDs(_ context.Context, ids []int) ([]*FeedVersionServiceWindow, []error) {
	return notImplBatch[*FeedVersionServiceWindow](ids)
}
func (UnimplementedFinder) FlexStopTimesByStopIDs(context.Context, *int, *StopTimeFilter, []FVPair) ([][]*FlexStopTime, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FlexStopTimesByLocationIDs(context.Context, *int, *StopTimeFilter, []FVPair) ([][]*FlexStopTime, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FlexStopTimesByLocationGroupIDs(context.Context, *int, *StopTimeFilter, []FVPair) ([][]*FlexStopTime, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FlexStopTimesByTripIDs(context.Context, *int, *TripStopTimeFilter, []FVPair) ([][]*FlexStopTime, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) FrequenciesByTripIDs(context.Context, *int, []int) ([][]*Frequency, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) LevelsByIDs(_ context.Context, ids []int) ([]*Level, []error) {
	return notImplBatch[*Level](ids)
}
func (UnimplementedFinder) LevelsByParentStationIDs(context.Context, *int, []int) ([][]*Level, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) LocationGroupsByFeedVersionIDs(context.Context, *int, *LocationGroupFilter, []int) ([][]*LocationGroup, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) LocationGroupsByIDs(_ context.Context, ids []int) ([]*LocationGroup, []error) {
	return notImplBatch[*LocationGroup](ids)
}
func (UnimplementedFinder) LocationGroupsByStopIDs(context.Context, *int, []int) ([][]*LocationGroup, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) LocationsByFeedVersionIDs(context.Context, *int, *LocationFilter, []int) ([][]*Location, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) LocationsByIDs(_ context.Context, ids []int) ([]*Location, []error) {
	return notImplBatch[*Location](ids)
}
func (UnimplementedFinder) OperatorsByAgencyIDs(_ context.Context, ids []int) ([]*Operator, []error) {
	return notImplBatch[*Operator](ids)
}
func (UnimplementedFinder) OperatorsByCOIFs(_ context.Context, ids []int) ([]*Operator, []error) {
	return notImplBatch[*Operator](ids)
}
func (UnimplementedFinder) OperatorsByFeedIDs(context.Context, *int, *OperatorFilter, []int) ([][]*Operator, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) PathwaysByFromStopIDs(context.Context, *int, *PathwayFilter, []int) ([][]*Pathway, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) PathwaysByIDs(_ context.Context, ids []int) ([]*Pathway, []error) {
	return notImplBatch[*Pathway](ids)
}
func (UnimplementedFinder) PathwaysByToStopIDs(context.Context, *int, *PathwayFilter, []int) ([][]*Pathway, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) RouteAttributesByRouteIDs(_ context.Context, ids []int) ([]*RouteAttribute, []error) {
	return notImplBatch[*RouteAttribute](ids)
}
func (UnimplementedFinder) RouteGeometriesByRouteIDs(context.Context, *int, []int) ([][]*RouteGeometry, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) RouteHeadwaysByRouteIDs(context.Context, *int, []int) ([][]*RouteHeadway, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) RoutesByAgencyIDs(context.Context, *int, *RouteFilter, []int) ([][]*Route, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) RoutesByFeedVersionIDs(context.Context, *int, *RouteFilter, []int) ([][]*Route, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) RoutesByIDs(_ context.Context, ids []int) ([]*Route, []error) {
	return notImplBatch[*Route](ids)
}
func (UnimplementedFinder) RouteStopPatternsByRouteIDs(context.Context, *int, []int) ([][]*RouteStopPattern, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) RouteStopsByRouteIDs(context.Context, *int, []int) ([][]*RouteStop, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) RouteStopsByStopIDs(context.Context, *int, []int) ([][]*RouteStop, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) SegmentPatternsByRouteIDs(context.Context, *int, *SegmentPatternFilter, []int) ([][]*SegmentPattern, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) SegmentPatternsBySegmentIDs(context.Context, *int, *SegmentPatternFilter, []int) ([][]*SegmentPattern, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) SegmentsByFeedVersionIDs(context.Context, *int, *SegmentFilter, []int) ([][]*Segment, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) SegmentsByIDs(_ context.Context, ids []int) ([]*Segment, []error) {
	return notImplBatch[*Segment](ids)
}
func (UnimplementedFinder) SegmentsByRouteIDs(context.Context, *int, *SegmentFilter, []int) ([][]*Segment, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) ShapesByIDs(_ context.Context, ids []int) ([]*Shape, []error) {
	return notImplBatch[*Shape](ids)
}
func (UnimplementedFinder) StopExternalReferencesByStopIDs(_ context.Context, ids []int) ([]*StopExternalReference, []error) {
	return notImplBatch[*StopExternalReference](ids)
}
func (UnimplementedFinder) StopObservationsByStopIDs(context.Context, *int, *StopObservationFilter, []int) ([][]*StopObservation, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) StopPlacesByStopID(_ context.Context, keys []StopPlaceParam) ([]*StopPlace, []error) {
	return notImplBatch[*StopPlace](keys)
}
func (UnimplementedFinder) StopsByFeedVersionIDs(context.Context, *int, *StopFilter, []int) ([][]*Stop, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) StopsByIDs(_ context.Context, ids []int) ([]*Stop, []error) {
	return notImplBatch[*Stop](ids)
}
func (UnimplementedFinder) StopsByLevelIDs(context.Context, *int, *StopFilter, []int) ([][]*Stop, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) StopsByLocationGroupIDs(context.Context, *int, []int) ([][]*Stop, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) StopsByParentStopIDs(context.Context, *int, *StopFilter, []int) ([][]*Stop, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) StopsByRouteIDs(context.Context, *int, *StopFilter, []int) ([][]*Stop, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) StopTimesByStopIDs(context.Context, *int, *StopTimeFilter, []FVPair) ([][]*StopTime, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) StopTimesByTripIDs(context.Context, *int, *TripStopTimeFilter, []FVPair) ([][]*StopTime, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) TargetStopsByStopIDs(_ context.Context, ids []int) ([]*Stop, []error) {
	return notImplBatch[*Stop](ids)
}
func (UnimplementedFinder) TripsByFeedVersionIDs(context.Context, *int, *TripFilter, []int) ([][]*Trip, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) TripsByIDs(_ context.Context, ids []int) ([]*Trip, []error) {
	return notImplBatch[*Trip](ids)
}
func (UnimplementedFinder) TripsByRouteIDs(context.Context, *int, *TripFilter, []FVPair) ([][]*Trip, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) ValidationReportErrorExemplarsByValidationReportErrorGroupIDs(context.Context, *int, []int) ([][]*ValidationReportError, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) ValidationReportErrorGroupsByValidationReportIDs(context.Context, *int, []int) ([][]*ValidationReportErrorGroup, error) {
	return nil, notImplErr()
}
func (UnimplementedFinder) ValidationReportsByFeedVersionIDs(context.Context, *int, *ValidationReportFilter, []int) ([][]*ValidationReport, error) {
	return nil, notImplErr()
}

// EntityMutator

func (UnimplementedFinder) StopCreate(context.Context, StopSetInput) (int, error) {
	return 0, notImplErr()
}
func (UnimplementedFinder) StopUpdate(context.Context, StopSetInput) (int, error) {
	return 0, notImplErr()
}
func (UnimplementedFinder) StopDelete(context.Context, int) error { return notImplErr() }
func (UnimplementedFinder) PathwayCreate(context.Context, PathwaySetInput) (int, error) {
	return 0, notImplErr()
}
func (UnimplementedFinder) PathwayUpdate(context.Context, PathwaySetInput) (int, error) {
	return 0, notImplErr()
}
func (UnimplementedFinder) PathwayDelete(context.Context, int) error { return ErrNotImplemented }
func (UnimplementedFinder) LevelCreate(context.Context, LevelSetInput) (int, error) {
	return 0, notImplErr()
}
func (UnimplementedFinder) LevelUpdate(context.Context, LevelSetInput) (int, error) {
	return 0, notImplErr()
}
func (UnimplementedFinder) LevelDelete(context.Context, int) error { return ErrNotImplemented }
