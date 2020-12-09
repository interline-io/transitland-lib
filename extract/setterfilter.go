package extract

import (
	"os"

	"github.com/interline-io/transitland-lib/internal/graph"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlcsv"
)

// SetterFilter overrides entity values using a copier filter.
type SetterFilter struct {
	nodes map[graph.Node]map[string]string
}

// NewSetterFilter returns an initialized SetterFilter.
func NewSetterFilter() *SetterFilter {
	return &SetterFilter{
		nodes: map[graph.Node]map[string]string{},
	}
}

// AddValuesFromFile reads a CSV file and calls AddValue on each row.
func (tx *SetterFilter) AddValuesFromFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	tlcsv.ReadRows(f, func(row tlcsv.Row) {
		efn, _ := row.Get("filename")
		eid, _ := row.Get("entity_id")
		key, _ := row.Get("key")
		val, _ := row.Get("value")
		tx.AddValue(efn, eid, key, val)
	})
	return nil

}

// AddValue sets a new value to override.
func (tx *SetterFilter) AddValue(filename string, eid string, key string, value string) {
	n := graph.NewNode(filename, eid)
	entv, ok := tx.nodes[*n]
	if !ok {
		entv = map[string]string{}
	}
	entv[key] = value
	tx.nodes[*n] = entv
}

type hasEntityKey interface {
	EntityKey() string
}

// Filter overrides values on entities.
func (tx *SetterFilter) Filter(ent tl.Entity, emap *tl.EntityMap) error {
	if v, ok := ent.(hasEntityKey); ok {
		if entv, ok := tx.nodes[*graph.NewNode(ent.Filename(), v.EntityKey())]; ok {
			for k, v := range entv {
				if err := tlcsv.SetString(ent, k, v); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
