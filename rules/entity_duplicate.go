package rules

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

type hasEntityKey interface {
	EntityKey() string
}

// EntityDuplicateCheck determines if a unique entity ID is present more than once in the file.
type EntityDuplicateCheck struct {
	duplicates *tt.EntityMap
}

// Validate .
func (e *EntityDuplicateCheck) Validate(ent tt.Entity) []error {
	if e.duplicates == nil {
		e.duplicates = tt.NewEntityMap()
	}
	v, ok := ent.(hasEntityKey)
	if !ok {
		return nil
	}
	eid := v.EntityKey()
	if eid == "" {
		return nil
	}
	var errs []error
	efn := ent.Filename()
	if _, ok := e.duplicates.Get(efn, eid); ok {
		errs = append(errs, causes.NewDuplicateIDError(eid))
	} else {
		e.duplicates.Set(efn, eid, eid)
	}
	return errs
}
