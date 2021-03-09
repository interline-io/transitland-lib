package validator

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/rt"
	"github.com/interline-io/transitland-lib/rules"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/twpayne/go-geom"
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
	copier.Options
}

// Validator checks a GTFS source for errors and warnings.
type Validator struct {
	Reader      tl.Reader
	Options     Options
	rtValidator *rt.Validator
}

// NewValidator returns a new Validator.
func NewValidator(reader tl.Reader, options Options) (*Validator, error) {
	// Default options
	options.IncludeServiceLevels = true
	options.IncludeEntities = true
	options.IncludeRouteGeometries = true
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
func (v *Validator) Validate() (*Result, error) {
	reader := v.Reader
	result := Result{}
	result.EarliestCalendarDate = time.Now()
	result.LatestCalendarDate = time.Now()

	// Check file infos first, so we exit early if a file exceeds the row limit.
	if reader2, ok := reader.(*tlcsv.Reader); ok {
		fvfis, err := dmfr.NewFeedVersionFileInfosFromReader(reader2)
		if err != nil {
			result.FailureReason = fmt.Sprintf("Could not read basic CSV data from file: %s", err.Error())
			return &result, nil
		}
		result.Files = fvfis
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
					return &result, nil
				}
			}
		}
	}

	// Main validation
	w := emptyWriter{}
	w.Open()
	copier := copier.NewCopier(v.Reader, &w, v.Options.Options)
	copier.AllowEntityErrors = true
	copier.AllowReferenceErrors = true
	if v.Options.BestPractices {
		copier.AddValidator(&rules.NoScheduledServiceCheck{}, 1)
		copier.AddValidator(&rules.StopTooCloseCheck{}, 1)
		copier.AddValidator(&rules.StopTooFarCheck{}, 1)
		copier.AddValidator(&rules.DuplicateRouteNameCheck{}, 1)
		copier.AddValidator(&rules.DuplicateFareRuleCheck{}, 1)
		copier.AddValidator(&rules.FrequencyOverlapCheck{}, 1)
		copier.AddValidator(&rules.StopTooFarFromShapeCheck{}, 1)
		copier.AddValidator(&rules.StopTimeFastTravelCheck{}, 1)
		copier.AddValidator(&rules.BlockOverlapCheck{}, 1)
		copier.AddValidator(&rules.InvalidTimezoneCheck{}, 1)
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
	}
	if len(v.Options.ValidateRealtimeMessages) > 0 {
		copier.AddValidator(v.rtValidator, 1)
	}
	if r := copier.Copy(); r != nil {
		result.Result = *r
	} else {
		result.FailureReason = "Failed to validate feed"
		return &result, nil
	}

	// Validate realtime messages
	for _, fn := range v.Options.ValidateRealtimeMessages {
		fmt.Println("validating rt message:", fn)
		msg, err := rt.ReadMsg(fn)
		if err != nil {
			panic(err)
		}
		rterrs := v.rtValidator.ValidateFeedMessage(msg, nil)
		result.HandleError(filepath.Base(fn), rterrs)
	}

	// Service levels
	if v.Options.IncludeServiceLevels {
		fvsls, err := dmfr.NewFeedVersionServiceInfosFromReader(reader)
		if err != nil {
			result.FailureReason = fmt.Sprintf("Could not calculate service levels: %s", err.Error())
			return &result, nil
		}
		for i, fvsl := range fvsls {
			if i > v.Options.IncludeEntitiesLimit {
				break
			}
			// TODO: deal with routes later.
			// For now only copy feed level service levels...
			if !fvsl.RouteID.Valid {
				continue
			}
			result.ServiceLevels = append(result.ServiceLevels, fvsl)
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
			result.Agencies = append(result.Agencies, ent)
			if len(result.Agencies) >= v.Options.IncludeEntitiesLimit {
				break
			}
		}
		for ent := range reader.Routes() {
			ent := ent
			if s, ok := routeShapes[ent.RouteID]; ok {
				g := tl.Geometry{Geometry: s, Valid: true}
				ent.Geometry = g
			}
			result.Routes = append(result.Routes, ent)
			if len(result.Routes) >= v.Options.IncludeEntitiesLimit {
				break
			}
		}
		for ent := range reader.Stops() {
			result.Stops = append(result.Stops, ent)
			if len(result.Stops) >= v.Options.IncludeEntitiesLimit {
				break
			}
		}
		for ent := range reader.FeedInfos() {
			result.FeedInfos = append(result.FeedInfos, ent)
			if len(result.FeedInfos) >= v.Options.IncludeEntitiesLimit {
				break
			}
		}
	}
	result.Success = true
	return &result, nil
}
