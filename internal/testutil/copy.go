package testutil

import (
	"fmt"

	"github.com/interline-io/gotransit"
)

type canCreateFV interface {
	CreateFeedVersion(reader gotransit.Reader) (int, error)
}

// DirectCopy does a direct reader->writer copy, with minimal validation and changes.
func DirectCopy(reader gotransit.Reader, writer gotransit.Writer) error {
	emap := gotransit.NewEntityMap()
	errs := []error{}
	sts := []gotransit.Entity{}
	cp := func(ent gotransit.Entity) {
		if e1, ok := ent.(*gotransit.StopTime); ok {
			e2 := *e1 // dereference
			if err := e2.UpdateKeys(emap); err != nil {
				errs = append(errs, err)
			}
			sts = append(sts, &e2)
			if len(sts) > 1000 {
				if err := writer.AddEntities(sts); err != nil {
					errs = append(errs)
				}
				sts = nil
			}
			return
		}
		// All other entities
		sid := ent.EntityID()
		if err := ent.UpdateKeys(emap); err != nil {
			errs = append(errs, fmt.Errorf("entity: %#v error: %s", ent, err))
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
	// Write trailing StopTimes
	if err := writer.AddEntities(sts); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
