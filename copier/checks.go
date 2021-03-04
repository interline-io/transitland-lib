package copier

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

// Validator provides
type Validator interface {
	ValidateEntity(*Copier, tl.Entity) ([]error, []error)
}

// Basic validators

type EntityErrorCheck struct{}

func (e *EntityErrorCheck) ValidateEntity(c *Copier, ent tl.Entity) ([]error, []error) {
	return ent.Errors(), ent.Warnings()
}

type EntityReferenceCheck struct{}

func (e *EntityReferenceCheck) ValidateEntity(c *Copier, ent tl.Entity) ([]error, []error) {
	a := ent.UpdateKeys(c.EntityMap)
	if a != nil {
		return []error{a}, nil
	}
	return nil, nil
}

type EntityDuplicateCheck struct{}

func (e *EntityDuplicateCheck) ValidateEntity(c *Copier, ent tl.Entity) ([]error, []error) {
	var errs []error
	// Check for duplicate entities.
	efn := ent.Filename()
	eid := ent.EntityID()
	if _, ok := c.duplicateMap.Get(efn, eid); ok && len(eid) > 0 {
		errs = append(errs, causes.NewDuplicateIDError(eid))
	} else {
		c.duplicateMap.Set(efn, eid, eid)
	}
	return errs, nil
}
