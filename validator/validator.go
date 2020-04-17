package validator

import (
	"fmt"

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
	cp.EntityErrorHandler = func(ent gotransit.Entity, errs []error) {
		if len(errs) == 0 {
			return
		}
		fmt.Printf("entity error handler: %#v --- %s\n", ent, errs)
	}
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
