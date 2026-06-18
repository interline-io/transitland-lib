package bestpractices

import (
	"fmt"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tt"
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
func (e *NoScheduledServiceCheck) Validate(ent tt.Entity) []error {
	v, ok := ent.(*gtfs.Calendar)
	if !ok {
		return nil
	}
	svc := service.NewService(*v, v.CalendarDates...)
	if svc.HasAtLeastOneDay() {
		return nil
	}
	return []error{&NoScheduledServiceError{ServiceID: v.ServiceID.Val}}
}
