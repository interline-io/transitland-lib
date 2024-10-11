package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tlutil"
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

// NoScheduledServiceCheck checks for NoScheduledServiceErrors.
type NoScheduledServiceCheck struct{}

// Validate .
func (e *NoScheduledServiceCheck) Validate(ent tl.Entity) []error {
	v, ok := ent.(*tlutil.Service)
	if !ok {
		return nil
	}
	if v.HasAtLeastOneDay() {
		return nil
	}
	return []error{&NoScheduledServiceError{ServiceID: v.ServiceID}}
}
