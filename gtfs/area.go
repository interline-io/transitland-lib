package gtfs

import (
	"github.com/interline-io/transitland-lib/tt"
)

// Area fare_areas.txt
type Area struct {
	AreaID    tt.String `csv:",required" standardized_sort:"1"`
	AreaName  tt.String
	AgencyIDs tt.Strings `csv:"-" db:"agency_ids"` // interline ext
	Geometry  tt.Polygon `csv:"-"`                 // interline ext
	tt.BaseEntity
}

func (ent *Area) EntityKey() string {
	return ent.AreaID.Val
}

func (ent *Area) EntityID() string {
	return ent.AreaID.Val
}

func (ent *Area) Filename() string {
	return "areas.txt"
}

func (ent *Area) TableName() string {
	return "gtfs_areas"
}
