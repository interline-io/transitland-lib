package validator

import (
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
	"github.com/interline-io/gotransit/copier"
)

// Validator checks a GTFS source for errors and warnings.
type Validator struct {
	Reader gotransit.Reader
	Copier *copier.Copier
}

// NewValidator returns a new Validator.
func NewValidator(reader gotransit.Reader) (*Validator, error) {
	// Create empty writer
	w := emptyWriter{}
	w.Open()
	// Copy to empty writer and validate
	cp := copier.NewCopier(reader, &w)
	cp.AllowEntityErrors = true
	cp.AllowReferenceErrors = true
	return &Validator{Reader: reader, Copier: &cp}, nil
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
