package extract

import (
	"fmt"

	"github.com/interline-io/gotransit"
)

type setterFilter struct {
	nodes map[node]map[string]string
}

func newSetterFilter() *setterFilter {
	return &setterFilter{
		nodes: map[node]map[string]string{},
	}
}

type canSetString interface {
	SetString(string, string) error
}

func (tx *setterFilter) Filter(ent gotransit.Entity, emap *gotransit.EntityMap) error {
	ent2, ok := ent.(canSetString)
	if !ok {
		return nil
	}
	if entv, ok := tx.nodes[*entityNode(ent)]; ok {
		fmt.Println("found: ", entv)
		for k, v := range entv {
			fmt.Println(k, v)
			ent2.SetString(k, v)
		}
	}
	return nil
}
