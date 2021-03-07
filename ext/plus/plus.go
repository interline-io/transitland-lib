package plus

import (
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/tl"
)

func init() {
	e := func() ext.Extension { return Ext{} }
	ext.RegisterExtension("plus", e)
}

// Ext is the GTFS Plus Extension.
type Ext struct {
}

// Create the necessary database tables.
func (ext Ext) Create(writer tl.Writer) error {
	return nil
}

// Entities returns the entities defined by GTFS Plus.
func (ext Ext) Entities() []tl.Entity {
	return []tl.Entity{
		&CalendarAttribute{},
		&RiderCategory{},
		&FareRiderCategory{},
		&FarezoneAttribute{},
		&Direction{},
		&RealtimeRoute{},
		&RealtimeStop{},
		&StopAttribute{},
		&Timepoint{},
	}
}

// Copy uses the Copier to copy Entities.
func (ext Ext) Copy(c *copier.Copier) error {
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
	out := make(chan RiderCategory, 1000)
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
