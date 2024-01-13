package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/rt"
	"github.com/interline-io/transitland-lib/rules"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/request"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/twpayne/go-geom"
	"google.golang.org/protobuf/encoding/protojson"
)

type errorWithContext interface {
	Context() *causes.Context
}

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
	result := &Result{}
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
	} else {
		// result.Result = *cpResult
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
