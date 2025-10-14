package extract

import (
	"os"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/graph"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tt"
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
func (tx *SetterFilter) Filter(ent tt.Entity, emap *tt.EntityMap) error {
	v, ok := ent.(hasEntityKey)
	if !ok {
		return nil
	}
	entv, ok := tx.nodes[*graph.NewNode(ent.Filename(), v.EntityKey())]
	if !ok {
		// Check for filters that apply to all entities
		entv, ok = tx.nodes[*graph.NewNode(ent.Filename(), "*")]
	}
	if !ok {
		return nil
	}
	for setterKey, newValue := range entv {
		if err := tlcsv.SetString(ent, setterKey, newValue); err != nil {
			log.Error().Msgf("Failed to set field '%s': %v", setterKey, err)
			continue
		}
		// Handle "*"
		if entv, ok := tx.nodes[*graph.NewNode(ent.Filename(), "*")]; ok {
			for k, v := range entv {
				if err := tlcsv.SetString(ent, k, v); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
