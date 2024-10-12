package gtfs

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// StopArea stop_areas.txt
type StopArea struct {
	AreaID tt.Key
	StopID tt.Key
	tt.BaseEntity
}

func (ent *StopArea) Filename() string {
	return "stop_areas.txt"
}

func (ent *StopArea) TableName() string {
	return "gtfs_stop_areas"
}

func (ent *StopArea) UpdateKeys(emap *EntityMap) error {
	if fkid, ok := emap.Get("areas.txt", ent.AreaID.Val); ok {
		ent.AreaID.Set(fkid)
	} else {
		return causes.NewInvalidReferenceError("area_id", ent.AreaID.Val)
	}
	if fkid, ok := emap.Get("stops.txt", ent.StopID.Val); ok {
		ent.StopID.Set(fkid)
	} else {
		return causes.NewInvalidReferenceError("stop_id", ent.StopID.Val)
	}
	return nil
}

func (ent *StopArea) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, tt.CheckPresent("area_id", ent.AreaID.Val)...)
	errs = append(errs, tt.CheckPresent("stop_id", ent.StopID.Val)...)
	return errs
}
