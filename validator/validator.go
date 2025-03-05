// Package validator provides GTFS and GTFS-RT validation utilities.
package validator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-lib/rt"
	"github.com/interline-io/transitland-lib/rules"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/twpayne/go-geom"
	"google.golang.org/protobuf/encoding/protojson"
)

var defaultMaxEnts = 10000

var defaultMaxFileRows = map[string]int64{
	"agency.txt":     1000,
	"routes.txt":     1000,
	"stops.txt":      100_000,
	"trips.txt":      1_000_000,
	"stop_times.txt": 10_000_000,
	"shapes.txt":     10_000_000,
}

// Options defines options for the Validator.
type Options struct {
	BestPractices            bool
	CheckFileLimits          bool
	IncludeServiceLevels     bool
	IncludeEntities          bool
	IncludeEntitiesLimit     int
	IncludeRouteGeometries   bool
	UseHeaderTimestamp       bool
	ValidateRealtimeMessages []string
	IncludeRealtimeJson      bool
	MaxRTMessageSize         uint64
	EvaluateAt               time.Time
	EvaluateAtTimezone       string
	copier.Options
}

// Validator checks a GTFS source for errors and warnings.
type Validator struct {
	Reader          adapters.Reader
	Options         Options
	rtValidator     *rt.Validator
	defaultTimezone string
}

// NewValidator returns a new Validator.
func NewValidator(reader adapters.Reader, options Options) (*Validator, error) {
	// Default options
	if options.IncludeEntitiesLimit == 0 {
		options.IncludeEntitiesLimit = defaultMaxEnts
	}
	return &Validator{
		Reader:      reader,
		Options:     options,
		rtValidator: rt.NewValidator(),
	}, nil
}

// Validate performs a basic validation, as well as optional extended reports.
func (v *Validator) Validate(ctx context.Context) (*Result, error) {
	tzName, err := v.setDefaultTimezone(v.Options.EvaluateAtTimezone)
	if err != nil {
		return nil, err
	}
	evaluateAt, evaluateAtLocal, err := v.getTimes(v.Options.EvaluateAt, tzName)
	if err != nil {
		return nil, err
	}
	result := NewResult(evaluateAt, evaluateAtLocal)

	// Validate static
	if v.Reader != nil {
		stResult, err := v.ValidateStatic(v.Reader, evaluateAt, evaluateAtLocal)
		if err != nil {
			result.FailureReason.Set(err.Error())
			return result, err
		}
		result.Details = stResult.Details
		result.IncludesStatic = stResult.IncludesStatic
		for k, v := range stResult.Errors {
			result.Errors[k] = v
		}
		for k, v := range stResult.Warnings {
			result.Warnings[k] = v
		}
	}

	// Validate realtime
	if len(v.Options.ValidateRealtimeMessages) > 0 {
		rtResult, err := v.ValidateRTs(ctx, v.Options.ValidateRealtimeMessages, evaluateAt, evaluateAtLocal)
		if err != nil {
			result.FailureReason.Set(err.Error())
			return result, err
		}
		// Copy
		result.IncludesRT = rtResult.IncludesRT
		result.Details.Realtime = append(result.Details.Realtime, rtResult.Details.Realtime...)
		for k, v := range rtResult.Errors {
			result.Errors[k] = v
		}
		for k, v := range rtResult.Warnings {
			result.Warnings[k] = v
		}
	}

	// Return
	result.Success.Set(true)
	return result, nil
}

