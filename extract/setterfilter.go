package extract

import (
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/gtcsv"
)

type setterFilter struct {
	nodes map[node]map[string]string
}

func newSetterFilter() *setterFilter {
	return &setterFilter{
		nodes: map[node]map[string]string{},
	}
}

func (tx *setterFilter) Filter(ent gotransit.Entity, emap *gotransit.EntityMap) error {
	if entv, ok := tx.nodes[*entityNode(ent)]; ok {
		for k, v := range entv {
			if err := gtcsv.SetString(ent, k, v); err != nil {
				return err
			}
		}
	}
	return nil
}
