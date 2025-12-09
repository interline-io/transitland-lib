package gql

import (
	"context"
	"errors"

	"github.com/99designs/gqlgen/graphql"
	"github.com/interline-io/transitland-lib/internal/generated/gqlout"
	"github.com/interline-io/transitland-lib/server/meters"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlxy"
)

// RESOLVER_DEFAULT_LIMIT is the default API limit
const (
	RESOLVER_DEFAULT_LIMIT             = 100
	RESOLVER_FEED_MAXLIMIT             = 100_000
	RESOLVER_CENSUS_MAXLIMIT           = 100_000
	RESOLVER_STOP_OBSERVATION_MAXLIMIT = 100_000
	RESOLVER_LOCATION_MAXLIMIT         = 100_000
)

// RESOLVER_MAXLIMIT is the API limit maximum
var RESOLVER_MAXLIMIT = 1_000

// resolverCheckLimit checks the limit is positive and below the maximum limit.
func resolverCheckLimit(limit *int) *int {
	return resolverCheckLimitMax(limit, RESOLVER_MAXLIMIT)
}

// checkLimit checks the limit is positive and below the maximum limit.
func resolverCheckLimitMax(limit *int, maxLimit int) *int {
	a := RESOLVER_DEFAULT_LIMIT
	if limit == nil {
		return &a
	} else {
		a = *limit
	}
	if a < 0 {
		a = 0
	} else if a >= maxLimit {
		a = maxLimit
	}
	return &a
}

func checkCursor(after *int) *model.Cursor {
	var cursor *model.Cursor
	if after != nil {
		c := model.NewCursor(0, *after)
		cursor = &c
	}
	return cursor
}

func addMetric(ctx context.Context, resolverName string) context.Context {
	if apiMeter := meters.ForContext(ctx); apiMeter != nil {
		apiMeter.ApplyDimension("resolver", resolverName)
	}
	return ctx
}

func checkGeo(maxRadius float64, near *model.PointRadius, bbox *model.BoundingBox) error {
	// No max radius set
	if maxRadius == 0 {
		return nil
	}
	// Check if radius is too large
	if near != nil && near.Radius > maxRadius {
		return errors.New("radius too large")
	}
	// Check if bbox is too large
	if bbox != nil && !checkBbox(bbox, maxRadius*maxRadius) {
		return errors.New("bbox too large")
	}
	return nil
}

func checkBbox(bbox *model.BoundingBox, maxAreaM2 float64) bool {
	approxDiag := tlxy.DistanceHaversine(tlxy.Point{Lon: bbox.MinLon, Lat: bbox.MinLat}, tlxy.Point{Lon: bbox.MaxLon, Lat: bbox.MaxLat})
	approxArea := 0.5 * (approxDiag * approxDiag)
	return approxArea < maxAreaM2
}

func ptr[T any, PT *T](v T) PT {
	a := v
	return &a
}

func checkFloat(v *float64, min float64, max float64) float64 {
	if v == nil || *v < min {
		return min
	} else if *v > max {
		return max
	}
	return *v
}

func containsField(ctx context.Context, fieldName string) bool {
	fields := graphql.CollectFieldsCtx(ctx, nil)
	for _, field := range fields {
		if field.Name == fieldName {
			return true
		}
	}
	return false
}

func convertScheduleRelationship(sr string) *model.ScheduleRelationship {
	var msr model.ScheduleRelationship
	switch sr {
	case "STATIC":
		msr = model.ScheduleRelationshipStatic
	case "SCHEDULED":
		msr = model.ScheduleRelationshipScheduled
	case "ADDED":
		msr = model.ScheduleRelationshipAdded
	case "CANCELED":
		msr = model.ScheduleRelationshipCanceled
	case "UNSCHEDULED":
		msr = model.ScheduleRelationshipUnscheduled
	case "REPLACEMENT":
		msr = model.ScheduleRelationshipReplacement
	case "DUPLICATED":
		msr = model.ScheduleRelationshipDuplicated
	case "DELETED":
		msr = model.ScheduleRelationshipDeleted
	case "SKIPPED":
		msr = model.ScheduleRelationshipSkipped
	case "NO_DATA":
		msr = model.ScheduleRelationshipNoData
	default:
		return nil
	}
	return &msr
}

// Resolver .
type Resolver struct{}

// Query .
func (r *Resolver) Query() gqlout.QueryResolver { return &queryResolver{r} }

// Mutation .
func (r *Resolver) Mutation() gqlout.MutationResolver { return &mutationResolver{r} }

