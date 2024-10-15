package gtfs

import (
	"github.com/interline-io/transitland-lib/tt"
)

// StopArea stop_areas.txt
type StopArea struct {
	AreaID tt.Key `csv:",required" target:"areas.txt"`
	StopID tt.Key `csv:",required" target:"stops.txt"`
	tt.BaseEntity
}

func (ent *StopArea) Filename() string {
	return "stop_areas.txt"
}

func (ent *StopArea) TableName() string {
	return "gtfs_stop_areas"
}
