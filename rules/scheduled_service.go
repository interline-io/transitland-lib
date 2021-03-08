package rules

import "github.com/interline-io/transitland-lib/tl"

// NoScheduledServiceError reports when a service entry contains no active days.
type NoScheduledServiceError struct{ bc }

func (e *NoScheduledServiceError) Error() string {
	return "service contains no active days"
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
	return []error{&NoScheduledServiceError{}}
}
