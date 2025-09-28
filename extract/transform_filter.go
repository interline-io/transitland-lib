package extract

import (
	"net/url"
	"os"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/graph"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tt"
)

var transformFunctions = map[string]func(string) (string, error){
	"uppercase": func(s string) (string, error) {
		return strings.ToUpper(s), nil
	},
	"lowercase": func(s string) (string, error) {
		return strings.ToLower(s), nil
	},
	"trim": func(s string) (string, error) {
		return strings.TrimSpace(s), nil
	},
	"urlescape": func(s string) (string, error) {
		return url.QueryEscape(s), nil
	},
	"replace_spaces_with_underscores": func(s string) (string, error) {
		return strings.ReplaceAll(s, " ", "_"), nil
	},
	// Add more transformation functions as needed
}

type TransformFilter struct {
	nodes map[graph.Node]map[string]string
}

func NewTransformFilter() *TransformFilter {
	return &TransformFilter{
		nodes: map[graph.Node]map[string]string{},
	}
}

// AddValuesFromFile reads a CSV file and calls AddValue on each row.
func (tx *TransformFilter) AddValuesFromFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	tlcsv.ReadRows(f, func(row tlcsv.Row) {
		efn, _ := row.Get("filename")
		eid, _ := row.Get("entity_id")
		key, _ := row.Get("key")
		funcName, _ := row.Get("func_name")
		tx.AddValue(efn, eid, key, funcName)
	})
	return nil

}

// AddValue sets a new value to transform.
func (tx *TransformFilter) AddValue(filename string, eid string, key string, funcName string) {
	// Check if funcName exists
	_, ok := transformFunctions[funcName]
	if !ok {
		log.Error().Msgf("Transform function '%s' not found", funcName)
		return
	}
	n := graph.NewNode(filename, eid)
	entv, ok := tx.nodes[*n]
	if !ok {
		entv = map[string]string{}
	}
	entv[key] = funcName
	tx.nodes[*n] = entv
}

func (tx *TransformFilter) Filter(ent tt.Entity, emap *tt.EntityMap) error {
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
	for transformKey, funcName := range entv {
		transformFunc, ok := transformFunctions[funcName]
		if !ok {
			log.Error().Msgf("Transform function '%s' not found", funcName)
			continue // Skip if function not found
		}
		// FIXME: This is not really a public API
		header, err := tlcsv.MapperCache.GetHeader(ent.Filename())
		if err != nil {
			log.Error().Msgf("Failed to get header for '%s': %v", ent.Filename(), err)
			continue
		}
		for _, headerKey := range header {
			if !(transformKey == "*" || headerKey == transformKey) {
				continue
			}
			originalValue, err := tlcsv.GetString(ent, headerKey)
			if err != nil {
				log.Error().Msgf("Failed to get field '%s': %v", headerKey, err)
				continue
			}
			newValue, err := transformFunc(originalValue)
			if err != nil {
				log.Error().Msgf("Transform function '%s' failed on field '%s': %v", funcName, headerKey, err)
				continue
			}
			// Set the transformed value back to the entity
			if err := tlcsv.SetString(ent, headerKey, newValue); err != nil {
				log.Error().Msgf("Failed to set field '%s': %v", headerKey, err)
				continue
			}
		}
	}
	return nil
}
