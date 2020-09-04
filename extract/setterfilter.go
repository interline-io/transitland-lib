package extract

import (
	tl "github.com/interline-io/transitland-lib"
	"github.com/interline-io/transitland-lib/gtcsv"
	"github.com/interline-io/transitland-lib/internal/graph"
)

type setterFilter struct {
	nodes map[graph.Node]map[string]string
}

func NewSetterFilter() *setterFilter {
	return &setterFilter{
		nodes: map[graph.Node]map[string]string{},
	}
}

func (tx *setterFilter) AddValue(filename string, eid string, key string, value string) {
	n := graph.NewNode(filename, eid)
	entv, ok := tx.nodes[*n]
	if !ok {
		entv = map[string]string{}
	}
	entv[key] = value
	tx.nodes[*n] = entv
}

func (tx *setterFilter) Filter(ent tl.Entity, emap *tl.EntityMap) error {
	if entv, ok := tx.nodes[*graph.NewNode(ent.Filename(), ent.EntityID())]; ok {
		for k, v := range entv {
			if err := gtcsv.SetString(ent, k, v); err != nil {
				return err
			}
		}
	}
	return nil
}
