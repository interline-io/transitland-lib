package rules

import "github.com/interline-io/transitland-lib/tl"

// EntityErrorCheck runs the entity's built in Errors() check.
type EntityErrorCheck struct{}

// Validate .
func (e *EntityErrorCheck) Validate(ent tl.Entity) []error {
	return ent.Errors()
}
