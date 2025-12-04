package gtfs

import (
	"github.com/interline-io/transitland-lib/tt"
)

// LocationGroupStop location_group_stops.txt
type LocationGroupStop struct {
	LocationGroupID tt.Key `csv:",required" target:"location_groups.txt"`
	StopID          tt.Key `csv:",required" target:"stops.txt"`
	tt.BaseEntity
}

func (ent *LocationGroupStop) Filename() string {
	return "location_group_stops.txt"
}

func (ent *LocationGroupStop) TableName() string {
	return "gtfs_location_group_stops"
}

// UpdateKeys updates Entity references.
func (ent *LocationGroupStop) UpdateKeys(emap *tt.EntityMap) error {
	return tt.FirstError(
		tt.TrySetField(emap.UpdateKey(&ent.LocationGroupID, "location_groups.txt"), "location_group_id"),
		tt.TrySetField(emap.UpdateKey(&ent.StopID, "stops.txt"), "stop_id"),
	)
}