func (v *Validator) ValidateStatic(reader adapters.Reader, evaluateAt time.Time, evaluateAtLocal time.Time) (*Result, error) {
	result := NewResult(evaluateAt, evaluateAtLocal)

	v.rtValidator = rt.NewValidator()
	copier, err := v.setupCopier(reader)
	if err != nil {
		return nil, err
	}

	details := ResultDetails{}
	if reader2, ok := reader.(*tlcsv.Reader); ok {
		result.IncludesStatic.Set(true)
		fvfis, err := stats.NewFeedVersionFileInfosFromReader(reader2)
		if err != nil {
			result.FailureReason.Set(fmt.Sprintf("Could not read basic CSV data from file: %s", err.Error()))
			return result, nil
		}
		details.Files = fvfis
		// Maximum file limits
		if v.Options.CheckFileLimits {
			for _, fvfi := range fvfis {
				if maxRows, ok := defaultMaxFileRows[fvfi.Name]; ok && fvfi.Rows > maxRows {
					result.FailureReason.Set(fmt.Sprintf(
						"File '%s' exceeded maximum size; got %d rows, max allowed %d rows",
						fvfi.Name,
						fvfi.Rows,
						maxRows,
					))
					return result, nil
				}
			}
		}
	}

	// get sha1 and service period; continue even if errors
	fv, err := stats.NewFeedVersionFromReader(reader)
	_ = err
	details.SHA1.Set(fv.SHA1)
	details.EarliestCalendarDate = fv.EarliestCalendarDate
	details.LatestCalendarDate = fv.LatestCalendarDate

	// Main validation
	cpResult := copier.Copy()
	if cpResult == nil {
		result.FailureReason.Set("failed to validate feed")
		return result, nil
	}

	// Service levels
	if v.Options.IncludeServiceLevels {
		fvsls, err := stats.NewFeedVersionServiceLevelsFromReader(reader)
		if err != nil {
			result.FailureReason.Set(fmt.Sprintf("Could not calculate service levels: %s", err.Error()))
			return result, nil
		}
		for i, fvsl := range fvsls {
			if i > v.Options.IncludeEntitiesLimit {
				break
			}
			details.ServiceLevels = append(details.ServiceLevels, fvsl)
		}
	}

	routeShapes := map[string]*geom.MultiLineString{}
	if v.Options.IncludeRouteGeometries {
		// Build shapes...
		routeShapes = buildRouteShapes(reader)
	}

	// Include some basic entities in the report
	if v.Options.IncludeEntities {
		// Add basic entities
		for ent := range reader.Agencies() {
			details.Agencies = append(details.Agencies, ent)
			if len(details.Agencies) >= v.Options.IncludeEntitiesLimit {
				break
			}
		}
		for ent := range reader.Routes() {
			ent := ent
			if s, ok := routeShapes[ent.RouteID.Val]; ok {
				g := tt.NewGeometry(s)
				ent.Geometry = g
			}
			details.Routes = append(details.Routes, ent)
			if len(details.Routes) >= v.Options.IncludeEntitiesLimit {
				break
			}
		}
		for ent := range reader.Stops() {
			details.Stops = append(details.Stops, ent)
			if len(details.Stops) >= v.Options.IncludeEntitiesLimit {
				break
			}
		}
		for ent := range reader.FeedInfos() {
			details.FeedInfos = append(details.FeedInfos, ent)
			if len(details.FeedInfos) >= v.Options.IncludeEntitiesLimit {
				break
			}
		}
	}

	// Copy out errors and warnings
	for k, v := range cpResult.Errors {
		result.Errors[k] = copierEgToValidationEg(v)
	}
	for k, v := range cpResult.Warnings {
		result.Warnings[k] = copierEgToValidationEg(v)
	}

	// Return
	result.Success.Set(true)
	result.Details = details
	return result, nil

}

