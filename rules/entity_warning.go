package rules

import "github.com/interline-io/transitland-lib/tl"

// EntityWarningCheck runs the entity's built in Warnings() check.
type EntityWarningCheck struct{}

// Validate .
func (e *EntityWarningCheck) Validate(ent tl.Entity) []error {
	return ent.Warnings()
}
