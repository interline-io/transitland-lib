package gtfs

import (
	"github.com/interline-io/transitland-lib/tt"
)

// StopArea stop_areas.txt
type StopArea struct {
	AreaID tt.Key `target:"areas.txt"`
	StopID tt.Key `target:"stops.txt"`
	tt.BaseEntity
}

func (ent *StopArea) Filename() string {
	return "stop_areas.txt"
}

func (ent *StopArea) TableName() string {
	return "gtfs_stop_areas"
}

func (ent *StopArea) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.LoadErrors()...)
	errs = append(errs, tt.CheckPresent("area_id", ent.AreaID.Val)...)
	errs = append(errs, tt.CheckPresent("stop_id", ent.StopID.Val)...)
	return errs
}