func (v *Validator) ValidateRTs(ctx context.Context, rtUrls []string, evaluateAt time.Time, evaluateAtLocal time.Time) (*Result, error) {
	// Validate realtime
	result := NewResult(evaluateAt, evaluateAtLocal)
	result.IncludesRT.Set(true)
	for _, fn := range rtUrls {
		// Validate RT message
		rtResult, err := v.ValidateRT(ctx, fn, evaluateAt, evaluateAtLocal)
		if err != nil {
			return result, err
		}
		// Create a temp copier result to handle errors
		cpResult := copier.NewResult(v.Options.ErrorLimit)
		cpResult.HandleError(filepath.Base(fn), rtResult.Errors)
		if len(rtResult.Errors) > v.Options.ErrorLimit {
			rtResult.Errors = rtResult.Errors[0:v.Options.ErrorLimit]
		}
		// Copy out results
		result.Details.Realtime = append(result.Details.Realtime, rtResult)
		for k, eg := range cpResult.Errors {
			result.Errors[k] = copierEgToValidationEg(eg)
		}
		for k, eg := range cpResult.Warnings {
			result.Warnings[k] = copierEgToValidationEg(eg)
		}
	}
	return result, nil
}

// Validate realtime messages
func (v *Validator) ValidateRT(ctx context.Context, fn string, evaluateAt time.Time, evaluateAtLocal time.Time) (RealtimeResult, error) {
	log.For(ctx).Info().Str("url", fn).Msg("Validating GTFS-RT")
	rtResult := RealtimeResult{
		Url: fn,
	}
	var rterrs []error
	msg, err := rt.ReadURL(ctx, fn, request.WithMaxSize(v.Options.MaxRTMessageSize), request.WithAllowLocal)
	if err != nil {
		rterrs = append(rterrs, err)
	} else {
		if v.Options.UseHeaderTimestamp {
			evaluateAt = time.Unix(int64(msg.GetHeader().GetTimestamp()), 0)
			evaluateAtLocal = evaluateAt.In(evaluateAtLocal.Location())
			log.For(ctx).Debug().Str("evaluateAt", evaluateAt.String()).Str("evaluateAtLocal", evaluateAtLocal.String()).Msg("Using header timestamps for evaluation time")
		} else {
			log.For(ctx).Debug().Str("evaluateAt", evaluateAt.String()).Str("evaluateAtLocal", evaluateAtLocal.String()).Msg("Using provided timestamp for evaluation time")
		}
		rtResult.EntityCounts = v.rtValidator.EntityCounts(msg)
		rterrs = v.rtValidator.ValidateFeedMessage(msg, nil)
		if tripUpdateStats, err := v.rtValidator.TripUpdateStats(evaluateAtLocal, msg); err != nil {
			rterrs = append(rterrs, err)
		} else {
			rtResult.TripUpdateStats = tripUpdateStats
		}
		if vehiclePositionStats, err := v.rtValidator.VehiclePositionStats(evaluateAtLocal, msg); err != nil {
			rterrs = append(rterrs, err)
		} else {
			rtResult.VehiclePositionStats = vehiclePositionStats
		}
	}

	if v.Options.IncludeRealtimeJson && msg != nil {
		mOpts := protojson.MarshalOptions{UseProtoNames: true}
		rtJson, err := mOpts.Marshal(msg)
		if err != nil {
			log.For(ctx).Error().Err(err).Msg("Could not convert RT message to JSON")
		}
		if err := json.Unmarshal(rtJson, &rtResult.Json); err != nil {
			log.For(ctx).Error().Err(err).Msg("Could not round-trip RT message back to JSON")
		}
	}
	rtResult.Errors = rterrs
	return rtResult, nil
}

func (v *Validator) setDefaultTimezone(tzName string) (string, error) {
	// Get default timezone
	if tzName == "" {
		tzName = v.defaultTimezone
	}
	if v.Reader != nil && tzName == "" {
		// Get service window and timezone
		fvsw, err := stats.NewFeedVersionServiceWindowFromReader(v.Reader)
		if err != nil {
			return "", err
		}
		tzName = fvsw.DefaultTimezone.Val
		v.defaultTimezone = tzName
	}
	return tzName, nil
}

func (v *Validator) getTimes(now time.Time, tzName string) (time.Time, time.Time, error) {
	if tzName == "" {
		tzName = v.defaultTimezone
	}
	// We should always have a valid timezone at this point
	if now.IsZero() {
		now = time.Now().In(time.UTC)
	}
	tz, err := time.LoadLocation(tzName)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	nowLocal := now.In(tz)
	return now, nowLocal, nil
}

