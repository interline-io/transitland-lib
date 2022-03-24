package redate

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
)

type ServiceMergeFilter struct {
	services map[string]*tl.Service
}

func NewServiceMergeFilter() (*ServiceMergeFilter, error) {
	return &ServiceMergeFilter{services: map[string]*tl.Service{}}, nil
}

func (tf *ServiceMergeFilter) Filter(ent tl.Entity, emap *tl.EntityMap) error {
	svc, ok := ent.(*tl.Service)
	if !ok {
		return nil
	}
	for k, v := range tf.services {
		if svc.Equal(v) {
			emap.Set("calendar.txt", svc.EntityID(), k)
			return fmt.Errorf("merged service '%s' with '%s'", svc.EntityID(), k)
		}
	}
	// save a copy
	tf.services[ent.EntityID()] = tl.NewService(svc.Calendar, svc.CalendarDates()...)
	return nil
}