// Agency .
func (r *Resolver) Agency() gqlout.AgencyResolver { return &agencyResolver{r} }

// Feed .
func (r *Resolver) Feed() gqlout.FeedResolver { return &feedResolver{r} }

// FeedState .
func (r *Resolver) FeedState() gqlout.FeedStateResolver { return &feedStateResolver{r} }

// FeedVersion .
func (r *Resolver) FeedVersion() gqlout.FeedVersionResolver { return &feedVersionResolver{r} }

// Route .
func (r *Resolver) Route() gqlout.RouteResolver { return &routeResolver{r} }

// RouteStop .
func (r *Resolver) RouteStop() gqlout.RouteStopResolver { return &routeStopResolver{r} }

// RouteHeadway .
func (r *Resolver) RouteHeadway() gqlout.RouteHeadwayResolver { return &routeHeadwayResolver{r} }

// RouteStopPattern .
func (r *Resolver) RouteStopPattern() gqlout.RouteStopPatternResolver {
	return &routePatternResolver{r}
}

// Segment .
func (r *Resolver) Segment() gqlout.SegmentResolver { return &segmentResolver{r} }

// SegmentPattern .
func (r *Resolver) SegmentPattern() gqlout.SegmentPatternResolver { return &segmentPatternResolver{r} }

// Stop .
func (r *Resolver) Stop() gqlout.StopResolver { return &stopResolver{r} }

// Trip .
func (r *Resolver) Trip() gqlout.TripResolver { return &tripResolver{r} }

// StopTime .
func (r *Resolver) StopTime() gqlout.StopTimeResolver { return &stopTimeResolver{r} }

// FlexStopTime .
func (r *Resolver) FlexStopTime() gqlout.FlexStopTimeResolver { return &flexStopTimeResolver{r} }

// Location .
func (r *Resolver) Location() gqlout.LocationResolver { return &locationResolver{r} }

// BookingRule .
func (r *Resolver) BookingRule() gqlout.BookingRuleResolver { return &bookingRuleResolver{r} }

// LocationGroup .
func (r *Resolver) LocationGroup() gqlout.LocationGroupResolver { return &locationGroupResolver{r} }

// LocationGroupStop .
func (r *Resolver) LocationGroupStop() gqlout.LocationGroupStopResolver {
	return &locationGroupStopResolver{r}
}

// Operator .
func (r *Resolver) Operator() gqlout.OperatorResolver { return &operatorResolver{r} }

// FeedVersionGtfsImport .
func (r *Resolver) FeedVersionGtfsImport() gqlout.FeedVersionGtfsImportResolver {
	return &feedVersionGtfsImportResolver{r}
}

func (r *Resolver) Level() gqlout.LevelResolver {
	return &levelResolver{r}
}

// Calendar .
func (r *Resolver) Calendar() gqlout.CalendarResolver {
	return &calendarResolver{r}
}

// CensusGeography .
func (r *Resolver) CensusGeography() gqlout.CensusGeographyResolver {
	return &censusGeographyResolver{r}
}

func (r *Resolver) CensusValue() gqlout.CensusValueResolver {
	return &censusValueResolver{r}
}

func (r *Resolver) CensusTable() gqlout.CensusTableResolver {
	return &censusTableResolver{r}
}

// Pathway .
func (r *Resolver) Pathway() gqlout.PathwayResolver {
	return &pathwayResolver{r}
}

// StopExternalReference .
func (r *Resolver) StopExternalReference() gqlout.StopExternalReferenceResolver {
	return &stopExternalReferenceResolver{r}
}

// Directions .
func (r *Resolver) Directions(ctx context.Context, where model.DirectionRequest) (*model.Directions, error) {
	dr := directionsResolver{r}
	return dr.Directions(ctx, where)
}

func (r *Resolver) Place() gqlout.PlaceResolver {
	return &placeResolver{r}
}

func (r *Resolver) ValidationReport() gqlout.ValidationReportResolver {
	return &validationReportResolver{r}
}

func (r *Resolver) ValidationReportErrorGroup() gqlout.ValidationReportErrorGroupResolver {
	return &validationReportErrorGroupResolver{r}
}

func (r *Resolver) CensusDataset() gqlout.CensusDatasetResolver {
	return &censusDatasetResolver{r}
}

func (r *Resolver) CensusSource() gqlout.CensusSourceResolver {
	return &censusSourceResolver{r}
}

func (r *Resolver) CensusLayer() gqlout.CensusLayerResolver {
	return &censusLayerResolver{r}
}
