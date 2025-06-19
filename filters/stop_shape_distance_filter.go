package filters

import "github.com/interline-io/transitland-lib/tt"

type StopShapeDistanceFilter struct {
	Distance float64
}

func (f *StopShapeDistanceFilter) Filter(ent tt.Entity, emap *tt.EntityMap) error {

}
