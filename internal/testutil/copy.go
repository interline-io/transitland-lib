package testutil

import (
	"fmt"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/tt"
)

type canCreateFV interface {
	CreateFeedVersion(reader adapters.Reader) (int, error)
}

type Filter interface {
	Filter(tt.Entity, *tt.EntityMap) error
}

type DirectCopierOptions struct{}

type DirectCopier struct {
	reader  adapters.Reader
	writer  adapters.Writer
	opts    DirectCopierOptions
	filters []Filter
}

func NewDirectCopier(reader adapters.Reader, writer adapters.Writer, opts DirectCopierOptions) (*DirectCopier, error) {
	return &DirectCopier{
		reader: reader,
		writer: writer,
		opts:   opts,
	}, nil
}

func (dc *DirectCopier) AddFilter(f Filter) error {
	dc.filters = append(dc.filters, f)
	return nil
}

func (dc *DirectCopier) Copy() error {
	emap := tt.NewEntityMap()
	var errs []error
	cp := func(ent tt.Entity) {
		sid := ent.EntityID()
		for _, filter := range dc.filters {
			if err := filter.Filter(ent, emap); err != nil {
				// these are not real errors, just a marker to skip entity
				// errs = append(errs, err)
			}
		}
		if extEnt, ok := ent.(tt.EntityWithReferences); ok {
			if err := extEnt.UpdateKeys(emap); err != nil {
				errs = append(errs, fmt.Errorf("entity: %#v error: %s", ent, err))
			}
		}
		eid, err := dc.writer.AddEntity(ent)
		if err != nil {
			errs = append(errs, fmt.Errorf("entity: %#v error: %s", ent, err))
		}
		emap.SetEntity(ent, sid, eid)
	}
	// Create any FV
	if w2, ok := dc.writer.(canCreateFV); ok {
		w2.CreateFeedVersion(dc.reader)
	}
	// Run callback on each entity
	AllEntities(dc.reader, cp)
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// DirectCopy does a direct reader->writer copy, with minimal validation and changes.
func DirectCopy(reader adapters.Reader, writer adapters.Writer) error {
	cp, err := NewDirectCopier(reader, writer, DirectCopierOptions{})
	if err != nil {
		return err
	}
	return cp.Copy()
}
