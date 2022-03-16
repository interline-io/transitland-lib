package plus

import (
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/tl"
)

func init() {
	e := func(string) (ext.Extension, error) { return &Ext{}, nil }
	ext.RegisterExtension("plus", e)
}

// Ext is the GTFS Plus Extension.
type Ext struct {
	defaultAgency string
}

// Entities returns the entities defined by GTFS Plus.
func (ext Ext) Entities() []tl.Entity {
	return []tl.Entity{
		&CalendarAttribute{},
		&PlusRiderCategory{},
		&FareRiderCategory{},
		&FarezoneAttribute{},
		&Direction{},
		&RealtimeRoute{},
		&RealtimeStop{},
		&StopAttribute{},
		&Timepoint{},
	}
}

func (ext *Ext) Filter(ent tl.Entity, emap *tl.EntityMap) error {
	switch v := ent.(type) {
	case *tl.Agency:
		ext.defaultAgency = v.AgencyID
	case *PlusRiderCategory:
		if v.AgencyID == "" {
			v.AgencyID = ext.defaultAgency
		}
	}
	return nil
}

// Copy uses the Copier to copy Entities.
func (ext *Ext) Copy(c *copier.Copier) error {
	copyCalendarAttributes(c)
	copyRiderCategories(c)
	copyFareRiderCategories(c)
	copyFarezoneAttributes(c)
	copyDirections(c)
	copyRealtimeRoutes(c)
	copyRealtimeStops(c)
	copyStopAttributes(c)
	copyTimepoint(c)
	return nil
}

func copyCalendarAttributes(copier *copier.Copier) {
	out := make(chan CalendarAttribute, 1000)
	copier.Reader.ReadEntities(out)
	for ent := range out {
		copier.CopyEntity(&ent)
	}
}

func copyRiderCategories(copier *copier.Copier) {
	out := make(chan PlusRiderCategory, 1000)
	copier.Reader.ReadEntities(out)
	for ent := range out {
		copier.CopyEntity(&ent)
	}
}

func copyFareRiderCategories(copier *copier.Copier) {
	out := make(chan FareRiderCategory, 1000)
	copier.Reader.ReadEntities(out)
	for ent := range out {
		copier.CopyEntity(&ent)
	}
}

func copyFarezoneAttributes(copier *copier.Copier) {
	out := make(chan FarezoneAttribute, 1000)
	copier.Reader.ReadEntities(out)
	for ent := range out {
		copier.CopyEntity(&ent)
	}
}

func copyDirections(copier *copier.Copier) {
	out := make(chan Direction, 1000)
	copier.Reader.ReadEntities(out)
	for ent := range out {
		copier.CopyEntity(&ent)
	}
}

func copyRealtimeRoutes(copier *copier.Copier) {
	out := make(chan RealtimeRoute, 1000)
	copier.Reader.ReadEntities(out)
	for ent := range out {
		copier.CopyEntity(&ent)
	}
}

func copyRealtimeStops(copier *copier.Copier) {
	out := make(chan RealtimeStop, 1000)
	copier.Reader.ReadEntities(out)
	for ent := range out {
		copier.CopyEntity(&ent)
	}
}

func copyStopAttributes(copier *copier.Copier) {
	out := make(chan StopAttribute, 1000)
	copier.Reader.ReadEntities(out)
	for ent := range out {
		copier.CopyEntity(&ent)
	}
}

func copyTimepoint(copier *copier.Copier) {
	out := make(chan Timepoint, 1000)
	copier.Reader.ReadEntities(out)
	for ent := range out {
		copier.CopyEntity(&ent)
	}
}
