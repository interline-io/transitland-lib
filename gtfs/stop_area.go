package gtfs

import (
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

// StopArea stop_areas.txt
type StopArea struct {
	AreaID string
	StopID string
	tt.BaseEntity
}

func (ent *StopArea) Filename() string {
	return "stop_areas.txt"
}

func (ent *StopArea) TableName() string {
	return "gtfs_stop_areas"
}

func (ent *StopArea) UpdateKeys(emap *EntityMap) error {
	if fkid, ok := emap.Get("areas.txt", ent.AreaID); ok {
		ent.AreaID = fkid
	} else {
		return causes.NewInvalidReferenceError("area_id", ent.AreaID)
	}
	if fkid, ok := emap.Get("stops.txt", ent.StopID); ok {
		ent.StopID = fkid
	} else {
		return causes.NewInvalidReferenceError("stop_id", ent.StopID)
	}
	return nil
}

func (ent *StopArea) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, tt.CheckPresent("area_id", ent.AreaID)...)
	errs = append(errs, tt.CheckPresent("stop_id", ent.StopID)...)
	return errs
}
