package validator

import (
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/rules"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

// Options defines options for the Validator.
type Options struct {
	BestPractices bool
}

// Validator checks a GTFS source for errors and warnings.
type Validator struct {
	Reader  tl.Reader
	Copier  *copier.Copier
	Options Options
}

// NewValidator returns a new Validator.
func NewValidator(reader tl.Reader, options Options) (*Validator, error) {
	// Create empty writer
	w := emptyWriter{}
	w.Open()
	// Copy to empty writer and validate
	cp := copier.NewCopier(reader, &w, copier.Options{})
	cp.AllowEntityErrors = true
	cp.AllowReferenceErrors = true
	if options.BestPractices {
		cp.AddValidator(&rules.NoScheduledServiceCheck{}, 1)
		cp.AddValidator(&rules.StopTooCloseCheck{}, 1)
		cp.AddValidator(&rules.StopTooFarCheck{}, 1)
		cp.AddValidator(&rules.DuplicateRouteNameCheck{}, 1)
		cp.AddValidator(&rules.DuplicateFareRuleCheck{}, 1)
		cp.AddValidator(&rules.FrequencyOverlapCheck{}, 1)
		cp.AddValidator(&rules.StopTooFarFromShapeCheck{}, 1)
		cp.AddValidator(&rules.StopTimeFastTravelCheck{}, 1)
	}
	return &Validator{Reader: reader, Copier: &cp, Options: options}, nil
}

// Validate checks the feed and returns any errors and warnings that are found.
func (v *Validator) Validate() ([]error, []error) {
	result := v.Copier.Copy()
	result.DisplayErrors()
	result.DisplaySummary()
	return result.Errors, result.Warnings
}

type errorWithContext interface {
	Context() *causes.Context
}
