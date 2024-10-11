package gtfs

import (
	"github.com/interline-io/transitland-lib/tl/tt"
)

// Area fare_areas.txt
type Area struct {
	AreaID    tt.String
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

func (ent *Area) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, tt.CheckPresent("area_id", ent.AreaID.Val)...)
	return errs
}
