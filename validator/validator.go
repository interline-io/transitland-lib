package validator

import (
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/rules"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
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
	result := Result{}

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
	}
	if r := copier.Copy(); r != nil {
		result.Result = *r
	} else {
		result.FailureReason = "Failed to validate feed"
		return &result, nil
	}

	result.Success = true
	return &result, nil
}
