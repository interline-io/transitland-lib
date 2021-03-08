package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
)

// NoScheduledServiceError reports when a service entry contains no active days.
type NoScheduledServiceError struct {
	ServiceID string
	bc
}

func (e *NoScheduledServiceError) Error() string {
	return fmt.Sprintf(
		"service '%s' contains no active days",
		e.ServiceID,
	)
}

// NoScheduledServiceCheck checks that a service contains at least one scheduled day, otherwise returns a warning.
type NoScheduledServiceCheck struct{}

// Validate .
func (e *NoScheduledServiceCheck) Validate(ent tl.Entity) []error {
	v, ok := ent.(*tl.Service)
	if !ok {
		return nil
	}
	if v.HasAtLeastOneDay() {
		return nil
	}
	return []error{&NoScheduledServiceError{ServiceID: v.ServiceID}}
}
