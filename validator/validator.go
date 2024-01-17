package validator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/dmfr/store"
	"github.com/interline-io/transitland-lib/rt"
	"github.com/interline-io/transitland-lib/rules"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/request"
	"github.com/interline-io/transitland-lib/tl/tt"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
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

// Result contains a validation report result,
type Result struct {
	Errors        map[string]*copier.ValidationErrorGroup
	Warnings      map[string]*copier.ValidationErrorGroup
	Success       bool          `json:"success"`
	FailureReason string        `json:"failure_reason"`
	Details       ResultDetails `json:"details"`
}

func NewResult() *Result {
	return &Result{
		Errors:   map[string]*copier.ValidationErrorGroup{},
		Warnings: map[string]*copier.ValidationErrorGroup{},
	}
}

type ResultDetails struct {
	SHA1                 string                         `json:"sha1"`
	EarliestCalendarDate tl.Date                        `json:"earliest_calendar_date"`
	LatestCalendarDate   tl.Date                        `json:"latest_calendar_date"`
	Timezone             string                         `json:"timezone"`
	Agencies             []tl.Agency                    `json:"agencies"`
	Routes               []tl.Route                     `json:"routes"`
	Stops                []tl.Stop                      `json:"stops"`
	FeedInfos            []tl.FeedInfo                  `json:"feed_infos"`
	Files                []dmfr.FeedVersionFileInfo     `json:"files"`
	ServiceLevels        []dmfr.FeedVersionServiceLevel `json:"service_levels"`
	Realtime             []RealtimeResult               `json:"realtime"`
}

type RealtimeResult struct {
	Url                  string                    `json:"url"`
	Json                 map[string]any            `json:"json"`
	EntityCounts         rt.EntityCounts           `json:"entity_counts"`
	TripUpdateStats      []rt.TripUpdateStats      `json:"trip_update_stats"`
	VehiclePositionStats []rt.VehiclePositionStats `json:"vehicle_position_stats"`
	Errors               []error
}

func (r *Result) Key() string {
	return fmt.Sprintf("report-%s-%d.json", r.Details.SHA1, time.Now().In(time.UTC).Unix())
}

// Options defines options for the Validator.
type Options struct {
	BestPractices            bool
	CheckFileLimits          bool
	IncludeServiceLevels     bool
	IncludeEntities          bool
	IncludeEntitiesLimit     int
	IncludeRouteGeometries   bool
	ValidateRealtimeMessages []string
	IncludeRealtimeJson      bool
	MaxRTMessageSize         uint64
	EvaluateAt               time.Time
	copier.Options
}

// Validator checks a GTFS source for errors and warnings.
type Validator struct {
	Reader      tl.Reader
	Options     Options
	rtValidator *rt.Validator
	copier      *copier.Copier
}

func (v *Validator) AddExtension(ext interface{}) error {
	return v.copier.AddExtension(ext)
}

// NewValidator returns a new Validator.
func NewValidator(reader tl.Reader, options Options) (*Validator, error) {
	// Default options
	if options.IncludeEntitiesLimit == 0 {
		options.IncludeEntitiesLimit = defaultMaxEnts
	}
	writer := &empty.Writer{}
	writer.Open()
	// Prepare copier
	options.Options.AllowEntityErrors = true
	options.Options.AllowReferenceErrors = true
	copier, err := copier.NewCopier(reader, writer, options.Options)
	if err != nil {
		return nil, err
	}
	rtv := rt.NewValidator()
	copier.AddValidator(rtv, 1)

	// Best practices extension
	if options.BestPractices {
		copier.AddValidator(&rules.NoScheduledServiceCheck{}, 1)
		copier.AddValidator(&rules.StopTooCloseCheck{}, 1)
		copier.AddValidator(&rules.StopTooFarCheck{}, 1)
		copier.AddValidator(&rules.DuplicateRouteNameCheck{}, 1)
		copier.AddValidator(&rules.DuplicateFareRuleCheck{}, 1)
		copier.AddValidator(&rules.FrequencyOverlapCheck{}, 1)
		copier.AddValidator(&rules.StopTooFarFromShapeCheck{}, 1)
		copier.AddValidator(&rules.StopTimeFastTravelCheck{}, 1)
		copier.AddValidator(&rules.BlockOverlapCheck{}, 1)
		copier.AddValidator(&rules.AgencyIDRecommendedCheck{}, 1)
		copier.AddValidator(&rules.DescriptionEqualsName{}, 1)
		copier.AddValidator(&rules.RouteExtendedTypesCheck{}, 1)
		copier.AddValidator(&rules.InsufficientColorContrastCheck{}, 1)
		copier.AddValidator(&rules.RouteShortNameTooLongCheck{}, 1)
		copier.AddValidator(&rules.ShortServiceCheck{}, 1)
		copier.AddValidator(&rules.ServiceAllDaysEmptyCheck{}, 1)
		copier.AddValidator(&rules.NullIslandCheck{}, 1)
		copier.AddValidator(&rules.FrequencyDurationCheck{}, 1)
		copier.AddValidator(&rules.MinTransferTimeCheck{}, 1)
		copier.AddValidator(&rules.RouteNamesPrefixCheck{}, 1)
		copier.AddValidator(&rules.RouteNamesCharactersCheck{}, 1)
	}
	// OK
	return &Validator{
		Reader:      reader,
		Options:     options,
		copier:      copier,
		rtValidator: rtv,
	}, nil
}

