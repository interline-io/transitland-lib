package plus

import (
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/copier"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/log"
)

func init() {
	ext := func() gotransit.Extension { return Ext{} }
	gotransit.RegisterExtension("plus", ext)
}

// Ext is the GTFS Plus Extension.
type Ext struct {
}

// Create the necessary database tables.
func (ext Ext) Create(writer gotransit.Writer) error {
	w, ok := writer.(*gtdb.Writer)
	if !ok {
		return nil
	}
	db := w.Adapter.DB()
	for _, e := range ext.Entities() {
		db.AutoMigrate(e)
	}
	return nil

}

// Entities returns the Entities defined by GTFS Plus.
func (ext Ext) Entities() []gotransit.Entity {
	return []gotransit.Entity{
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
		if err := ent.UpdateKeys(copier.EntityMap); err != nil {
			continue
		}
		if _, err := copier.Writer.AddEntity(&ent); err != nil {
			panic(err)
		}
	}
}

func copyRiderCategories(copier *copier.Copier) {
	out := make(chan RiderCategory, 1000)
	copier.Reader.ReadEntities(out)
	for ent := range out {
		if len(ent.AgencyID) == 0 {
			ent.AgencyID = copier.DefaultAgencyID
		}
		if err := ent.UpdateKeys(copier.EntityMap); err != nil {
			log.Debug("%s '%s' failed to update keys: %s", ent.Filename(), ent.EntityID(), err)
			continue
		}
		if _, err := copier.Writer.AddEntity(&ent); err != nil {
			panic(err)
		}
	}
}

func copyFareRiderCategories(copier *copier.Copier) {
	out := make(chan FareRiderCategory, 1000)
	copier.Reader.ReadEntities(out)
	for ent := range out {
		if err := ent.UpdateKeys(copier.EntityMap); err != nil {
			log.Debug("%s '%s' failed to update keys: %s", ent.Filename(), ent.EntityID(), err)
			continue
		}
		if _, err := copier.Writer.AddEntity(&ent); err != nil {
			panic(err)
		}
	}
}

func copyFarezoneAttributes(copier *copier.Copier) {
	out := make(chan FarezoneAttribute, 1000)
	copier.Reader.ReadEntities(out)
	for ent := range out {
		if err := ent.UpdateKeys(copier.EntityMap); err != nil {
			log.Debug("%s '%s' failed to update keys: %s", ent.Filename(), ent.EntityID(), err)
			continue
		}
		if _, err := copier.Writer.AddEntity(&ent); err != nil {
			panic(err)
		}
	}
}

func copyDirections(copier *copier.Copier) {
	out := make(chan Direction, 1000)
	copier.Reader.ReadEntities(out)
	for ent := range out {
		if err := ent.UpdateKeys(copier.EntityMap); err != nil {
			log.Debug("%s '%s' failed to update keys: %s", ent.Filename(), ent.EntityID(), err)
			continue
		}
		if _, err := copier.Writer.AddEntity(&ent); err != nil {
			panic(err)
		}
	}
}

func copyRealtimeRoutes(copier *copier.Copier) {
	out := make(chan RealtimeRoute, 1000)
	copier.Reader.ReadEntities(out)
	for ent := range out {
		if err := ent.UpdateKeys(copier.EntityMap); err != nil {
			log.Debug("%s '%s' failed to update keys: %s", ent.Filename(), ent.EntityID(), err)
			continue
		}
		if _, err := copier.Writer.AddEntity(&ent); err != nil {
			panic(err)
		}
	}
}

func copyRealtimeStops(copier *copier.Copier) {
	out := make(chan RealtimeStop, 1000)
	copier.Reader.ReadEntities(out)
	for ent := range out {
		if err := ent.UpdateKeys(copier.EntityMap); err != nil {
			log.Debug("%s '%s' failed to update keys: %s", ent.Filename(), ent.EntityID(), err)
			continue
		}
		if _, err := copier.Writer.AddEntity(&ent); err != nil {
			panic(err)
		}
	}
}

func copyStopAttributes(copier *copier.Copier) {
	out := make(chan StopAttribute, 1000)
	copier.Reader.ReadEntities(out)
	for ent := range out {
		if err := ent.UpdateKeys(copier.EntityMap); err != nil {
			continue
		}
		if _, err := copier.Writer.AddEntity(&ent); err != nil {
			log.Debug("%s '%s' failed to update keys: %s", ent.Filename(), ent.EntityID(), err)
			panic(err)
		}
	}
}

func copyTimepoint(copier *copier.Copier) {
	out := make(chan Timepoint, 1000)
	copier.Reader.ReadEntities(out)
	for ent := range out {
		if err := ent.UpdateKeys(copier.EntityMap); err != nil {
			log.Debug("%s '%s' failed to update keys: %s", ent.Filename(), ent.EntityID(), err)
			continue
		}
		if _, err := copier.Writer.AddEntity(&ent); err != nil {
			panic(err)
		}
	}
}