func (v *Validator) setupCopier(reader adapters.Reader) (*copier.Copier, error) {
	writer := &empty.Writer{}
	writer.Open()

	// Prepare copier
	cpOpts := v.Options.Options
	cpOpts.AllowEntityErrors = true
	cpOpts.AllowReferenceErrors = true
	cpOpts.AddExtensionWithLevel(v.rtValidator, 1)

	// Best practices extension
	if v.Options.BestPractices {
		cpOpts.AddExtensionWithLevel(&rules.NoScheduledServiceCheck{}, 1)
		cpOpts.AddExtensionWithLevel(&rules.StopTooCloseCheck{}, 1)
		cpOpts.AddExtensionWithLevel(&rules.StopTooFarCheck{}, 1)
		cpOpts.AddExtensionWithLevel(&rules.DuplicateRouteNameCheck{}, 1)
		cpOpts.AddExtensionWithLevel(&rules.FrequencyOverlapCheck{}, 1)
		cpOpts.AddExtensionWithLevel(&rules.StopTooFarFromShapeCheck{}, 1)
		cpOpts.AddExtensionWithLevel(&rules.StopTimeFastTravelCheck{}, 1)
		cpOpts.AddExtensionWithLevel(&rules.BlockOverlapCheck{}, 1)
		cpOpts.AddExtensionWithLevel(&rules.AgencyIDRecommendedCheck{}, 1)
		cpOpts.AddExtensionWithLevel(&rules.DescriptionEqualsName{}, 1)
		cpOpts.AddExtensionWithLevel(&rules.RouteExtendedTypesCheck{}, 1)
		cpOpts.AddExtensionWithLevel(&rules.InsufficientColorContrastCheck{}, 1)
		cpOpts.AddExtensionWithLevel(&rules.RouteShortNameTooLongCheck{}, 1)
		cpOpts.AddExtensionWithLevel(&rules.ShortServiceCheck{}, 1)
		cpOpts.AddExtensionWithLevel(&rules.ServiceAllDaysEmptyCheck{}, 1)
		cpOpts.AddExtensionWithLevel(&rules.NullIslandCheck{}, 1)
		cpOpts.AddExtensionWithLevel(&rules.FrequencyDurationCheck{}, 1)
		cpOpts.AddExtensionWithLevel(&rules.MinTransferTimeCheck{}, 1)
		cpOpts.AddExtensionWithLevel(&rules.RouteNamesPrefixCheck{}, 1)
		cpOpts.AddExtensionWithLevel(&rules.RouteNamesCharactersCheck{}, 1)
	}

	copier, err := copier.NewCopier(reader, writer, cpOpts)
	if err != nil {
		return nil, err
	}
	return copier, nil
}

func copierEgToValidationEg(eg *copier.ValidationErrorGroup) *ValidationReportErrorGroup {
	ret := ValidationReportErrorGroup{
		Filename:  eg.Filename,
		Field:     eg.Field,
		ErrorType: eg.ErrorType,
		ErrorCode: eg.ErrorCode,
		GroupKey:  eg.GroupKey,
		Count:     eg.Count,
		Level:     eg.Level,
	}
	for _, egErr := range eg.Errors {
		ret.Errors = append(ret.Errors, ValidationReportErrorExemplar{
			Line:       egErr.Line,
			Message:    egErr.Message,
			EntityID:   egErr.EntityID,
			Value:      egErr.Value,
			Geometry:   egErr.Geometry,
			EntityJson: egErr.EntityJson,
		})
	}
	return &ret
}

