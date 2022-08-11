package testutil

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
)

type canCreateFV interface {
	CreateFeedVersion(reader tl.Reader) (int, error)
}

// DirectCopy does a direct reader->writer copy, with minimal validation and changes.
func DirectCopy(reader tl.Reader, writer tl.Writer) error {
	emap := tl.NewEntityMap()
	errs := []error{}
	cp := func(ent tl.Entity) {
		// All other entities
		sid := ent.EntityID()
		if extEnt, ok := ent.(tl.EntityWithReferences); ok {
			if err := extEnt.UpdateKeys(emap); err != nil {
				errs = append(errs, fmt.Errorf("entity: %#v error: %s", ent, err))
			}
		}
		eid, err := writer.AddEntity(ent)
		if err != nil {
			errs = append(errs, fmt.Errorf("entity: %#v error: %s", ent, err))
		}
		emap.SetEntity(ent, sid, eid)
	}
	// Create any FV
	if w2, ok := writer.(canCreateFV); ok {
		w2.CreateFeedVersion(reader)
	}
	// Run callback on each entity
	AllEntities(reader, cp)
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
