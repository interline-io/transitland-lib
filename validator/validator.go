package validator

import (
	"fmt"
	"time"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/dmfr"
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
	BestPractices          bool
	CheckFileLimits        bool
	IncludeServiceLevels   bool
	IncludeEntities        bool
	IncludeEntitiesLimit   int
	IncludeRouteGeometries bool
	copier.Options
}

// Validator checks a GTFS source for errors and warnings.
type Validator struct {
	Reader  tl.Reader
	Options Options
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
		Reader:  reader,
		Options: options,
	}, nil
}

// Validate performs a basic validation, as well as optional extended reports.
func (v *Validator) Validate() (*Result, error) {
	t := time.Now()
	reader := v.Reader
	result := Result{}
	result.EarliestCalendarDate = time.Now()
	result.LatestCalendarDate = time.Now()

	// Check file infos first, so we exit early if a file exceeds the row limit.
	if reader2, ok := reader.(*tlcsv.Reader); ok {
		fmt.Println("file infos")
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
		fmt.Println("done:", float64(time.Now().UnixNano()-t.UnixNano())/1e9)
		t = time.Now()
	}

	// Main validation
	t = time.Now()
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
	}
	if r := copier.Copy(); r != nil {
		result.Result = *r
	} else {
		result.FailureReason = "Failed to validate feed"
		return &result, nil
	}
	fmt.Println("done:", float64(time.Now().UnixNano()-t.UnixNano())/1e9)
	t = time.Now()

	// Service levels
	if v.Options.IncludeServiceLevels {
		fmt.Println("service levels")
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
		fmt.Println("done:", float64(time.Now().UnixNano()-t.UnixNano())/1e9)
		t = time.Now()
	}

	routeShapes := map[string]*geom.MultiLineString{}
	if v.Options.IncludeRouteGeometries {
		// Build shapes...
		fmt.Println("building shapes")
		routeShapes = buildRouteShapes(reader)
		fmt.Println("done:", float64(time.Now().UnixNano()-t.UnixNano())/1e9)
		t = time.Now()
	}

	// Include some basic entities in the report
	if v.Options.IncludeEntities {
		// Add basic entities
		fmt.Println("adding basic entities")
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
		fmt.Println("done:", float64(time.Now().UnixNano()-t.UnixNano())/1e9)
	}
	result.Success = true
	return &result, nil
}
