package validator

import (
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
	"github.com/interline-io/gotransit/copier"
	"github.com/interline-io/gotransit/internal/log"
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
	cp.NormalizeServiceIDs = true
	return &Validator{Reader: reader, Copier: &cp}, nil
}

// Validate checks the feed and returns any errors and warnings that are found.
func (v *Validator) Validate() ([]error, []error) {
	result := v.Copier.Copy()
	displayCopyResult(result)
	return result.Errors, result.Warnings
}

type errorWithContext interface {
	Context() *causes.Context
}

func displayCopyResult(result *copier.CopyResult) {
	keys := map[string][]error{}
	for _, err := range result.Errors {
		efn := ""
		if v, ok := err.(errorWithContext); ok {
			ctx := v.Context()
			efn = ctx.Filename
		}
		keys[efn] = append(keys[efn], err)
	}
	for k, v := range keys {
		log.Info("filename: %s", k)
		group := map[string][]error{}
		for _, err := range v {
			eid := ""
			if v, ok := err.(errorWithContext); ok {
				ctx := v.Context()
				eid = ctx.EntityID
			}
			group[eid] = append(group[eid], err)
		}
		for k, v := range group {
			log.Info("\t%s:", k)
			for _, err := range v {
				log.Info("\t\t%s", err)
			}
		}
	}
}