func SaveValidationReport(ctx context.Context, atx tldb.Adapter, result *Result, fvid int, reportStorage string) error {
	// Save validation reports
	result.FeedVersionID = fvid

	// Save JSON
	if reportStorage != "" {
		result.File.Set(result.Key())
		store, err := request.GetStore(reportStorage)
		if err != nil {
			return err
		}
		jj, err := json.Marshal(result)
		if err != nil {
			return err
		}
		jb := bytes.NewReader(jj)
		if err := store.Upload(context.Background(), result.File.Val, jb); err != nil {
			return err
		}
	}

	// Save record
	if _, err := atx.Insert(ctx, result); err != nil {
		log.For(ctx).Error().Err(err).Msg("failed to save validation report")
		return err
	}

	// Save error groups
	var combinedErrors []*ValidationReportErrorGroup
	for _, eg := range result.Errors {
		combinedErrors = append(combinedErrors, eg)
	}
	for _, eg := range result.Warnings {
		combinedErrors = append(combinedErrors, eg)
	}
	for _, eg := range combinedErrors {
		eg.ValidationReportID = result.ID
		if _, err := atx.Insert(ctx, eg); err != nil {
			log.For(ctx).Error().Err(err).Msg("failed to save validation report error group")
			return err
		}
		for _, egErr := range eg.Errors {
			egErr.ValidationReportErrorGroupID = eg.ID
			if _, err := atx.Insert(ctx, &egErr); err != nil {
				log.For(ctx).Error().Err(err).Msg("failed to save validation report error exemplar")
				return err
			}
		}
	}

	// Save additional stats
	for _, r := range result.Details.Realtime {
		for _, s := range r.TripUpdateStats {
			tripReport := ValidationReportTripUpdateStat{
				ValidationReportID:      result.ID,
				AgencyID:                s.AgencyID,
				RouteID:                 s.RouteID,
				TripScheduledIDs:        tt.NewStrings(s.TripScheduledIDs),
				TripScheduledCount:      s.TripScheduledCount,
				TripScheduledMatched:    s.TripScheduledMatched,
				TripScheduledNotMatched: s.TripScheduledNotMatched,
				TripRtIDs:               tt.NewStrings(s.TripRtIDs),
				TripRtCount:             s.TripRtCount,
				TripRtMatched:           s.TripRtMatched,
				TripRtNotMatched:        s.TripRtNotMatched,
				TripRtNotFoundIDs:       tt.NewStrings(s.TripRtNotFoundIDs),
				TripRtAddedIDs:          tt.NewStrings(s.TripRtAddedIDs),
				TripRtNotFoundCount:     s.TripRtNotFoundCount,
				TripRtAddedCount:        s.TripRtAddedCount,
			}
			if _, err := atx.Insert(ctx, &tripReport); err != nil {
				log.For(ctx).Error().Err(err).Msg("failed to save trip update stat")
				return err
			}
		}
		for _, s := range r.VehiclePositionStats {
			vpReport := ValidationReportVehiclePositionStat{
				ValidationReportID:      result.ID,
				AgencyID:                s.AgencyID,
				RouteID:                 s.RouteID,
				TripScheduledIDs:        tt.NewStrings(s.TripScheduledIDs),
				TripScheduledCount:      s.TripScheduledCount,
				TripScheduledMatched:    s.TripScheduledMatched,
				TripScheduledNotMatched: s.TripScheduledNotMatched,
				TripRtIDs:               tt.NewStrings(s.TripRtIDs),
				TripRtCount:             s.TripRtCount,
				TripRtMatched:           s.TripRtMatched,
				TripRtNotMatched:        s.TripRtNotMatched,
				TripRtNotFoundIDs:       tt.NewStrings(s.TripRtNotFoundIDs),
				TripRtAddedIDs:          tt.NewStrings(s.TripRtAddedIDs),
				TripRtNotFoundCount:     s.TripRtNotFoundCount,
				TripRtAddedCount:        s.TripRtAddedCount,
			}
			if _, err := atx.Insert(ctx, &vpReport); err != nil {
				log.For(ctx).Error().Err(err).Msg("failed to save vehicle position stat")
				return err
			}
		}
	}
	return nil
}
