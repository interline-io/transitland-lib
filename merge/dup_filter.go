package merge

import (
	"errors"

	"github.com/interline-io/transitland-lib/tl"
)

type hasEntityKey interface {
	EntityKey() string
}

type DupFilter struct {
	duplicates *tl.EntityMap
}

func (e *DupFilter) Filter(ent tl.Entity) error {
	if e.duplicates == nil {
		e.duplicates = tl.NewEntityMap()
	}
	v, ok := ent.(hasEntityKey)
	if !ok {
		return nil
	}
	eid := v.EntityKey()
	if eid == "" {
		return nil
	}
	efn := ent.Filename()
	if _, ok := e.duplicates.Get(efn, eid); ok {
		return errors.New("duplicate")
	}
	return nil
}
