package gtfs

import (
	"github.com/interline-io/transitland-lib/tt"
)

// LocationGroup location_groups.txt
type LocationGroup struct {
	LocationGroupID   tt.String `csv:",required" standardized_sort:"1"`
	LocationGroupName tt.String
	tt.BaseEntity
}

func (ent *LocationGroup) EntityKey() string {
	return ent.LocationGroupID.Val
}

func (ent *LocationGroup) EntityID() string {
	return entID(ent.ID, ent.LocationGroupID.Val)
}

func (ent *LocationGroup) Filename() string {
	return "location_groups.txt"
}

func (ent *LocationGroup) TableName() string {
	return "gtfs_location_groups"
}
