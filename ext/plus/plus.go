package plus

import (
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
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
func (ext Ext) Entities() []tt.Entity {
	return []tt.Entity{
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

func (ext *Ext) Filter(ent tt.Entity, _ *tt.EntityMap) error {
	switch v := ent.(type) {
	case *gtfs.Agency:
		ext.defaultAgency = v.AgencyID.Val
	case *PlusRiderCategory:
		if v.AgencyID == "" {
			v.AgencyID = ext.defaultAgency
		}
	}
	return nil
}

// Copy uses the Copier to copy Entities.
func (ext *Ext) Copy(c adapters.EntityCopier) error {
	if err := copyCalendarAttributes(c); err != nil {
		return err
	}
	if err := copyRiderCategories(c); err != nil {
		return err
	}
	if err := copyFareRiderCategories(c); err != nil {
		return err
	}
	if err := copyFarezoneAttributes(c); err != nil {
		return err
	}
	if err := copyDirections(c); err != nil {
		return err
	}
	if err := copyRealtimeRoutes(c); err != nil {
		return err
	}
	if err := copyRealtimeStops(c); err != nil {
		return err
	}
	if err := copyStopAttributes(c); err != nil {
		return err
	}
	if err := copyTimepoint(c); err != nil {
		return err
	}
	if err := copyRouteAttributes(c); err != nil {
		return err
	}
	return nil
}

func copyCalendarAttributes(copier adapters.EntityCopier) error {
	out := make(chan CalendarAttribute, 1000)
	copier.Reader().ReadEntities(out)
	for ent := range out {
		if err := copier.CopyEntity(&ent); err != nil {
			return err
		}
	}
	return nil
}

func copyRiderCategories(copier adapters.EntityCopier) error {
	out := make(chan PlusRiderCategory, 1000)
	copier.Reader().ReadEntities(out)
	for ent := range out {
		if err := copier.CopyEntity(&ent); err != nil {
			return err
		}
	}
	return nil
}

func copyFareRiderCategories(copier adapters.EntityCopier) error {
	out := make(chan FareRiderCategory, 1000)
	copier.Reader().ReadEntities(out)
	for ent := range out {
		if err := copier.CopyEntity(&ent); err != nil {
			return err
		}
	}
	return nil
}

func copyFarezoneAttributes(copier adapters.EntityCopier) error {
	out := make(chan FarezoneAttribute, 1000)
	copier.Reader().ReadEntities(out)
	for ent := range out {
		if err := copier.CopyEntity(&ent); err != nil {
			return err
		}
	}
	return nil
}

func copyDirections(copier adapters.EntityCopier) error {
	out := make(chan Direction, 1000)
	copier.Reader().ReadEntities(out)
	for ent := range out {
		if err := copier.CopyEntity(&ent); err != nil {
			return err
		}
	}
	return nil
}

func copyRealtimeRoutes(copier adapters.EntityCopier) error {
	out := make(chan RealtimeRoute, 1000)
	copier.Reader().ReadEntities(out)
	for ent := range out {
		if err := copier.CopyEntity(&ent); err != nil {
			return err
		}
	}
	return nil
}

func copyRealtimeStops(copier adapters.EntityCopier) error {
	out := make(chan RealtimeStop, 1000)
	copier.Reader().ReadEntities(out)
	for ent := range out {
		if err := copier.CopyEntity(&ent); err != nil {
			return err
		}
	}
	return nil
}

func copyStopAttributes(copier adapters.EntityCopier) error {
	out := make(chan StopAttribute, 1000)
	copier.Reader().ReadEntities(out)
	for ent := range out {
		if err := copier.CopyEntity(&ent); err != nil {
			return err
		}
	}
	return nil
}

func copyTimepoint(copier adapters.EntityCopier) error {
	out := make(chan Timepoint, 1000)
	copier.Reader().ReadEntities(out)
	for ent := range out {
		if err := copier.CopyEntity(&ent); err != nil {
			return err
		}
	}
	return nil
}

func copyRouteAttributes(copier adapters.EntityCopier) error {
	out := make(chan RouteAttribute, 1000)
	copier.Reader().ReadEntities(out)
	for ent := range out {
		if err := copier.CopyEntity(&ent); err != nil {
			return err
		}
	}
	return nil
}
