package rules

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

type hasEntityKey interface {
	EntityKey() string
}

// EntityDuplicateCheck determines if a unique entity ID is present more than once in the file.
type EntityDuplicateIDCheck struct {
	duplicates *tt.EntityMap
}

// Validate .
func (e *EntityDuplicateIDCheck) Validate(ent tt.Entity) []error {
	if e.duplicates == nil {
		e.duplicates = tt.NewEntityMap()
	}
	eid := ""
	if v, ok := ent.(hasEntityKey); ok {
		eid = v.EntityKey()
	} else {
		return nil
	}
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

/////////

type hasDuplicateKey interface {
	DuplicateKey() string
}

type EntityDuplicateKeyCheck struct {
	duplicates *tt.EntityMap
}

// Validate .
func (e *EntityDuplicateKeyCheck) Validate(ent tt.Entity) []error {
	if e.duplicates == nil {
		e.duplicates = tt.NewEntityMap()
	}
	eid := ""
	if v, ok := ent.(hasDuplicateKey); ok {
		eid = v.DuplicateKey()
	} else {
		return nil
	}
	if eid == "" {
		return nil
	}
	var errs []error
	efn := ent.Filename()
	if _, ok := e.duplicates.Get(efn, eid); ok {
		errs = append(errs, causes.NewDuplicateKeyError(eid))
	} else {
		e.duplicates.Set(efn, eid, eid)
	}
	return errs
}