// Validate performs a basic validation, as well as optional extended reports.
func (v *Validator) Validate() (*Result, error) {
	reader := v.Reader
	result := NewResult()
	details := ResultDetails{}

	// Check file infos first, so we exit early if a file exceeds the row limit.
	if reader2, ok := reader.(*tlcsv.Reader); ok {
		fvfis, err := dmfr.NewFeedVersionFileInfosFromReader(reader2)
		if err != nil {
			result.FailureReason = fmt.Sprintf("Could not read basic CSV data from file: %s", err.Error())
			return result, nil
		}
		details.Files = fvfis
		// Maximum file limits
		if v.Options.CheckFileLimits {
			for _, fvfi := range fvfis {
				if maxRows, ok := defaultMaxFileRows[fvfi.Name]; ok && fvfi.Rows > maxRows {
					result.FailureReason = fmt.Sprintf(
						"File '%s' exceeded maximum size; got %d rows, max allowed %d rows",
						fvfi.Name,
						fvfi.Rows,
						maxRows,
					)
					return result, nil
				}
			}
		}
	}

	// get sha1 and service period; continue even if errors
	fv, err := tl.NewFeedVersionFromReader(reader)
	_ = err
	details.SHA1 = fv.SHA1
	details.EarliestCalendarDate = fv.EarliestCalendarDate
	details.LatestCalendarDate = fv.LatestCalendarDate

	// Main validation
	cpResult := v.copier.Copy()
	if cpResult == nil {
		result.FailureReason = errors.New("failed to validate feed").Error()
		return result, nil
	}

	// Service levels
	if v.Options.IncludeServiceLevels {
		fvsls, err := dmfr.NewFeedVersionServiceLevelsFromReader(reader)
		if err != nil {
			result.FailureReason = fmt.Sprintf("Could not calculate service levels: %s", err.Error())
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
			if s, ok := routeShapes[ent.RouteID]; ok {
				g := tl.Geometry{Geometry: s, Valid: true}
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

	// get service window and timezone
	evaluateAt := v.Options.EvaluateAt
	if !evaluateAt.IsZero() {
		fvsw, err := dmfr.NewFeedVersionServiceWindowFromReader(reader)
		_ = err
		details.Timezone = fvsw.DefaultTimezone.Val
		tz, _ := time.LoadLocation(details.Timezone)
		evaluateAt = time.Now().In(tz)
	}

	// Validate realtime
	for _, fn := range v.Options.ValidateRealtimeMessages {
		rtResult, err := v.ValidateRT(fn, evaluateAt)
		if err != nil {
			return result, err
		}
		cpResult.HandleError(filepath.Base(fn), rtResult.Errors)
		if len(rtResult.Errors) > v.Options.ErrorLimit {
			rtResult.Errors = rtResult.Errors[0:v.Options.ErrorLimit]
		}
		details.Realtime = append(details.Realtime, rtResult)
	}

	// Copy out errors and warnings
	for k, v := range cpResult.Errors {
		result.Errors[k] = v
	}
	for k, v := range cpResult.Warnings {
		result.Warnings[k] = v
	}

	// Return
	result.Success = true
	result.Details = details
	return result, nil
}

// Validate realtime messages
func (v *Validator) ValidateRT(fn string, evaluateAt time.Time) (RealtimeResult, error) {
	rtResult := RealtimeResult{
		Url: fn,
	}
	var rterrs []error
	msg, err := rt.ReadURL(fn, request.WithMaxSize(v.Options.MaxRTMessageSize))
	if err != nil {
		rterrs = append(rterrs, err)
	} else {
		rtResult.EntityCounts = v.rtValidator.EntityCounts(msg)
		rterrs = v.rtValidator.ValidateFeedMessage(msg, nil)
		if tripUpdateStats, err := v.rtValidator.TripUpdateStats(evaluateAt, msg); err != nil {
			rterrs = append(rterrs, err)
		} else {
			rtResult.TripUpdateStats = tripUpdateStats
		}
		if vehiclePositionStats, err := v.rtValidator.VehiclePositionStats(evaluateAt, msg); err != nil {
			rterrs = append(rterrs, err)
		} else {
			rtResult.VehiclePositionStats = vehiclePositionStats
		}
	}
	if v.Options.IncludeRealtimeJson && msg != nil {
		rtJson, err := protojson.Marshal(msg)
		if err != nil {
			log.Error().Err(err).Msg("Could not convert RT message to JSON")
		}
		if err := json.Unmarshal(rtJson, &rtResult.Json); err != nil {
			log.Error().Err(err).Msg("Could not round-trip RT message back to JSON")
		}
	}
	rtResult.Errors = rterrs
	return rtResult, nil
}

//////

type ValidationReport struct {
	Validator        tt.String
	ValidatorVersion tt.String
	Success          tt.Bool
	FailureReason    tt.String
	ReportedAt       tt.Time
	File             tt.String
	tl.BaseEntity
}

func (e *ValidationReport) TableName() string {
	return "tl_validation_reports"
}

//////

type ValidationReportErrorGroup struct {
	ValidationReportID int
	Filename           string
	Field              string
	ErrorType          string
	ErrorCode          string
	Count              int
	tl.DatabaseEntity
}

func (e *ValidationReportErrorGroup) TableName() string {
	return "tl_validation_report_error_groups"
}

//////

type ValidationReportErrorExemplar struct {
	ValidationReportErrorGroupID int
	Line                         int
	Message                      string
	EntityID                     string
	Value                        string
	Geometry                     tt.Geometry
	tl.DatabaseEntity
}

func (e *ValidationReportErrorExemplar) TableName() string {
	return "tl_validation_report_error_exemplars"
}

//////

type ValidationReportTripUpdateStat struct {
	ValidationReportID int
	AgencyID           string
	RouteID            string
	TripScheduledCount int
	TripMatchCount     int
	TripScheduledIDs   tt.Strings `db:"trip_scheduled_ids"`
	tl.DatabaseEntity
}

func (e *ValidationReportTripUpdateStat) TableName() string {
	return "tl_validation_trip_update_stats"
}

//////

type ValidationReportVehiclePositionStat struct {
	ValidationReportID int
	AgencyID           string
	RouteID            string
	TripScheduledCount int
	TripMatchCount     int
	TripScheduledIDs   tt.Strings `db:"trip_scheduled_ids"`
	tl.DatabaseEntity
}

func (e *ValidationReportVehiclePositionStat) TableName() string {
	return "tl_validation_vehicle_position_stats"
}

func SaveValidationReport(atx tldb.Adapter, result *Result, reportedAt time.Time, fvid int, reportStorage string) error {
	// Save validation reports
	validationReport := ValidationReport{}
	validationReport.FeedVersionID = fvid
	validationReport.ReportedAt = tt.NewTime(reportedAt)
	validationReport.Validator = tt.NewString("transitland-lib")
	validationReport.ValidatorVersion = tt.NewString(tl.VERSION)
	validationReport.Success = tt.NewBool(result.Success)
	validationReport.FailureReason = tt.NewString(result.FailureReason)

	// Save JSON
	if reportStorage != "" {
		validationReport.File = tt.NewString(result.Key())
		store, err := store.GetStore(reportStorage)
		if err != nil {
			return err
		}
		jj, err := json.Marshal(result)
		if err != nil {
			return err
		}
		jb := bytes.NewReader(jj)
		if err := store.Upload(context.Background(), validationReport.File.Val, tl.Secret{}, jb); err != nil {
			return err
		}
	}

	// Save record
	if _, err := atx.Insert(&validationReport); err != nil {
		log.Error().Err(err).Msg("failed to save validation report")
		return err
	}

	// Save error groups
	for _, eg := range result.Errors {
		egEnt := ValidationReportErrorGroup{
			ValidationReportID: validationReport.ID,
			Filename:           eg.Filename,
			Field:              eg.Field,
			ErrorType:          eg.ErrorType,
			ErrorCode:          eg.ErrorCode,
			Count:              eg.Count,
		}
		if _, err := atx.Insert(&egEnt); err != nil {
			log.Error().Err(err).Msg("failed to save validation report error group")
			return err
		}
		for _, egErr := range eg.Errors {
			if _, err := atx.Insert(&ValidationReportErrorExemplar{
				ValidationReportErrorGroupID: egEnt.ID,
				Line:                         egErr.Line,
				Message:                      egErr.Message,
				EntityID:                     egErr.EntityID,
				Value:                        egErr.Value,
				Geometry:                     egErr.Geometry,
			}); err != nil {
				log.Error().Err(err).Msg("failed to save validation report error exemplar")
				return err
			}
		}
	}

	// Save additional stats
	for _, r := range result.Details.Realtime {
		for _, s := range r.TripUpdateStats {
			tripReport := ValidationReportTripUpdateStat{
				ValidationReportID: validationReport.ID,
				AgencyID:           s.AgencyID,
				RouteID:            s.RouteID,
				TripScheduledCount: s.TripScheduledCount,
				TripMatchCount:     s.TripMatchCount,
				TripScheduledIDs:   tt.NewStrings(s.TripScheduledIDs),
			}
			if _, err := atx.Insert(&tripReport); err != nil {
				log.Error().Err(err).Msg("failed to save trip update stat")
				return err
			}
		}
		for _, s := range r.VehiclePositionStats {
			vpReport := ValidationReportVehiclePositionStat{
				ValidationReportID: validationReport.ID,
				AgencyID:           s.AgencyID,
				RouteID:            s.RouteID,
				TripScheduledCount: s.TripScheduledCount,
				TripMatchCount:     s.TripMatchCount,
				TripScheduledIDs:   tt.NewStrings(s.TripScheduledIDs),
			}
			if _, err := atx.Insert(&vpReport); err != nil {
				log.Error().Err(err).Msg("failed to save vehicle position stat")
				return err
			}
		}
	}
	return nil
}
