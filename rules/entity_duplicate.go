package rules

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

// EntityDuplicateCheck determines if a unique entity ID is present more than once in the file.
type EntityDuplicateCheck struct {
	duplicates *tl.EntityMap
}

// Validate .
func (e *EntityDuplicateCheck) Validate(ent tl.Entity) []error {
	if e.duplicates == nil {
		e.duplicates = tl.NewEntityMap()
	}
	eid := ent.EntityID()
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
