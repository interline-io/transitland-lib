package redate

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
)

type ServiceMerge struct {
	services map[string]*tl.Service
}

func (tf *ServiceMerge) Filter(ent tl.Entity, emap *tl.EntityMap) error {
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
	if tf.services == nil {
		tf.services = map[string]*tl.Service{}
	}
	tf.services[ent.EntityID()] = svc
	return nil
}
