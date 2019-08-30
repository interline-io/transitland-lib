package pathways

import (
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/copier"
)

func init() {
	ext := func() gotransit.Extension { return Ext{} }
	gotransit.RegisterExtension("pathways", ext)
}

// Ext defines the Pathways Extension.
type Ext struct {
}

// Create the tables.
func (ext Ext) Create(writer gotransit.Writer) error {
	return nil
}

// Entities returns the Extension Entity types.
func (ext Ext) Entities() []gotransit.Entity {
	return []gotransit.Entity{
		&Level{},
		&Pathway{},
	}
}

// Copy the Extension Entities with the provided Copier.
func (ext Ext) Copy(copier *copier.Copier) error {
	out := make(chan Pathway, 1000)
	copier.Reader.ReadEntities(out)
	for ent := range out {
		if err := ent.UpdateKeys(copier.EntityMap); err != nil {
			continue
		}
		if _, err := copier.Writer.AddEntity(&ent); err != nil {
			panic(err)
		}
	}
	////////
	out2 := make(chan Level, 1000)
	copier.Reader.ReadEntities(out2)
	for ent := range out2 {
		if err := ent.UpdateKeys(copier.EntityMap); err != nil {
			continue
		}
		if _, err := copier.Writer.AddEntity(&ent); err != nil {
			panic(err)
		}
	}
	return nil
}
